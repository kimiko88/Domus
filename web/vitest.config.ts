import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // VitePWA is intentionally excluded from the unit-test plugin graph.
      // This local shim resolves its virtual module without generating a service worker.
      'virtual:pwa-register': fileURLToPath(new URL('./src/test/mocks/pwa-register.ts', import.meta.url))
    }
  },
  test: {
    environment: 'happy-dom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    include: ['src/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}']
  }
})
