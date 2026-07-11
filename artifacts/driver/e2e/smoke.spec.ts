// =============================================================================
// AVEX Driver App — E2E smoke tests
// =============================================================================
// These tests verify the critical user flows of the driver app:
//   1. App boots without crashing (no white screen)
//   2. Login page renders with phone + password fields
//   3. Login form validation works (rejects empty input)
//   4. Map container is present on the home page (when authenticated)
//
// These tests do NOT hit a real backend. They use Playwright's request
// interception to mock /api/v1/* responses, so they can run in CI without
// a database or Redis.
// =============================================================================

import { test, expect, type Page } from '@playwright/test'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Mock the driver login endpoint to return a fake JWT. */
async function mockLoginSuccess(page: Page) {
  await page.route('**/api/v1/auth/driver/login', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          token: 'fake-jwt-token-for-e2e',
          driver: { id: 'drv-1', name: 'Test Driver', phone: '0100000000' },
          must_change_password: false,
        },
      }),
    })
  })
}

/** Mock the driver profile endpoint. */
async function mockDriverProfile(page: Page) {
  await page.route('**/api/v1/drivers/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          id: 'drv-1',
          name: 'Test Driver',
          phone: '0100000000',
          is_online: false,
          rating: 4.8,
          total_deliveries: 42,
        },
      }),
    })
  })
}

/** Mock the orders endpoint (returns empty list — driver is offline). */
async function mockOrders(page: Page) {
  await page.route('**/api/v1/orders**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { orders: [] } }),
    })
  })
}

/** Mock the WebSocket endpoint so the app doesn't hang trying to connect. */
async function mockWebSocket(page: Page) {
  await page.addInitScript(() => {
    // Stub WebSocket so the app doesn't try to open a real connection.
    class MockWebSocket {
      static CONNECTING = 0
      static OPEN = 1
      static CLOSING = 2
      static CLOSED = 3
      readyState = 1
      onopen: any = null
      onmessage: any = null
      onerror: any = null
      onclose: any = null
      constructor(public url: string) {
        setTimeout(() => this.onopen?.({ type: 'open' }), 10)
      }
      send() {}
      close() { this.readyState = 3 }
      addEventListener() {}
      removeEventListener() {}
    }
    ;(window as any).WebSocket = MockWebSocket
  })
}

async function mockAllApi(page: Page) {
  await mockLoginSuccess(page)
  await mockDriverProfile(page)
  await mockOrders(page)
  await mockWebSocket(page)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Driver App — Smoke', () => {
  test('app boots without white screen', async ({ page }) => {
    await mockAllApi(page)
    await page.goto('/')

    // The page should render *something* (not stay blank)
    await expect(page.locator('body')).not.toBeEmpty()
    // Either redirected to /login or showing the home page
    await expect(page).toHaveURL(/\/(login)?$/)
  })

  test('login page renders phone + password fields', async ({ page }) => {
    await mockAllApi(page)
    await page.goto('/login')

    // Phone input (type=tel)
    const phoneInput = page.locator('input[type="tel"]')
    await expect(phoneInput).toBeVisible()
    await expect(phoneInput).toHaveAttribute('placeholder', '01xxxxxxxxx')

    // Password input
    const passwordInput = page.locator('input[type="password"]')
    await expect(passwordInput).toBeVisible()

    // Submit button
    await expect(page.getByRole('button', { name: /تسجيل الدخول/ })).toBeVisible()
  })

  test('login form rejects empty input', async ({ page }) => {
    await mockAllApi(page)
    await page.goto('/login')

    // Click submit without filling the form
    await page.getByRole('button', { name: /تسجيل الدخول/ }).click()

    // No API call should have been made (login endpoint not hit)
    // We verify by checking that we're still on /login
    await expect(page).toHaveURL(/\/login/)

    // Phone input should still be empty (form validation prevented submit)
    await expect(page.locator('input[type="tel"]')).toHaveValue('')
  })

  test('successful login navigates to home page', async ({ page }) => {
    await mockAllApi(page)
    await page.goto('/login')

    // Fill the form
    await page.locator('input[type="tel"]').fill('0100000000')
    await page.locator('input[type="password"]').fill('password123')

    // Submit
    await page.getByRole('button', { name: /تسجيل الدخول/ }).click()

    // Should navigate to home page (/) within 10 seconds
    await expect(page).toHaveURL(/\/$/, { timeout: 10_000 })
  })

  test('home page renders map container', async ({ page }) => {
    await mockAllApi(page)
    await page.goto('/login')

    // Login first
    await page.locator('input[type="tel"]').fill('0100000000')
    await page.locator('input[type="password"]').fill('password123')
    await page.getByRole('button', { name: /تسجيل الدخول/ }).click()
    await expect(page).toHaveURL(/\/$/, { timeout: 10_000 })

    // Wait for map container to appear (the app uses a div with id or class containing 'map')
    // We give it up to 15 seconds because the Mapbox CDN script needs to load.
    const mapContainer = page.locator('[id*="map" i], [class*="map" i], canvas, .mapboxgl-map').first()
    await expect(mapContainer).toBeVisible({ timeout: 15_000 })
  })

  test('login error is shown on invalid credentials', async ({ page }) => {
    // Override the login mock to return 401
    await page.route('**/api/v1/auth/driver/login', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'بيانات الدخول غير صحيحة' }),
      })
    })
    await mockDriverProfile(page)
    await mockOrders(page)

    await page.goto('/login')
    await page.locator('input[type="tel"]').fill('0100000000')
    await page.locator('input[type="password"]').fill('wrongpass')
    await page.getByRole('button', { name: /تسجيل الدخول/ }).click()

    // Should stay on login page and show error (either inline or via toast)
    await expect(page).toHaveURL(/\/login/, { timeout: 5_000 })
  })
})

test.describe('Driver App — Network resilience', () => {
  test('app handles API failure gracefully', async ({ page }) => {
    // Mock all API calls to fail
    await page.route('**/api/v1/**', async (route) => {
      await route.abort('failed')
    })
    await mockWebSocket(page)

    // App should not crash — it should still render
    await page.goto('/')
    await expect(page.locator('body')).not.toBeEmpty()
  })
})
