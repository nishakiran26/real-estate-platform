import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ROLES } from '@/utils/constants'

vi.mock('@/services/api', () => ({
  authApi: { me: vi.fn(), refresh: vi.fn(), logout: vi.fn() },
}))
vi.mock('@/services/http', () => ({ configureAuth: vi.fn() }))

const blank = { template: '<div />' }

/**
 * The guard logic under test is the same one in src/router/index.js, mounted on
 * stub components so the test exercises access control rather than views.
 */
function buildRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'home', component: blank },
      { path: '/login', name: 'login', component: blank, meta: { guestOnly: true } },
      {
        path: '/account',
        name: 'customer-dashboard',
        component: blank,
        meta: { requiresAuth: true, roles: [ROLES.CUSTOMER, ROLES.ADMIN] },
      },
      {
        path: '/client',
        name: 'client-dashboard',
        component: blank,
        meta: { requiresAuth: true, roles: [ROLES.CLIENT, ROLES.ADMIN] },
      },
      {
        path: '/admin',
        name: 'admin-dashboard',
        component: blank,
        meta: { requiresAuth: true, roles: [ROLES.ADMIN] },
      },
    ],
  })

  router.beforeEach(async (to) => {
    const auth = useAuthStore()
    auth.install()

    if (auth.accessToken && !auth.user) {
      await auth.loadProfile()
    }

    const requiresAuth = to.matched.some((record) => record.meta.requiresAuth)
    if (requiresAuth && !auth.isAuthenticated) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }

    const allowed = to.matched.reduce(
      (acc, record) => (record.meta.roles ? record.meta.roles : acc),
      null,
    )
    if (allowed && !allowed.some((role) => auth.roles.includes(role))) {
      return auth.homeRoute
    }

    if (to.meta.guestOnly && auth.isAuthenticated) {
      return auth.homeRoute
    }
    return true
  })

  return router
}

function signIn(roles) {
  const auth = useAuthStore()
  auth.persist(
    { access_token: 'token', refresh_token: 'refresh', expires_in: 900 },
    {
      id: 1,
      full_name: 'Test User',
      email: 'test@example.com',
      email_verified: true,
      mobile_verified: true,
      roles: roles.map((code, index) => ({ id: index + 1, code, name: code })),
    },
  )
  return auth
}

describe('router guards', () => {
  let router

  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    router = buildRouter()
  })

  it('sends an anonymous visitor to login and remembers where they were going', async () => {
    await router.push('/account')
    expect(router.currentRoute.value.name).toBe('login')
    expect(router.currentRoute.value.query.redirect).toBe('/account')
  })

  it('lets a customer into their own account area', async () => {
    signIn([ROLES.CUSTOMER])
    await router.push('/account')
    expect(router.currentRoute.value.name).toBe('customer-dashboard')
  })

  it('keeps a customer out of the admin console', async () => {
    signIn([ROLES.CUSTOMER])
    await router.push('/admin')
    expect(router.currentRoute.value.name).toBe('customer-dashboard')
  })

  it('keeps a customer out of the builder portal', async () => {
    signIn([ROLES.CUSTOMER])
    await router.push('/client')
    expect(router.currentRoute.value.name).toBe('customer-dashboard')
  })

  it('keeps a builder out of the admin console', async () => {
    signIn([ROLES.CLIENT])
    await router.push('/admin')
    expect(router.currentRoute.value.name).toBe('client-dashboard')
  })

  it('lets an administrator reach every console', async () => {
    signIn([ROLES.ADMIN])

    await router.push('/admin')
    expect(router.currentRoute.value.name).toBe('admin-dashboard')

    await router.push('/client')
    expect(router.currentRoute.value.name).toBe('client-dashboard')

    await router.push('/account')
    expect(router.currentRoute.value.name).toBe('customer-dashboard')
  })

  it('redirects a signed-in user away from the login page', async () => {
    signIn([ROLES.CLIENT])
    await router.push('/login')
    expect(router.currentRoute.value.name).toBe('client-dashboard')
  })

  it('allows anonymous access to public routes', async () => {
    await router.push('/')
    expect(router.currentRoute.value.name).toBe('home')
  })
})
