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

export const agentAuthAPI = {
  // Support agents use the standard user login
  login: (data: { phone: string; password: string }) =>
    apiFetch<{ token: string; user: any; agent?: any; must_change_password: boolean }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  // FIXED: was /users/me (user endpoint), now /agents/me (agent endpoint)
  me: () => apiFetch<any>('/agents/me').catch(() => null),
}

export const agentAPI = {
  // Use support module endpoints
  getStats: () => apiFetch<any>('/admin/dashboard').catch(() => ({
    openTickets: 0, assignedTickets: 0, resolvedToday: 0, avgResponseTime: 0,
  })),
  // FIXED: backend returns { items, total, limit, offset } (Page wrapper),
  // not { tickets }. We map items → tickets for the frontend.
  getTickets: (filter: string = '') =>
    apiFetch<{ items: any[]; total: number; limit?: number; offset?: number }>(`/support/tickets${filter ? `?status=${filter}` : ''}`)
      .then((r) => ({ tickets: r.items || [], total: r.total || 0 }))
      .catch(() => ({ tickets: [], total: 0 })),
  // FIXED: backend returns a single TicketDTO (not { ticket, messages }).
  // We need to fetch messages separately.
  getTicket: async (id: string): Promise<{ ticket: any; messages: any[] }> => {
    try {
      const ticket = await apiFetch<any>(`/support/tickets/${id}`)
      let messages: any[] = []
      try {
        const msgResult = await apiFetch<{ items: any[]; total: number }>(`/support/tickets/${id}/messages`)
        messages = msgResult.items || []
      } catch {}
      return { ticket, messages }
    } catch {
      return { ticket: null, messages: [] }
    }
  },
  // assignTicket: send the agent's user ID (from JWT subject)
  assignTicket: (id: string, agentId: string = '') =>
    apiFetch<{ success: boolean }>(`/support/tickets/${id}/assign`, { method: 'POST', body: JSON.stringify({ agent_id: agentId }) }),
  setPriority: (id: string, priority: string) =>
    apiFetch(`/support/tickets/${id}/priority`, { method: 'POST', body: JSON.stringify({ priority }) }),
  // FIXED: sender_id should be the agent's ID, not empty string.
  // The caller should pass the agent ID from the auth store.
  sendMessage: (id: string, body: string, agentId: string, isInternal: boolean = false) =>
    apiFetch<{ id: string }>(`/support/tickets/${id}/messages`, { method: 'POST', body: JSON.stringify({ sender_type: isInternal ? 'internal' : 'agent', sender_id: agentId, body }) }),
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
