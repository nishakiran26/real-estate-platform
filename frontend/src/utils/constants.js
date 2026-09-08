/** Values that are part of the API contract and referenced across views. */

export const ROLES = {
  ADMIN: 'ADMIN',
  CLIENT: 'CLIENT',
  CUSTOMER: 'CUSTOMER',
}

export const PROPERTY_STATUS = {
  DRAFT: 'DRAFT',
  SUBMITTED: 'SUBMITTED',
  UNDER_REVIEW: 'UNDER_REVIEW',
  APPROVED: 'APPROVED',
  PUBLISHED: 'PUBLISHED',
  REJECTED: 'REJECTED',
  SOLD_OUT: 'SOLD_OUT',
}

/** Badge colour per workflow state, used by the client and admin tables. */
export const STATUS_VARIANT = {
  DRAFT: 'neutral',
  SUBMITTED: 'info',
  UNDER_REVIEW: 'warning',
  APPROVED: 'success',
  PUBLISHED: 'success',
  REJECTED: 'danger',
  SOLD_OUT: 'neutral',
}

export const INQUIRY_STATUS_VARIANT = {
  NEW: 'info',
  CONTACTED: 'warning',
  SITE_VISIT_SCHEDULED: 'warning',
  NEGOTIATING: 'warning',
  CLOSED_WON: 'success',
  CLOSED_LOST: 'neutral',
}

export const RERA_STATUS_VARIANT = {
  VERIFIED: 'success',
  PENDING_VERIFICATION: 'warning',
  REJECTED: 'danger',
  NOT_AVAILABLE: 'neutral',
}

export const MEDIA_TYPES = [
  { value: 'PROPERTY_IMAGE', label: 'Property image' },
  { value: 'SOCIETY_IMAGE', label: 'Society image' },
  { value: 'FLOOR_PLAN', label: 'Floor plan' },
  { value: 'MASTER_PLAN', label: 'Master plan' },
  { value: 'LOCATION_MAP', label: 'Location map' },
  { value: 'CONSTRUCTION_IMAGE', label: 'Construction progress' },
]

export const DOC_TYPES = [
  { value: 'BROCHURE', label: 'Brochure' },
  { value: 'RERA_CERTIFICATE', label: 'RERA certificate' },
  { value: 'OTHER', label: 'Other document' },
]

export const EVENT_TYPES = {
  VIEW: 'VIEW',
  INQUIRY: 'INQUIRY',
  WHATSAPP_CLICK: 'WHATSAPP_CLICK',
  PHONE_CLICK: 'PHONE_CLICK',
  EMAIL_CLICK: 'EMAIL_CLICK',
  BROCHURE_DOWNLOAD: 'BROCHURE_DOWNLOAD',
}

export const ANALYTICS_PERIODS = [
  { value: 'today', label: 'Today' },
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: 'custom', label: 'Custom range' },
]

export const PAGE_SIZES = [12, 24, 48]

/** Placeholder shown when a listing has no cover image yet. */
export const PLACEHOLDER_IMAGE =
  'data:image/svg+xml;charset=utf-8,' +
  encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" width="640" height="420" viewBox="0 0 640 420">
      <rect width="640" height="420" fill="#eff8f5"/>
      <path d="M180 270h280v60H180z" fill="#d6ece7"/>
      <path d="M215 160l70 80h-140zM390 130l90 110H300z" fill="#a9d5cb"/>
      <circle cx="455" cy="120" r="22" fill="#c9e4dc"/>
      <text x="320" y="365" font-family="sans-serif" font-size="19" fill="#5c8a80"
        text-anchor="middle">Image coming soon</text>
    </svg>`,
  )
