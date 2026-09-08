import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // The VITE_* settings live in the repository-root .env alongside the backend
  // configuration, so one file describes the whole stack.
  envDir: fileURLToPath(new URL('..', import.meta.url)),
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    host: true,
  },
  build: {
    // Route-level code splitting is handled by the router's dynamic imports;
    // these chunks keep the vendor bundle from being re-downloaded on deploys.
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue', 'vue-router', 'pinia'],
          http: ['axios'],
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
  },
})
