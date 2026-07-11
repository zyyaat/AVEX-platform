// =============================================================================
// AVEX Driver App — Playwright E2E config
// =============================================================================
// Runs end-to-end tests against:
//   - A locally running dev server (default), OR
//   - A preview server started from the built bundle.
//
// Usage:
//   pnpm --filter driver test:e2e            # run all E2E tests
//   pnpm --filter driver test:e2e:ui         # interactive UI mode
//   pnpm --filter driver test:e2e:install    # install chromium browser
// =============================================================================

import { defineConfig, devices } from '@playwright/test'

const PORT = process.env.PLAYWRIGHT_PORT || '4173'
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL || `http://localhost:${PORT}`

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI
    ? [['github'], ['html', { open: 'never' }]]
    : 'list',
  timeout: 30_000,
  expect: { timeout: 5_000 },

  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Auto-start a preview server from the production build.
  // For dev-server tests, run `pnpm --filter driver dev` first and set
  // PLAYWRIGHT_BASE_URL=http://localhost:5173.
  webServer: process.env.PLAYWRIGHT_EXTERNAL_SERVER
    ? undefined
    : {
        command: `pnpm build && pnpm serve --port ${PORT}`,
        url: BASE_URL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
})
