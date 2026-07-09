// AVEX Driver — API client for Go backend
// Connects directly to the Go backend (no Next.js proxy).

const API_BASE = import.meta.env.VITE_API_BASE || ''  // Empty = relative URLs through Vite proxy

let authToken: string | null = null

export function setAuthToken(token: string | null) {
  authToken = token
  if (typeof window !== 'undefined') {
    if (token) localStorage.setItem('avex_driver_token', token)
    else localStorage.removeItem('avex_driver_token')
  }
}

export function getAuthToken(): string | null {
  if (authToken) return authToken
  if (typeof window !== 'undefined') {
    authToken = localStorage.getItem('avex_driver_token')
  }
  return authToken
}

export function getAPIBase(): string {
  return API_BASE
}

// ===== TYPES =====

export interface Driver {
  id: string
  user_id: string
  vehicle_type: string
  license_plate: string
  status: string // offline | online | busy | suspended
  rating: number
  rating_count: number
  acceptance_rate: number
  completion_rate: number
  total_deliveries: number
  zone_ids: string[]
  current_order_id: string
  go_online_at: string | null
  go_offline_at: string | null
  created_at: string
  updated_at: string
}

export interface DriverLocation {
  driver_id: string
  lat: number
  lng: number
  bearing: number
  speed: number
  accuracy: number
  captured_at: string
  received_at: string
}

export interface DispatchOffer {
  id: string
  order_id: string
  driver_id: string
  zone_id: string
  status: string // pending | accepted | rejected | expired | cancelled
  pickup_lat: number
  pickup_lng: number
  delivery_lat: number
  delivery_lng: number
  est_distance_m: number | null
  est_duration_s: number | null
  est_fare_cents: number | null
  currency: string
  offer_ttl: string
  offered_at: string
  expires_at: string
  attempt_number: number
}

export interface ActiveOrder {
  id: string
  order_number: string
  user_id: string
  restaurant_id: string
  customer_name: string
  customer_phone: string
  delivery_lat: number
  delivery_lng: number
  delivery_address: string
  delivery_notes: string
  items: { menu_item_id: string; name: string; name_ar: string; price: number; quantity: number }[]
  subtotal: number
  delivery_fee: number
  discount: number
  tax: number
  total: number
  currency: string
  payment_method: string
  status: string
  driver_id: string
  restaurant_name: string
  restaurant_lat: number
  restaurant_lng: number
  created_at: string
  picked_up_at: string | null
  delivered_at: string | null
}

export interface NearbyDriver {
  driver_id: string
  lat: number
  lng: number
  distance_m: number
  bearing: number
  speed: number
  location_age: number
  captured_at: string
}

// ===== CORE FETCH =====

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
    if (typeof window !== 'undefined') {
      const base = (import.meta.env.BASE_URL || '/').replace(/\/$/, '')
      window.location.href = `${base}/login`
    }
    throw new Error('انتهت الجلسة — يرجى تسجيل الدخول مرة أخرى')
  }
  
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(error.error || `HTTP ${res.status}`)
  }
  
  // Some endpoints return no content
  if (res.status === 204) return {} as T
  
  const text = await res.text()
  if (!text) return {} as T
  
  const json = JSON.parse(text)
  // Our Go backend wraps responses in { "data": ... }
  return json.data !== undefined ? json.data : json
}

// ===== AUTH API =====

export const driverAuthAPI = {
  login: (data: { phone: string; password: string }) =>
    apiFetch<{ token: string; user?: any; must_change_password: boolean }>(
      '/api/v1/auth/login', { method: 'POST', body: JSON.stringify(data) }
    ),
  register: (data: { phone: string; password: string; name: string }) =>
    apiFetch<{ token: string }>(
      '/api/v1/auth/register', { method: 'POST', body: JSON.stringify(data) }
    ),
}

// ===== DRIVER API =====

