// AVEX Customer - API client
// All paths use /api/v1/ prefix to match the Go backend.

const API_BASE = '/api/v1'

import { toCamelCase } from './transformer'

let authToken: string | null = null

export function setAuthToken(token: string | null) {
  authToken = token
  if (typeof window !== 'undefined') {
    if (token) localStorage.setItem('avex_token', token)
    else localStorage.removeItem('avex_token')
  }
}

export function getAuthToken(): string | null {
  if (authToken) return authToken
  if (typeof window !== 'undefined') {
    authToken = localStorage.getItem('avex_token')
  }
  return authToken
}

export interface User {
  id: string
  name: string
  phone: string
  email: string
  loyaltyPoints: number
  isAdmin: boolean
  createdAt: string
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
    const error = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(error.error || `HTTP ${res.status}`)
  }
  const text = await res.text()
  if (!text) return {} as T
  const json = JSON.parse(text)
  // Our Go backend wraps responses in { "data": ... }
  const payload = json.data !== undefined ? json.data : json
  // Transform snake_case keys to camelCase so frontend types work correctly.
  return toCamelCase<T>(payload)
}

export const authAPI = {
  register: (data: { name: string; phone: string; password: string; email?: string }) =>
    apiFetch<{ token: string; user: User }>('/auth/register', { method: 'POST', body: JSON.stringify(data) }),
  login: (data: { phone: string; password: string }) =>
    apiFetch<{ token: string; user: User }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  me: () => apiFetch<User>('/users/me'),
}

export const menuAPI = {
  getCategories: () => apiFetch<{ categories: any[] }>('/categories').catch(() => ({ categories: [] })),
  getRestaurants: () => apiFetch<{ restaurants: any[] }>('/restaurants').catch(() => ({ restaurants: [] })),
  getRestaurant: (id: string) => apiFetch<any>(`/restaurants/${id}`),
  getMenu: (restaurantId: string) => apiFetch<any>(`/restaurants/${restaurantId}/menu`),
}

export const ordersAPI = {
  create: (data: any) => apiFetch<any>('/orders', { method: 'POST', body: JSON.stringify(data) }),
  // FIXED: was '/orders' (admin endpoint), now '/orders/my' (user's orders)
  // Response shape is { items, total } (Page wrapper), not { orders }
  getMyOrders: () => apiFetch<{ items: any[]; total: number }>('/orders/my').catch(() => ({ items: [], total: 0 })),
  // FIXED: was query '?number=X', backend expects path param '/orders/track/{orderNumber}'
  trackByNumber: (orderNumber: string) => apiFetch<any>(`/orders/track/${encodeURIComponent(orderNumber)}`),
}

export const couponsAPI = {
  validate: (code: string, subtotal: number) =>
    apiFetch<{ valid: boolean; discount: number; code: string }>('/promotions/validate', { method: 'POST', body: JSON.stringify({ code, order_total: subtotal, currency: 'EGP' }) }),
}

export const userAPI = {
  getAddresses: () => apiFetch<{ addresses: any[] }>('/addresses').catch(() => ({ addresses: [] })),
  saveAddress: (data: any) => apiFetch<{ id: string }>('/addresses', { method: 'POST', body: JSON.stringify(data) }),
  deleteAddress: (id: string) => apiFetch(`/addresses/${id}`, { method: 'DELETE' }),
  getFavorites: () => apiFetch<{ favorites: any[] }>('/favorites').catch(() => ({ favorites: [] })),
  toggleFavorite: (menuItemId: string) => apiFetch<{ favorited: boolean }>(`/favorites/${menuItemId}/toggle`, { method: 'POST' }),
  getCards: () => apiFetch<{ cards: any[] }>('/cards').catch(() => ({ cards: [] })),
  saveCard: (data: any) => apiFetch<{ id: string }>('/cards', { method: 'POST', body: JSON.stringify(data) }),
  deleteCard: (id: string) => apiFetch(`/cards/${id}`, { method: 'DELETE' }),
  setDefaultCard: (id: string) => apiFetch<{ success: boolean }>(`/cards/${id}/default`, { method: 'POST' }),
}

export const adminAPI = {
  getCategories: () => apiFetch<{ categories: any[] }>('/admin/categories').catch(() => ({ categories: [] })),
  createCategory: (data: any) => apiFetch<{ id: string }>('/admin/categories', { method: 'POST', body: JSON.stringify(data) }),
  updateCategory: (id: string, data: any) => apiFetch(`/admin/categories/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteCategory: (id: string) => apiFetch(`/admin/categories/${id}`, { method: 'DELETE' }),
  getMenuItems: () => apiFetch<{ items: any[] }>('/admin/menu-items').catch(() => ({ items: [] })),
  createMenuItem: (data: any) => apiFetch<{ id: string }>('/admin/menu-items', { method: 'POST', body: JSON.stringify(data) }),
  updateMenuItem: (id: string, data: any) => apiFetch(`/admin/menu-items/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteMenuItem: (id: string) => apiFetch(`/admin/menu-items/${id}`, { method: 'DELETE' }),
  updateOrderStatus: (id: string, status: string) => apiFetch(`/orders/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  updateSetting: (key: string, value: string) => apiFetch('/admin/settings', { method: 'PUT', body: JSON.stringify({ key, value }) }),
}
