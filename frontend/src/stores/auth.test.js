import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from './auth'

vi.mock('@/services/api', () => ({
  authApi: {
    login: vi.fn(),
    register: vi.fn(),
    refresh: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
  },
}))

vi.mock('@/services/http', () => ({ configureAuth: vi.fn() }))

import { authApi } from '@/services/api'

const session = (roles, extra = {}) => ({
  tokens: { access_token: 'access-token', refresh_token: 'refresh-token', expires_in: 900 },
  user: {
    id: 1,
    full_name: 'Priya Sharma',
    email: 'priya@example.com',
    mobile: '9876543210',
    email_verified: true,
    mobile_verified: true,
    roles: roles.map((code, index) => ({ id: index + 1, code, name: code })),
    ...extra,
  },
})

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('starts signed out', () => {
    const auth = useAuthStore()
    expect(auth.isAuthenticated).toBe(false)
    expect(auth.roles).toEqual([])
  })

  it('stores tokens and the user after a successful login', async () => {
    authApi.login.mockResolvedValue({ data: session(['CUSTOMER']) })

    const auth = useAuthStore()
    await auth.login({ email: 'priya@example.com', password: 'Password@123' })

    expect(auth.isAuthenticated).toBe(true)
    expect(auth.accessToken).toBe('access-token')
    expect(auth.displayName).toBe('Priya Sharma')
    // Tokens survive a page reload.
    expect(localStorage.getItem('rera_access_token')).toBe('access-token')
    expect(localStorage.getItem('rera_refresh_token')).toBe('refresh-token')
  })

  it('derives role flags from the user payload', async () => {
    authApi.login.mockResolvedValue({ data: session(['ADMIN']) })

    const auth = useAuthStore()
    await auth.login({ email: 'admin@example.com', password: 'x' })

    expect(auth.isAdmin).toBe(true)
    expect(auth.isClient).toBe(false)
    expect(auth.isCustomer).toBe(false)
    expect(auth.homeRoute).toEqual({ name: 'admin-dashboard' })
  })

  it('sends a builder to the builder dashboard and a buyer to their account', async () => {
    authApi.login.mockResolvedValue({ data: session(['CLIENT'], { client_id: 7 }) })
    const builder = useAuthStore()
    await builder.login({ email: 'builder@example.com', password: 'x' })
    expect(builder.homeRoute).toEqual({ name: 'client-dashboard' })
    expect(builder.clientId).toBe(7)

    setActivePinia(createPinia())
    localStorage.clear()
    authApi.login.mockResolvedValue({ data: session(['CUSTOMER']) })
    const buyer = useAuthStore()
    await buyer.login({ email: 'buyer@example.com', password: 'x' })
    expect(buyer.homeRoute).toEqual({ name: 'customer-dashboard' })
  })

  it('clears the session on logout even when the revoke call fails', async () => {
    authApi.login.mockResolvedValue({ data: session(['CUSTOMER']) })
    authApi.logout.mockRejectedValue(new Error('network down'))

    const auth = useAuthStore()
    await auth.login({ email: 'priya@example.com', password: 'x' })
    await auth.logout()

    expect(auth.isAuthenticated).toBe(false)
    expect(localStorage.getItem('rera_access_token')).toBeNull()
  })

  it('replaces the token pair when a refresh succeeds', async () => {
    authApi.login.mockResolvedValue({ data: session(['CUSTOMER']) })
    authApi.refresh.mockResolvedValue({
      data: { access_token: 'new-access', refresh_token: 'new-refresh', expires_in: 900 },
    })

    const auth = useAuthStore()
    await auth.login({ email: 'priya@example.com', password: 'x' })

    await expect(auth.refreshSession()).resolves.toBe(true)
    expect(auth.accessToken).toBe('new-access')
    expect(auth.refreshToken).toBe('new-refresh')
  })

  it('signs the user out when the refresh token is rejected', async () => {
    authApi.login.mockResolvedValue({ data: session(['CUSTOMER']) })
    authApi.refresh.mockRejectedValue(new Error('token reuse detected'))

    const auth = useAuthStore()
    await auth.login({ email: 'priya@example.com', password: 'x' })

    await expect(auth.refreshSession()).resolves.toBe(false)
    expect(auth.isAuthenticated).toBe(false)
  })

  it('does not attempt a refresh without a refresh token', async () => {
    const auth = useAuthStore()
    await expect(auth.refreshSession()).resolves.toBe(false)
    expect(authApi.refresh).not.toHaveBeenCalled()
  })

  it('flags an account whose contact details are unverified', async () => {
    authApi.login.mockResolvedValue({
      data: session(['CUSTOMER'], { email_verified: false, mobile_verified: false }),
    })

    const auth = useAuthStore()
    await auth.login({ email: 'priya@example.com', password: 'x' })
    expect(auth.needsVerification).toBe(true)
  })
})
