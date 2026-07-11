// AVEX Admin - API client
// All paths use /api/v1/ prefix to match the Go backend.

const API_BASE = '/api/v1'

import { toCamelCase } from './transformer'

let authToken: string | null = null
export function setAuthToken(t: string | null) {
  authToken = t
  if (typeof window !== 'undefined') {
    if (t) localStorage.setItem('avex_admin_token', t)
    else localStorage.removeItem('avex_admin_token')
  }
}
export function getAuthToken(): string | null {
  if (authToken) return authToken
  if (typeof window !== 'undefined') authToken = localStorage.getItem('avex_admin_token')
  return authToken
}
async function apiFetch<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = getAuthToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  }
  if (token) headers['Authorization'] = `Bearer ${token}`
  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE}${endpoint}`
  const res = await fetch(url, { ...options, headers })
  if (res.status === 401) {
    // 401 can mean two things:
    // 1. Wrong credentials at login (backend returns "invalid phone or password")
    // 2. Expired/invalid token on authenticated endpoints ("invalid or expired token")
    // We must extract the ACTUAL error message — not show a generic "session expired".
    let errorMsg = 'انتهت الجلسة — يرجى تسجيل الدخول مرة أخرى'
    try {
      const errBody = await res.json()
      if (typeof errBody.error === 'string') {
        errorMsg = errBody.error
      } else if (errBody.error && typeof errBody.error.message === 'string') {
        errorMsg = errBody.error.message
      } else if (typeof errBody.message === 'string') {
        errorMsg = errBody.message
      }
    } catch {}
    // Only clear the token if this is NOT a login/register endpoint.
    const isAuthEndpoint = url.includes('/auth/login') || url.includes('/auth/register') || url.includes('/auth/driver/login') || url.includes('/auth/driver/register')
    if (!isAuthEndpoint) {
      setAuthToken(null)
    }
    throw new Error(errorMsg)
  }
  if (!res.ok) {
    // Backend returns { error: { message: "...", code: "..." } } (nested object).
    // Some endpoints may return { error: "string" }.
    // If JSON parse fails (e.g. HTML error page from proxy), show HTTP status.
    let errorMsg = ''
    try {
      const errBody = await res.json()
      if (typeof errBody.error === 'string') {
        errorMsg = errBody.error
      } else if (errBody.error && typeof errBody.error.message === 'string') {
        errorMsg = errBody.error.message
      } else if (typeof errBody.message === 'string') {
        errorMsg = errBody.message
      } else if (typeof errBody.error === 'object') {
        errorMsg = JSON.stringify(errBody.error)
      } else {
        errorMsg = `HTTP ${res.status}`
      }
    } catch {
      errorMsg = `HTTP ${res.status}`
    }
    throw new Error(errorMsg)
  }
  const text = await res.text()
  if (!text) return {} as T
  const json = JSON.parse(text)
  // Our Go backend wraps responses in { "data": ... }
  const payload = json.data !== undefined ? json.data : json
  // Transform snake_case keys to camelCase so frontend types work correctly.
  return toCamelCase<T>(payload)
}

export const adminAuthAPI = {
  login: (data: { phone: string; password: string }) =>
    apiFetch<{ token: string; user: any }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  me: () => apiFetch<any>('/users/me'),
}

export const adminAPI = {
  // Dashboard (not yet in backend — will return fallback data)
  getDashboard: () => apiFetch<any>('/admin/dashboard').catch(() => ({
    todayOrders: 0, activeOrders: 0, onlineDrivers: 0,
    todayRevenue: 0, platformMargin: 0, openTickets: 0,
    totalCustomers: 0, totalRestaurants: 0, byStatus: {},
  })),
  getOrders: (status?: string) => apiFetch<{ orders: any[] }>(`/orders?status=${status || ''}`).catch(() => ({ orders: [] })),

  // Zones (not yet in backend)
  getZones: () => apiFetch<{ zones: any[] }>('/admin/zones').catch(() => ({ zones: [] })),
  createZone: (data: any) => apiFetch<{ id: string }>('/admin/zones', { method: 'POST', body: JSON.stringify(data) }),
  updateZone: (id: string, data: any) => apiFetch(`/admin/zones/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteZone: (id: string) => apiFetch(`/admin/zones/${id}`, { method: 'DELETE' }),

  // Tiers (not yet in backend)
  getTiers: () => apiFetch<{ tiers: any[] }>('/admin/tiers').catch(() => ({ tiers: [] })),
  createTier: (data: any) => apiFetch<{ id: string }>('/admin/tiers', { method: 'POST', body: JSON.stringify(data) }),
  updateTier: (id: string, data: any) => apiFetch(`/admin/tiers/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  updateTierThresholds: (id: string, data: any) => apiFetch(`/admin/tiers/${id}/thresholds`, { method: 'PUT', body: JSON.stringify(data) }),

  // Tier Prices (not yet in backend)
  getTierPrices: (zoneId?: string) => apiFetch<{ prices: any[] }>(`/admin/tier-prices?zone_id=${zoneId || ''}`).catch(() => ({ prices: [] })),
  updateTierPrice: (tierId: string, zoneId: string, data: any) =>
    apiFetch(`/admin/tier-prices/${tierId}/${zoneId}`, { method: 'PUT', body: JSON.stringify(data) }),

  // Drivers (exists in dispatch module)
  getDrivers: () => apiFetch<{ drivers: any[] }>('/admin/drivers').catch(() => ({ drivers: [] })),
  updateDriverStatus: (id: string, isActive: boolean) =>
    apiFetch(`/admin/drivers/${id}/status`, { method: 'PATCH', body: JSON.stringify({ isActive }) }),
  updateDriverTier: (id: string, tierId: string) =>
    apiFetch(`/admin/drivers/${id}/tier`, { method: 'PATCH', body: JSON.stringify({ tierId }) }),
  getDriverTierHistory: (id: string) => apiFetch<{ history: any[] }>(`/admin/drivers/${id}/tier-history`).catch(() => ({ history: [] })),
  createShift: (id: string, data: any) => apiFetch<{ id: string }>(`/admin/drivers/${id}/shifts`, { method: 'POST', body: JSON.stringify(data) }),
  getShifts: (id: string) => apiFetch<{ shifts: any[] }>(`/admin/drivers/${id}/shifts`).catch(() => ({ shifts: [] })),

  // Applications (not yet in backend)
  getApplications: () => apiFetch<{ applications: any[] }>('/admin/driver-applications').catch(() => ({ applications: [] })),
  createApplication: (data: any) => apiFetch<{ id: string }>('/admin/driver-applications', { method: 'POST', body: JSON.stringify(data) }),
  verifyApplication: (id: string) => apiFetch<{ success: boolean; driverId: string; initialPassword: string }>(`/admin/driver-applications/${id}/verify`, { method: 'PATCH' }),
  rejectApplication: (id: string, reason: string) => apiFetch(`/admin/driver-applications/${id}/reject`, { method: 'PATCH', body: JSON.stringify({ reason }) }),

  // Restaurants (exists in catalog module)
  getRestaurants: () => apiFetch<{ restaurants: any[] }>('/admin/restaurants').catch(() => ({ restaurants: [] })),
  createRestaurant: (data: any) => apiFetch<{ id: string; merchantPhone: string; merchantPassword: string }>('/admin/restaurants', { method: 'POST', body: JSON.stringify(data) }),
  updateRestaurant: (id: string, data: any) => apiFetch(`/admin/restaurants/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteRestaurant: (id: string) => apiFetch(`/admin/restaurants/${id}`, { method: 'DELETE' }),

  // Support (exists in support module)
  getTickets: () => apiFetch<{ tickets: any[] }>('/admin/support/tickets').catch(() => ({ tickets: [] })),
  resolveTicket: (id: string, notes: string) => apiFetch(`/admin/support/tickets/${id}/resolve`, { method: 'PATCH', body: JSON.stringify({ adminNotes: notes }) }),
  sendMessage: (id: string, body: string) => apiFetch<{ id: string }>(`/admin/support/tickets/${id}/messages`, { method: 'POST', body: JSON.stringify({ body }) }),
  cancelOrder: (id: string) => apiFetch(`/admin/support/tickets/${id}/cancel-order`, { method: 'POST' }),

  // Settings (exists in settings module)
  getSettings: () => apiFetch<{ settings: Record<string, string> }>('/admin/settings').catch(() => ({ settings: {} })),
  updateSetting: (key: string, value: string) => apiFetch('/admin/settings', { method: 'PUT', body: JSON.stringify({ key, value }) }),
}