export const driverAPI = {
  // Driver profile
  getDriver: (driverID: string) =>
    apiFetch<Driver>(`/api/v1/drivers/${driverID}`),
  getDriverByUserID: (userID: string) =>
    apiFetch<Driver>(`/api/v1/drivers?user_id=${userID}`),
  goOnline: (driverID: string) =>
    apiFetch<Driver>(`/api/v1/drivers/${driverID}/online`, { method: 'POST' }),
  goOffline: (driverID: string) =>
    apiFetch<Driver>(`/api/v1/drivers/${driverID}/offline`, { method: 'POST' }),

  // Location
  updateLocation: (driverID: string, data: { lat: number; lng: number; bearing: number; speed: number; accuracy: number; captured_at: string }) =>
    apiFetch<any>(`/api/v1/drivers/${driverID}/location`, { method: 'POST', body: JSON.stringify(data) }),
  getLocation: (driverID: string) =>
    apiFetch<DriverLocation>(`/api/v1/drivers/${driverID}/location`),

  // Nearby drivers
  findNearest: (lat: number, lng: number, radius: number, limit: number) =>
    apiFetch<NearbyDriver[]>(`/api/v1/drivers/nearby?lat=${lat}&lng=${lng}&radius=${radius}&limit=${limit}`),

  // Dispatch offers
  getOffer: (offerID: string) =>
    apiFetch<DispatchOffer>(`/api/v1/dispatch/offers/${offerID}`),
  acceptOffer: (offerID: string, driverID: string) =>
    apiFetch<DispatchOffer>(`/api/v1/dispatch/offers/${offerID}/accept`, { method: 'POST', body: JSON.stringify({ driver_id: driverID }) }),
  rejectOffer: (offerID: string, driverID: string, reason?: string) =>
    apiFetch<DispatchOffer>(`/api/v1/dispatch/offers/${offerID}/reject`, { method: 'POST', body: JSON.stringify({ driver_id: driverID, reason: reason || '' }) }),
  listOffersByDriver: (driverID: string, limit = 50, offset = 0) =>
    apiFetch<{ items: DispatchOffer[]; total: number }>(`/api/v1/dispatch/offers?driver_id=${driverID}&limit=${limit}&offset=${offset}`),

  // Orders
  getOrder: (orderID: string) =>
    apiFetch<ActiveOrder>(`/api/v1/orders/${orderID}`),
  listDriverOrders: (driverID: string, limit = 50, offset = 0) =>
    apiFetch<{ items: ActiveOrder[]; total: number }>(`/api/v1/orders?driver_id=${driverID}&limit=${limit}&offset=${offset}`),
  
  // Order lifecycle (driver actions)
  markPickedUp: (orderID: string, driverID: string, pickupPhotoURL?: string) =>
    apiFetch<ActiveOrder>(`/api/v1/orders/${orderID}/pickup`, { method: 'POST', body: JSON.stringify({ driver_id: driverID, pickup_photo_url: pickupPhotoURL || '' }) }),
  markDelivered: (orderID: string, driverID: string, deliveryPhotoURL?: string) =>
    apiFetch<ActiveOrder>(`/api/v1/orders/${orderID}/deliver`, { method: 'POST', body: JSON.stringify({ driver_id: driverID, delivery_photo_url: deliveryPhotoURL || '' }) }),

  // Admin: register new driver
  registerDriver: (data: { user_id: string; vehicle_type: string; license_plate: string; zone_ids: string[] }) =>
    apiFetch<Driver>('/api/v1/admin/drivers', { method: 'POST', body: JSON.stringify(data) }),
}

// ===== CATALOG API (for restaurant info) =====

export const catalogAPI = {
  getRestaurant: (restaurantID: string) =>
    apiFetch<any>(`/api/v1/restaurants/${restaurantID}`),
}

// ===== FINANCIAL API (for wallet) =====

export const financialAPI = {
  getWalletByOwner: (ownerType: string, ownerID: string) =>
    apiFetch<any>(`/api/v1/wallets?owner_type=${ownerType}&owner_id=${ownerID}`),
  listTransactions: (walletID: string, limit = 50, offset = 0) =>
    apiFetch<{ items: any[]; total: number }>(`/api/v1/wallets/${walletID}/transactions?limit=${limit}&offset=${offset}`),
}

// ===== SUPPORT API =====

export const supportAPI = {
  createTicket: (data: { user_id: string; subject: string; description: string; category: string; priority: string }) =>
    apiFetch<any>('/api/v1/support/tickets', { method: 'POST', body: JSON.stringify(data) }),
  getTicket: (id: string) =>
    apiFetch<any>(`/api/v1/support/tickets/${id}`),
  listMyTickets: (userID: string, limit = 50, offset = 0) =>
    apiFetch<{ items: any[]; total: number }>(`/api/v1/support/tickets?user_id=${userID}&limit=${limit}&offset=${offset}`),
  replyToTicket: (ticketID: string, data: { sender_type: string; sender_id: string; body: string }) =>
    apiFetch<any>(`/api/v1/support/tickets/${ticketID}/messages`, { method: 'POST', body: JSON.stringify(data) }),
  listMessages: (ticketID: string, limit = 50, offset = 0) =>
    apiFetch<{ items: any[]; total: number }>(`/api/v1/support/tickets/${ticketID}/messages?limit=${limit}&offset=${offset}`),
}

// ===== SETTINGS API =====

export const settingsAPI = {
  checkFeatureFlag: (name: string, userID: string) =>
    apiFetch<{ name: string; enabled: boolean }>(`/api/v1/feature-flags/check?name=${name}&user_id=${userID}`),
}

// ===== LOCALIZATION API =====

export const i18nAPI = {
  translate: (lang: string, key: string) =>
    apiFetch<{ key: string; value: string; lang: string; found: boolean }>(`/api/v1/i18n/translate?lang=${lang}&key=${encodeURIComponent(key)}`),
  bulkTranslate: (lang: string, keys: string[]) =>
    apiFetch<{ language_code: string; translations: Record<string, any> }>(`/api/v1/i18n/translate/bulk`, { method: 'POST', body: JSON.stringify({ language_code: lang, keys }) }),
}

// ===== WEBSOCKET =====

export function getWebSocketURL(token: string): string {
  if (!API_BASE) {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}/api/v1/ws?token=${encodeURIComponent(token)}`
  }
  const wsBase = API_BASE.replace('http://', 'ws://').replace('https://', 'wss://')
  return `${wsBase}/api/v1/ws?token=${encodeURIComponent(token)}`
}
