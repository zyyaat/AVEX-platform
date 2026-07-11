// Tests for the customer API client (src/lib/api.ts)
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setAuthToken, getAuthToken, authAPI, menuAPI } from './api'

function mockResponse(init: { status?: number; body?: unknown }) {
  const status = init.status ?? 200
  return {
    status,
    ok: status >= 200 && status < 300,
    text: async () => (init.body === undefined ? '' : JSON.stringify(init.body)),
    json: async () => init.body,
  } as unknown as Response
}

const fetchMock = vi.fn()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockReset()
  localStorage.clear()
  setAuthToken(null)
})

describe('auth token storage', () => {
  it('persists the token under avex_token', () => {
    setAuthToken('abc123')
    expect(localStorage.getItem('avex_token')).toBe('abc123')
    expect(getAuthToken()).toBe('abc123')
  })

  it('removes the token when cleared', () => {
    setAuthToken('abc123')
    setAuthToken(null)
    expect(localStorage.getItem('avex_token')).toBeNull()
    expect(getAuthToken()).toBeNull()
  })
})

describe('apiFetch behaviour', () => {
  it('uses the /api/v1 base and unwraps { data: ... }', async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({ body: { data: { token: 't1', user: { id: 'u1', name: 'Test' } } } })
    )

    const result = await authAPI.login({ phone: '0100', password: 'pw' })

    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/auth/login')
    expect(options.method).toBe('POST')
    expect(result.token).toBe('t1')
    expect(result.user.id).toBe('u1')
  })

  it('sends the Authorization header when a token is set', async () => {
    setAuthToken('tok-42')
    fetchMock.mockResolvedValueOnce(mockResponse({ body: { data: { id: 'u1' } } }))

    await authAPI.me()

    const [, options] = fetchMock.mock.calls[0]
    expect(options.headers['Authorization']).toBe('Bearer tok-42')
  })

  it('on 401: clears the token and redirects to /?auth=login', async () => {
    setAuthToken('expired')
    fetchMock.mockResolvedValueOnce(mockResponse({ status: 401 }))

    await expect(authAPI.me()).rejects.toThrow()

    expect(getAuthToken()).toBeNull()
    expect(window.location.href).toContain('/?auth=login')
  })

  it('extracts the error message from an { error } body', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({ status: 500, body: { error: 'boom' } }))
    await expect(authAPI.me()).rejects.toThrow('boom')
  })

  it('read endpoints fall back gracefully on network failure', async () => {
    fetchMock.mockRejectedValueOnce(new Error('network down'))
    const result = await menuAPI.getCategories()
    expect(result).toEqual({ categories: [] })
  })
})
