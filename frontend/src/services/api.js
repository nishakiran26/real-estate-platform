import http from './http'

/**
 * One module per audience, mirroring the backend route groups. Views never call
 * axios directly: they call these functions, so an endpoint change lands in one
 * place.
 *
 * Every function resolves to the API envelope: { success, message, data, meta }.
 */

export const authApi = {
  register: (payload) => http.post('/auth/register', payload),
  login: (payload) => http.post('/auth/login', payload),
  loginWithOtp: (payload) => http.post('/auth/login-otp', payload),
  sendOtp: (payload) => http.post('/auth/send-otp', payload),
  resendOtp: (payload) => http.post('/auth/resend-otp', payload),
  verifyOtp: (payload) => http.post('/auth/verify-otp', payload),
  forgotPassword: (payload) => http.post('/auth/forgot-password', payload),
  resetPassword: (payload) => http.post('/auth/reset-password', payload),
  changePassword: (payload) => http.post('/auth/change-password', payload),
  refresh: (refreshToken) => http.post('/auth/refresh-token', { refresh_token: refreshToken }),
  logout: (payload) => http.post('/auth/logout', payload),
  me: () => http.get('/auth/me'),
  loginHistory: () => http.get('/auth/login-history'),
}

export const publicApi = {
  homepage: () => http.get('/homepage'),
  meta: (params) => http.get('/meta', { params }),
  properties: (params) => http.get('/properties', { params }),
  property: (key) => http.get(`/properties/${key}`),
  similar: (key, limit = 8) => http.get(`/properties/${key}/similar`, { params: { limit } }),
  compare: (ids) => http.get('/properties/compare', { params: { ids: ids.join(',') } }),
  recentlyViewed: (limit = 8) => http.get('/properties/recently-viewed', { params: { limit } }),
  recommended: (limit = 8) => http.get('/properties/recommended', { params: { limit } }),
  whatsappLink: (key, configuration) =>
    http.get(`/properties/${key}/whatsapp`, { params: { configuration } }),
  brochureUrl: (key, docId) => `${http.defaults.baseURL}/properties/${key}/documents/${docId}/download`,
  cities: (params) => http.get('/cities', { params }),
  city: (slug) => http.get(`/cities/${slug}`),
  localities: (params) => http.get('/localities', { params }),
  locality: (citySlug, slug) => http.get(`/cities/${citySlug}/localities/${slug}`),
  amenities: () => http.get('/amenities'),
  builder: (id) => http.get(`/builders/${id}`),
  pages: () => http.get('/pages'),
  page: (slug) => http.get(`/pages/${slug}`),
  createInquiry: (payload) => http.post('/inquiries', payload),

  // The property is identified by the path, so a visitor cannot book a viewing
  // of one listing while looking at another.
  requestVisit: (key, payload) => http.post(`/properties/${key}/visits`, payload),

  // Reporting is open to visitors who have not signed in: the person who
  // notices a flat is already let is the one searching for it.
  reportProperty: (key, payload) => http.post(`/properties/${key}/report`, payload),

  // What a flat here actually goes for, from live listings only.
  rentGuide: (params) => http.get('/rent-guide', { params }),
  trackEvent: (payload) => http.post('/events', payload),
}

export const customerApi = {
  dashboard: () => http.get('/customer/dashboard'),
  updateProfile: (payload) => http.put('/customer/profile', payload),
  preferences: () => http.get('/customer/preferences'),
  savePreferences: (payload) => http.put('/customer/preferences', payload),
  updateConsent: (payload) => http.put('/customer/consent', payload),
  unsubscribe: () => http.post('/customer/unsubscribe'),
  favorites: (params) => http.get('/customer/favorites', { params }),
  addFavorite: (propertyId) => http.post(`/customer/favorites/${propertyId}`),
  removeFavorite: (propertyId) => http.delete(`/customer/favorites/${propertyId}`),
  recentlyViewed: (limit = 12) => http.get('/customer/recently-viewed', { params: { limit } }),
  visits: (params) => http.get('/customer/visits', { params }),

  // Saved searches. A renter who has not found the flat yet should not have to
  // check the site every day.
  rentalAlerts: () => http.get('/customer/rental-alerts'),
  saveRentalAlert: (payload) => http.post('/customer/rental-alerts', payload),
  setRentalAlertActive: (id, isActive) =>
    http.put(`/customer/rental-alerts/${id}`, { is_active: isActive }),
  deleteRentalAlert: (id) => http.delete(`/customer/rental-alerts/${id}`),
  recommendations: (limit = 12) => http.get('/customer/recommendations', { params: { limit } }),
  inquiries: (params) => http.get('/customer/inquiries', { params }),
  notifications: (params) => http.get('/customer/notifications', { params }),
  markNotificationRead: (id) => http.put(`/customer/notifications/${id}/read`),
  markAllNotificationsRead: () => http.put('/customer/notifications/read-all'),
  notificationPreferences: () => http.get('/customer/notification-preferences'),
  saveNotificationPreferences: (payload) => http.put('/customer/notification-preferences', payload),
}

