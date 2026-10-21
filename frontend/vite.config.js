import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Override when running Vite outside the Compose network, for example
// API_PROXY_TARGET=http://127.0.0.1:8080 npm run dev
const apiTarget = process.env.API_PROXY_TARGET || 'http://api:8080'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': apiTarget,
      '/healthz': apiTarget,
    },
  },
})
