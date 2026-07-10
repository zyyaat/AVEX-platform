// AVEX Admin - API client
// All paths use /api/v1/ prefix to match the Go backend.

const API_BASE = '/api/v1'

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
    setAuthToken(null)
    if (typeof window !== 'undefined') window.location.href = '/admin/login'
    throw new Error('انتهت الجلسة')
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  const text = await res.text()
  if (!text) return {} as T
  const json = JSON.parse(text)
  return json.data !== undefined ? json.data : json
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
  updateSetting: (key: string, value: string) => apiFetch('/admin/settings', { method: 'PUT', body: JSON.stringify({ key, value }) }),
}