export const clientApi = {
  dashboard: (params) => http.get('/client/dashboard', { params }),
  profile: () => http.get('/client/profile'),
  updateProfile: (payload) => http.put('/client/profile', payload),
  uploadLogo: (file) => upload('/client/logo', file),

  properties: (params) => http.get('/client/properties', { params }),
  property: (id) => http.get(`/client/properties/${id}`),
  createProperty: (payload) => http.post('/client/properties', payload),
  updateProperty: (id, payload) => http.put(`/client/properties/${id}`, payload),
  deleteProperty: (id) => http.delete(`/client/properties/${id}`),
  submitProperty: (id) => http.post(`/client/properties/${id}/submit`),
  statusHistory: (id) => http.get(`/client/properties/${id}/status-history`),
  propertyAnalytics: (id) => http.get(`/client/properties/${id}/analytics`),

  media: (id) => http.get(`/client/properties/${id}/media`),
  uploadMedia: (id, file, mediaType) =>
    upload(`/client/properties/${id}/media`, file, { media_type: mediaType }),
  deleteMedia: (id, mediaId) => http.delete(`/client/properties/${id}/media/${mediaId}`),
  setCover: (id, mediaId) => http.put(`/client/properties/${id}/media/cover`, { media_id: mediaId }),
  reorderMedia: (id, mediaIds) =>
    http.put(`/client/properties/${id}/media/reorder`, { media_ids: mediaIds }),

  documents: (id) => http.get(`/client/properties/${id}/documents`),
  uploadDocument: (id, file, docType, name) =>
    upload(`/client/properties/${id}/documents`, file, { doc_type: docType, name }),
  deleteDocument: (id, docId) => http.delete(`/client/properties/${id}/documents/${docId}`),

  visits: (params) => http.get('/client/visits', { params }),
  respondToVisit: (id, payload) => http.put(`/client/visits/${id}`, payload),

  // Availability and renewal sit outside the approval workflow: letting a flat,
  // or confirming it is still free, must not send the listing back for review.
  setAvailability: (id, payload) => http.put(`/client/properties/${id}/availability`, payload),
  renewListing: (id) => http.post(`/client/properties/${id}/renew`),
  expiringListings: (withinDays = 14) =>
    http.get('/client/expiring-listings', { params: { within_days: withinDays } }),

  inquiries: (params) => http.get('/client/inquiries', { params }),
  inquiry: (id) => http.get(`/client/inquiries/${id}`),
  updateInquiryStatus: (id, payload) => http.put(`/client/inquiries/${id}/status`, payload),
}

export const adminApi = {
  dashboard: (params) => http.get('/admin/dashboard', { params }),
  analytics: (params) => http.get('/admin/analytics', { params }),
  auditLogs: (params) => http.get('/admin/audit-logs', { params }),

  properties: (params) => http.get('/admin/properties', { params }),
  property: (id) => http.get(`/admin/properties/${id}`),
  statusHistory: (id) => http.get(`/admin/properties/${id}/status-history`),
  startReview: (id) => http.put(`/admin/properties/${id}/review`),
  approveProperty: (id, payload) => http.put(`/admin/properties/${id}/approve`, payload),
  rejectProperty: (id, payload) => http.put(`/admin/properties/${id}/reject`, payload),
  verifyRera: (id, payload) => http.put(`/admin/properties/${id}/rera`, payload),
  setFeatured: (id, featured) =>
    http.put(`/admin/properties/${id}/featured`, null, { params: { featured } }),

  users: (params) => http.get('/admin/users', { params }),
  customers: (params) => http.get('/admin/customers', { params }),
  clients: (params) => http.get('/admin/clients', { params }),
  updateUserStatus: (id, payload) => http.put(`/admin/users/${id}/status`, payload),
  assignRoles: (id, roleIds) => http.put(`/admin/users/${id}/roles`, { role_ids: roleIds }),
  roles: () => http.get('/admin/roles'),

  inquiries: (params) => http.get('/admin/inquiries', { params }),

  // Moderation: once anyone can post a flat, some listings are wrong or
  // dishonest, and someone has to answer for them.
  reports: (params) => http.get('/admin/reports', { params }),
  resolveReport: (id, payload) => http.put(`/admin/reports/${id}`, payload),
  sweepExpiredListings: () => http.post('/admin/listings/sweep-expiry'),

  // Trust and coverage.
  setClientVerification: (id, payload) => http.put(`/admin/clients/${id}/verification`, payload),
  createPropertyForClient: (id, payload) => http.post(`/admin/clients/${id}/properties`, payload),

  createCity: (payload) => http.post('/admin/cities', payload),
  updateCity: (id, payload) => http.put(`/admin/cities/${id}`, payload),
  deleteCity: (id) => http.delete(`/admin/cities/${id}`),

  createLocality: (payload) => http.post('/admin/localities', payload),
  updateLocality: (id, payload) => http.put(`/admin/localities/${id}`, payload),
  deleteLocality: (id) => http.delete(`/admin/localities/${id}`),

  createAmenity: (payload) => http.post('/admin/amenities', payload),
  updateAmenity: (id, payload) => http.put(`/admin/amenities/${id}`, payload),
  deleteAmenity: (id) => http.delete(`/admin/amenities/${id}`),

  banners: (params) => http.get('/admin/banners', { params }),
  createBanner: (payload) => http.post('/admin/banners', payload),
  updateBanner: (id, payload) => http.put(`/admin/banners/${id}`, payload),
  deleteBanner: (id) => http.delete(`/admin/banners/${id}`),

  featured: (params) => http.get('/admin/featured', { params }),
  addFeatured: (payload) => http.post('/admin/featured', payload),
  removeFeatured: (id) => http.delete(`/admin/featured/${id}`),

  pages: () => http.get('/admin/pages'),
  savePage: (payload) => http.post('/admin/pages', payload),
  deletePage: (id) => http.delete(`/admin/pages/${id}`),

  uploadAsset: (file, folder) => upload('/admin/uploads', file, { folder }),
}

// upload posts a multipart form. Content-Type is left unset so the browser adds
// the multipart boundary itself.
function upload(url, file, fields = {}) {
  const form = new FormData()
  form.append('file', file)
  for (const [key, value] of Object.entries(fields)) {
    if (value !== undefined && value !== null && value !== '') form.append(key, value)
  }
  return http.post(url, form, { headers: { 'Content-Type': undefined } })
}
