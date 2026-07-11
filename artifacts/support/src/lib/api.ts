// AVEX Support — API client
// All paths use /api/v1/ prefix to match the Go backend.
// Support-specific endpoints don't exist yet. All calls have .catch()
// fallbacks so the app doesn't crash.

const API_BASE = '/api/v1'

import { toCamelCase } from './transformer'

let authToken: string | null = null
export function setAuthToken(t: string | null) {
  authToken = t
  if (typeof window !== 'undefined') {
    if (t) localStorage.setItem('avex_agent_token', t)
    else localStorage.removeItem('avex_agent_token')
  }
}
export function getAuthToken(): string | null {
  if (authToken) return authToken
  if (typeof window !== 'undefined') authToken = localStorage.getItem('avex_agent_token')
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

export const agentAuthAPI = {
  // Support agents use the standard user login
  login: (data: { phone: string; password: string }) =>
    apiFetch<{ token: string; user: any; agent?: any; must_change_password: boolean }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  me: () => apiFetch<any>('/users/me').catch(() => null),
}

export const agentAPI = {
  // Use support module endpoints
  getStats: () => apiFetch<any>('/admin/dashboard').catch(() => ({
    openTickets: 0, assignedTickets: 0, resolvedToday: 0, avgResponseTime: 0,
  })),
  getTickets: (filter: string = '') =>
    apiFetch<{ tickets: any[]; agentId?: string }>(`/support/tickets${filter ? `?status=${filter}` : ''}`).catch(() => ({ tickets: [], agentId: '' })),
  getTicket: (id: string) =>
    apiFetch<{ ticket: any; messages: any[] }>(`/support/tickets/${id}`).catch(() => ({ ticket: null, messages: [] })),
  assignTicket: (id: string, agentId: string = '') =>
    apiFetch<{ success: boolean }>(`/support/tickets/${id}/assign`, { method: 'POST', body: JSON.stringify({ agent_id: agentId }) }),
  setPriority: (id: string, priority: string) =>
    apiFetch(`/support/tickets/${id}/priority`, { method: 'POST', body: JSON.stringify({ priority }) }),
  sendMessage: (id: string, body: string, isInternal: boolean = false) =>
    apiFetch<{ id: string }>(`/support/tickets/${id}/messages`, { method: 'POST', body: JSON.stringify({ sender_type: isInternal ? 'internal' : 'agent', sender_id: '', body }) }),
  resolveTicket: (id: string, notes: string) =>
    apiFetch(`/support/tickets/${id}/close`, { method: 'POST', body: JSON.stringify({ closed_by: 'agent', reason: notes }) }),
  cancelOrder: (id: string) =>
    apiFetch(`/support/tickets/${id}/cancel-order`, { method: 'POST' }),
  search: (q: string) =>
    apiFetch<{ customers: any[]; drivers: any[]; orders: any[] }>(`/admin/search?q=${encodeURIComponent(q)}`).catch(() => ({ customers: [], drivers: [], orders: [] })),
  getOrder: (id: string) =>
    apiFetch<{ order: any }>(`/orders/${id}`).catch(() => ({ order: null })),
  getDriver: (id: string) =>
    apiFetch<{ driver: any; stats: any; recentOrders: any[] }>(`/drivers/${id}`).catch(() => ({ driver: null, stats: null, recentOrders: [] })),
}
