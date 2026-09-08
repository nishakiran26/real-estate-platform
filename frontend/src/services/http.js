import axios from 'axios'

const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

// Anonymous visitors are tracked by a session id so views, WhatsApp clicks and
// inquiries can be attributed without forcing a login. The backend also sets a
// cookie; the header keeps one session across tabs and survives cookie clearing.
const SESSION_KEY = 'rera_session_id'

function sessionId() {
  let id = localStorage.getItem(SESSION_KEY)
  if (!id) {
    id =
      typeof crypto !== 'undefined' && crypto.randomUUID
        ? crypto.randomUUID()
        : `s-${Date.now()}-${Math.random().toString(36).slice(2)}`
    localStorage.setItem(SESSION_KEY, id)
  }
  return id
}

const http = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
  timeout: 20000,
  headers: { 'Content-Type': 'application/json' },
})

// Token accessors are injected by the auth store, which owns token storage.
// Without this indirection the store and this module would import each other.
let tokenProvider = () => null
let onRefresh = null
let onAuthFailure = null

export function configureAuth({ getAccessToken, refresh, onUnauthorized }) {
  tokenProvider = getAccessToken
  onRefresh = refresh
  onAuthFailure = onUnauthorized
}

http.interceptors.request.use((config) => {
  const token = tokenProvider()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  config.headers['X-Session-Id'] = sessionId()
  return config
})

// A single in-flight refresh is shared by every request that got a 401, so a
// burst of parallel calls does not trigger a burst of refreshes.
let refreshing = null

http.interceptors.response.use(
  (response) => response.data,
  async (error) => {
    const { response, config } = error

    if (!response) {
      return Promise.reject(
        normalize({ message: 'Network error. Please check your connection and try again.' }),
      )
    }

    const isRefreshCall = config?.url?.includes('/auth/refresh-token')
    if (response.status === 401 && !config?._retried && !isRefreshCall && onRefresh) {
      config._retried = true
      try {
        refreshing = refreshing || onRefresh()
        const ok = await refreshing
        refreshing = null
        if (ok) {
          return http(config)
        }
      } catch {
        refreshing = null
      }
      if (onAuthFailure) onAuthFailure()
    }

    return Promise.reject(normalize(response.data, response.status))
  },
)

// normalize turns any backend or transport failure into one predictable shape,
// so callers never have to guess where the message lives.
function normalize(data, status = 0) {
  const err = new Error(data?.message || 'Something went wrong. Please try again.')
  err.status = status
  err.fields = {}
  if (Array.isArray(data?.errors)) {
    for (const item of data.errors) {
      if (item?.field) err.fields[item.field] = item.message
    }
  }
  err.payload = data
  return err
}

export { sessionId }
export default http
