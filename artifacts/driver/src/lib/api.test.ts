// Tests for the driver API client (src/lib/api.ts)
// Covers the exact failure classes fixed in recent audits:
// {data} unwrapping, 401 handling, error extraction, WS URL prefix.
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  setAuthToken,
  getAuthToken,
  getAPIBase,
  driverAuthAPI,
  driverAPI,
  getWebSocketURL,
} from './api'

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
  it('persists the token to localStorage and reads it back', () => {
    setAuthToken('abc123')
    expect(localStorage.getItem('avex_driver_token')).toBe('abc123')
    expect(getAuthToken()).toBe('abc123')
  })

  it('removes the token from localStorage when cleared', () => {
    setAuthToken('abc123')
    setAuthToken(null)
    expect(localStorage.getItem('avex_driver_token')).toBeNull()
    expect(getAuthToken()).toBeNull()
  })
})

describe('apiFetch behaviour (via public API methods)', () => {
  it('uses the /api/v1 base and unwraps { data: ... } responses', async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({ body: { data: { token: 't1', must_change_password: false } } })
    )

    const result = await driverAuthAPI.login({ phone: '0100', password: 'pw' })

    expect(getAPIBase()).toBe('/api/v1')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/auth/driver/login')
    expect(options.method).toBe('POST')
    expect(result.token).toBe('t1')
    // After toCamelCase transform, must_change_password → mustChangePassword
    expect(result.mustChangePassword).toBe(false)
  })

  it('sends the Authorization header when a token is set', async () => {
    setAuthToken('tok-42')
    fetchMock.mockResolvedValueOnce(mockResponse({ body: { data: { id: 'd1' } } }))

    await driverAPI.getDriver('d1')

    const [, options] = fetchMock.mock.calls[0]
    expect(options.headers['Authorization']).toBe('Bearer tok-42')
    expect(options.headers['Content-Type']).toBe('application/json')
  })

  it('on 401: clears the token and throws (no redirect from apiFetch)', async () => {
    setAuthToken('expired-token')
    fetchMock.mockResolvedValueOnce(mockResponse({ status: 401 }))

    await expect(driverAPI.getDriver('d1')).rejects.toThrow()

    // Token should be cleared
    expect(getAuthToken()).toBeNull()
    expect(localStorage.getItem('avex_driver_token')).toBeNull()
    // apiFetch no longer redirects — the auth store handles it via state.
  })

  it('extracts the error message from an { error } body on failure', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({ status: 500, body: { error: 'boom' } }))

    await expect(driverAPI.getDriver('d1')).rejects.toThrow('boom')
  })

  it('returns an empty object for 204 No Content responses', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({ status: 204 }))

    const result = await driverAPI.goOnline('d1')
    expect(result).toEqual({})
  })
})

describe('getWebSocketURL', () => {
  it('builds the WS URL from window.location without double /api/v1 prefix', () => {
    const url = getWebSocketURL('my-token')

    expect(url).toBe('ws://localhost:3000/api/v1/ws?token=my-token')
    expect(url).not.toContain('/api/v1/api/v1')
  })

  it('URL-encodes the token', () => {
    const url = getWebSocketURL('a b&c')
    expect(url).toContain('token=a%20b%26c')
  })
})
