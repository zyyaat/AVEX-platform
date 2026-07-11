// AVEX Merchant — API client
// All paths use /api/v1/ prefix to match the Go backend.
// Note: Merchant-specific backend endpoints don't exist yet. All calls
// have .catch() fallbacks so the app doesn't crash.

const API_BASE = '/api/v1'

import { toCamelCase } from './transformer'

let authToken: string | null = null

export function setAuthToken(t: string | null) {
  authToken = t
  if (typeof window !== 'undefined') {
    if (t) localStorage.setItem('avex_merchant_token', t)
    else localStorage.removeItem('avex_merchant_token')
  }
}

export function getAuthToken(): string | null {
  if (authToken) return authToken
  if (typeof window !== 'undefined') authToken = localStorage.getItem('avex_merchant_token')
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
    // Clear the in-memory token. Don't redirect — let the auth store
    // handle it gracefully via initialize() + route guard.
    setAuthToken(null)
    throw new Error('انتهت الجلسة')
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  const text = await res.text()
  if (!text) return {} as T
  const json = JSON.parse(text)
  // Our Go backend wraps responses in { "data": ... }
  const payload = json.data !== undefined ? json.data : json
  // Transform snake_case keys to camelCase so frontend types work correctly.
  return toCamelCase<T>(payload)
}

// ===== Types =====
export interface Merchant {
  id: string
  name: string
  phone: string
  isActive: boolean
  mustChangePassword: boolean
  restaurant: {
    id: string
    name: string
    nameAr: string
    descriptionAr: string
    lat: number
    lng: number
    rating: number
    ratingCount: number
    isActive: boolean
    isPro: boolean
    deliveryTimeMin: number
    deliveryTimeMax: number
    deliveryFee: number
    minOrder: number
  }
}

export interface MerchantOrder {
  id: string
  orderNumber: string
  customerName: string
  phone: string
  locationAddress: string
  locationLat: number
  locationLng: number
  locationUrl: string
  subtotal: number
  deliveryFee: number
  discount: number
  total: number
  paymentMethod: string
  status: string
  createdAt: string
  updatedAt: string
  driverId: string
  scheduledFor: string
  itemsSummary: string
  itemsCount: number
}

export interface OrderItem {
  id: string
  menuItemId: string
  name: string
  price: number
  quantity: number
}

export interface MenuItem {
  id: string
  name: string
  nameAr: string
  description: string
  descriptionAr: string
  price: number
  image: string
  imageUrl: string
  isPopular: boolean
  isAvailable: boolean
  rating: number
  ratingCount: number
  prepTime: number
  calories: number
  categoryId: string
}

export interface Category {
  id: string
  name: string
  nameAr: string
  icon: string
}

export interface StoreHour {
  id: string
  dayOfWeek: number
  openTime: string
  closeTime: string
  isOpen: boolean
}

export interface MerchantStats {
  todayCount: number
  activeCount: number
  completedCount: number
  todayRevenue: number
  daily: { date: string; revenue: number; count: number }[] | null
}

export const merchantAuthAPI = {
  // Merchant login uses the same user login endpoint (merchant is a user with merchant role)
  login: (data: { phone: string; password: string }) =>
    apiFetch<{ token: string; user: any; merchant?: any; must_change_password: boolean }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  changePassword: (data: { oldPassword: string; newPassword: string }) =>
    apiFetch<{ success: boolean }>('/auth/change-password', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  // FIXED: was /users/me (user endpoint, returns UserDTO), now /merchants/me
  // (merchant endpoint, returns MerchantProfileDTO with restaurant info)
  me: () => apiFetch<any>('/merchants/me').catch(() => null),
}

export const merchantAPI = {
  // Orders — use the orders module with restaurant filter
  getOrders: (status?: string) =>
    apiFetch<{ orders: MerchantOrder[] | null }>(`/orders?status=${status || ''}`).catch(() => ({ orders: null })),
  getOrderItems: (id: string) =>
    apiFetch<{ items: OrderItem[] | null }>(`/orders/${id}/items`).catch(() => ({ items: null })),
  updateOrderStatus: (id: string, status: string) =>
    apiFetch<{ success: boolean; status: string }>(`/orders/${id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status }),
    }),

  // Menu — use catalog module admin endpoints
  getMenu: () =>
    apiFetch<{ items: MenuItem[] | null; categories: Category[] | null }>('/admin/menu-items').catch(() => ({ items: null, categories: null })),
  createMenuItem: (data: Partial<MenuItem>) =>
    apiFetch<{ id: string }>('/admin/menu-items', { method: 'POST', body: JSON.stringify(data) }),
  updateMenuItem: (id: string, data: Partial<MenuItem>) =>
    apiFetch<{ success: boolean }>(`/admin/menu-items/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteMenuItem: (id: string) =>
    apiFetch<{ success: boolean }>(`/admin/menu-items/${id}`, { method: 'DELETE' }),

  // Store hours — not yet in backend
  getHours: () => apiFetch<{ hours: StoreHour[] | null }>('/admin/store-hours').catch(() => ({ hours: null })),
  updateHours: (hours: any[]) =>
    apiFetch<{ success: boolean }>('/admin/store-hours', { method: 'PUT', body: JSON.stringify({ hours }) }),
  togglePause: (isActive: boolean) =>
    apiFetch<{ isActive: boolean }>('/admin/restaurants/pause', { method: 'POST', body: JSON.stringify({ isActive }) }),
  getStats: () => apiFetch<MerchantStats>('/admin/merchant-stats').catch(() => ({
    todayCount: 0, activeCount: 0, completedCount: 0, todayRevenue: 0, daily: null,
  })),
  getScheduledOrders: () =>
    apiFetch<{ scheduledOrders: any[] | null }>('/admin/scheduled-orders').catch(() => ({ scheduledOrders: null })),
}
