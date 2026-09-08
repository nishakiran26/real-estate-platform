import { defineStore } from 'pinia'
import { authApi } from '@/services/api'
import { configureAuth } from '@/services/http'

const ACCESS_KEY = 'rera_access_token'
const REFRESH_KEY = 'rera_refresh_token'
const USER_KEY = 'rera_user'

function readUser() {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

/**
 * Owns the session: tokens, the current user and the derived role flags the
 * router guards and navigation read.
 */
export const useAuthStore = defineStore('auth', {
  state: () => ({
    accessToken: localStorage.getItem(ACCESS_KEY) || '',
    refreshToken: localStorage.getItem(REFRESH_KEY) || '',
    user: readUser(),
    loading: false,
    initialised: false,
  }),

  getters: {
    isAuthenticated: (state) => Boolean(state.accessToken && state.user),
    roles: (state) => (state.user?.roles || []).map((role) => role.code),
    isAdmin() {
      return this.roles.includes('ADMIN')
    },
    isClient() {
      return this.roles.includes('CLIENT')
    },
    isCustomer() {
      return this.roles.includes('CUSTOMER')
    },
    clientId: (state) => state.user?.client_id || null,
    displayName: (state) => state.user?.full_name || '',
    initials: (state) =>
      (state.user?.full_name || '?')
        .split(' ')
        .filter(Boolean)
        .slice(0, 2)
        .map((part) => part[0].toUpperCase())
        .join(''),
    needsVerification: (state) =>
      Boolean(state.user) && !state.user.email_verified && !state.user.mobile_verified,
    // Where a role lands after signing in.
    homeRoute() {
      if (this.isAdmin) return { name: 'admin-dashboard' }
      if (this.isClient) return { name: 'client-dashboard' }
      return { name: 'customer-dashboard' }
    },
  },

  actions: {
    /** Wires the axios interceptors to this store exactly once. */
    install() {
      if (this.initialised) return
      configureAuth({
        getAccessToken: () => this.accessToken,
        refresh: () => this.refreshSession(),
        onUnauthorized: () => this.clear(),
      })
      this.initialised = true
    },

    persist(tokens, user) {
      if (tokens) {
        this.accessToken = tokens.access_token
        this.refreshToken = tokens.refresh_token
        localStorage.setItem(ACCESS_KEY, tokens.access_token)
        localStorage.setItem(REFRESH_KEY, tokens.refresh_token)
      }
      if (user) {
        this.user = user
        localStorage.setItem(USER_KEY, JSON.stringify(user))
      }
    },

    clear() {
      this.accessToken = ''
      this.refreshToken = ''
      this.user = null
      localStorage.removeItem(ACCESS_KEY)
      localStorage.removeItem(REFRESH_KEY)
      localStorage.removeItem(USER_KEY)
    },

    async register(payload) {
      this.loading = true
      try {
        const res = await authApi.register(payload)
        this.persist(res.data.tokens, res.data.user)
        return res.data
      } finally {
        this.loading = false
      }
    },

    async login(payload) {
      this.loading = true
      try {
        const res = await authApi.login(payload)
        this.persist(res.data.tokens, res.data.user)
        return res.data
      } finally {
        this.loading = false
      }
    },

    async loginWithOtp(payload) {
      this.loading = true
      try {
        const res = await authApi.loginWithOtp(payload)
        this.persist(res.data.tokens, res.data.user)
        return res.data
      } finally {
        this.loading = false
      }
    },

    /**
     * Exchanges the refresh token for a new pair. Returns false rather than
     * throwing so the axios interceptor can decide what to do.
     */
    async refreshSession() {
      if (!this.refreshToken) return false
      try {
        const res = await authApi.refresh(this.refreshToken)
        this.persist(res.data)
        return true
      } catch {
        this.clear()
        return false
      }
    },

    /** Re-reads the profile so role or verification changes are picked up. */
    async loadProfile() {
      if (!this.accessToken) return null
      try {
        const res = await authApi.me()
        this.persist(null, res.data)
        return res.data
      } catch {
        return null
      }
    },

    async logout(allDevices = false) {
      try {
        if (this.accessToken) {
          await authApi.logout({ refresh_token: this.refreshToken, all_devices: allDevices })
        }
      } catch {
        // A failed revoke must not trap the user in a signed-in state.
      } finally {
        this.clear()
      }
    },
  },
})
