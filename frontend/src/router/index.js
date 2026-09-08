import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ROLES } from '@/utils/constants'

/**
 * Every view is loaded lazily, so a visitor downloads the public bundle only.
 * Route meta drives both access control and the default page title:
 *   requiresAuth - a session is needed
 *   roles        - the roles allowed through
 *   guestOnly    - signed-in users are redirected to their dashboard
 */
const routes = [
  {
    path: '/',
    component: () => import('@/layouts/PublicLayout.vue'),
    children: [
      { path: '', name: 'home', component: () => import('@/views/public/HomeView.vue') },
      {
        path: 'properties',
        name: 'properties',
        component: () => import('@/views/public/PropertyListView.vue'),
        meta: { title: 'Properties' },
      },
      {
        path: 'properties/compare',
        name: 'compare',
        component: () => import('@/views/public/CompareView.vue'),
        meta: { title: 'Compare properties' },
      },
      {
        path: 'properties/:slug',
        name: 'property-detail',
        component: () => import('@/views/public/PropertyDetailView.vue'),
      },
      {
        path: 'city/:citySlug',
        name: 'city',
        component: () => import('@/views/public/PropertyListView.vue'),
      },
      {
        path: 'city/:citySlug/:localitySlug',
        name: 'locality',
        component: () => import('@/views/public/PropertyListView.vue'),
      },

      // The rent pages are the same list view pinned to the rental catalogue.
      // Purpose travels in route meta, so the URL alone decides what is shown
      // and the path stays clean enough to share and index.
      {
        path: 'rent',
        name: 'rent',
        component: () => import('@/views/public/PropertyListView.vue'),
        meta: { title: 'Properties for rent', purpose: 'RENT' },
      },
      {
        path: 'rent/:citySlug',
        name: 'rent-city',
        component: () => import('@/views/public/PropertyListView.vue'),
        meta: { purpose: 'RENT' },
      },
      {
        path: 'rent/:citySlug/:localitySlug',
        name: 'rent-locality',
        component: () => import('@/views/public/PropertyListView.vue'),
        meta: { purpose: 'RENT' },
      },
      {
        path: 'builders/:id',
        name: 'builder',
        component: () => import('@/views/public/BuilderView.vue'),
      },
      {
        path: 'page/:slug',
        name: 'static-page',
        component: () => import('@/views/public/StaticPageView.vue'),
      },
    ],
  },

  {
    path: '/',
    component: () => import('@/layouts/AuthLayout.vue'),
    children: [
      {
        path: 'login',
        name: 'login',
        component: () => import('@/views/auth/LoginView.vue'),
        meta: { title: 'Log in', guestOnly: true },
      },
      {
        path: 'register',
        name: 'register',
        component: () => import('@/views/auth/RegisterView.vue'),
        meta: { title: 'Create an account', guestOnly: true },
      },
      {
        path: 'forgot-password',
        name: 'forgot-password',
        component: () => import('@/views/auth/ForgotPasswordView.vue'),
        meta: { title: 'Reset your password', guestOnly: true },
      },
      {
        path: 'verify',
        name: 'verify-otp',
        component: () => import('@/views/auth/VerifyOtpView.vue'),
        meta: { title: 'Verify your account' },
      },
    ],
  },

  {
    path: '/account',
    component: () => import('@/layouts/DashboardLayout.vue'),
    meta: { requiresAuth: true, roles: [ROLES.CUSTOMER, ROLES.ADMIN] },
    children: [
      {
        path: '',
        name: 'customer-dashboard',
        component: () => import('@/views/customer/DashboardView.vue'),
        meta: { title: 'My dashboard' },
      },
      {
        path: 'profile',
        name: 'customer-profile',
        component: () => import('@/views/customer/ProfileView.vue'),
        meta: { title: 'My profile' },
      },
      {
        path: 'saved',
        name: 'customer-saved',
        component: () => import('@/views/customer/SavedView.vue'),
        meta: { title: 'Saved properties' },
      },
      {
        path: 'visits',
        name: 'customer-visits',
        component: () => import('@/views/customer/VisitsView.vue'),
        meta: { title: 'My visits' },
      },
      {
        path: 'saved-searches',
        name: 'customer-saved-searches',
        component: () => import('@/views/customer/SavedSearchesView.vue'),
        meta: { title: 'Saved searches' },
      },
      {
        path: 'recently-viewed',
        name: 'customer-recent',
        component: () => import('@/views/customer/RecentlyViewedView.vue'),
        meta: { title: 'Recently viewed' },
      },
      {
        path: 'inquiries',
        name: 'customer-inquiries',
        component: () => import('@/views/customer/InquiriesView.vue'),
        meta: { title: 'My inquiries' },
      },
      {
        path: 'preferences',
        name: 'customer-preferences',
        component: () => import('@/views/customer/PreferencesView.vue'),
        meta: { title: 'Preferences and notifications' },
      },
    ],
  },

  {
    path: '/client',
    component: () => import('@/layouts/DashboardLayout.vue'),
    meta: { requiresAuth: true, roles: [ROLES.CLIENT, ROLES.ADMIN] },
    children: [
      {
        path: '',
        name: 'client-dashboard',
        component: () => import('@/views/client/DashboardView.vue'),
        meta: { title: 'Builder dashboard' },
      },
      {
        path: 'profile',
        name: 'client-profile',
        component: () => import('@/views/client/CompanyProfileView.vue'),
        meta: { title: 'Company profile' },
      },
      {
        path: 'properties',
        name: 'client-properties',
        component: () => import('@/views/client/PropertiesView.vue'),
        meta: { title: 'My properties' },
      },
      {
        path: 'properties/new',
        name: 'client-property-new',
        component: () => import('@/views/client/PropertyFormView.vue'),
        meta: { title: 'Add a property' },
      },
      {
        path: 'properties/:id/edit',
        name: 'client-property-edit',
        component: () => import('@/views/client/PropertyFormView.vue'),
        meta: { title: 'Edit property' },
      },
      {
        path: 'properties/:id/analytics',
        name: 'client-property-analytics',
        component: () => import('@/views/client/PropertyAnalyticsView.vue'),
        meta: { title: 'Property analytics' },
      },
      {
        path: 'visits',
        name: 'client-visits',
        component: () => import('@/views/client/VisitsView.vue'),
        meta: { title: 'Visit requests' },
      },
      {
        path: 'leads',
        name: 'client-leads',
        component: () => import('@/views/client/LeadsView.vue'),
        meta: { title: 'Leads and inquiries' },
      },
      {
        path: 'settings',
        name: 'client-settings',
        component: () => import('@/views/client/SettingsView.vue'),
        meta: { title: 'Account settings' },
      },
    ],
  },

  {
    path: '/admin',
    component: () => import('@/layouts/DashboardLayout.vue'),
    meta: { requiresAuth: true, roles: [ROLES.ADMIN] },
    children: [
      {
        path: '',
        name: 'admin-dashboard',
        component: () => import('@/views/admin/DashboardView.vue'),
        meta: { title: 'Admin dashboard' },
      },
      {
        path: 'approvals',
        name: 'admin-approvals',
        component: () => import('@/views/admin/ApprovalQueueView.vue'),
        meta: { title: 'Approval queue' },
      },
      {
        path: 'properties',
        name: 'admin-properties',
        component: () => import('@/views/admin/PropertiesView.vue'),
        meta: { title: 'Property management' },
      },
      {
        path: 'properties/:id',
        name: 'admin-property-detail',
        component: () => import('@/views/admin/PropertyReviewView.vue'),
        meta: { title: 'Review property' },
      },
      {
        path: 'users',
        name: 'admin-users',
        component: () => import('@/views/admin/UsersView.vue'),
        meta: { title: 'User management' },
      },
      {
        path: 'clients',
        name: 'admin-clients',
        component: () => import('@/views/admin/ClientsView.vue'),
        meta: { title: 'Builder management' },
      },
      {
        path: 'inquiries',
        name: 'admin-inquiries',
        component: () => import('@/views/admin/InquiriesView.vue'),
        meta: { title: 'Lead management' },
      },
      {
        path: 'reports',
        name: 'admin-reports',
        component: () => import('@/views/admin/ReportsView.vue'),
        meta: { title: 'Reported listings' },
      },
      {
        path: 'locations',
        name: 'admin-locations',
        component: () => import('@/views/admin/LocationsView.vue'),
        meta: { title: 'Cities and localities' },
      },
      {
        path: 'amenities',
        name: 'admin-amenities',
        component: () => import('@/views/admin/AmenitiesView.vue'),
        meta: { title: 'Amenities' },
      },
      {
        path: 'content',
        name: 'admin-content',
        component: () => import('@/views/admin/ContentView.vue'),
        meta: { title: 'Homepage content' },
      },
      {
        path: 'analytics',
        name: 'admin-analytics',
        component: () => import('@/views/admin/AnalyticsView.vue'),
        meta: { title: 'Analytics' },
      },
      {
        path: 'audit',
        name: 'admin-audit',
        component: () => import('@/views/admin/AuditLogView.vue'),
        meta: { title: 'Audit log' },
      },
    ],
  },

  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/public/NotFoundView.vue'),
    meta: { title: 'Page not found' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, saved) {
    if (saved) return saved
    if (to.hash) return { el: to.hash, behavior: 'smooth' }
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  auth.install()

  // A stored token with no profile means a reload: fetch the profile before
  // deciding on access, otherwise a refresh would bounce the user to /login.
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

const APP_NAME = import.meta.env.VITE_APP_NAME || 'RERA Property Platform'

router.afterEach((to) => {
  // Views with richer titles (a property page) overwrite this in onMounted.
  document.title = to.meta.title ? `${to.meta.title} | ${APP_NAME}` : APP_NAME
})

export default router
