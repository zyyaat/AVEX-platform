import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'happy-dom',
    environmentOptions: {
      happyDOM: {
        url: 'http://localhost:3000/',
      },
    },
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
