import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api/auth': {
        target: 'http://auth-back:8080',
        changeOrigin: true,
      },
      '/api/note': {
        target: 'http://note-back:8080',
        changeOrigin: true,
      },
      '/api/catalog': {
        target: 'http://catalog-server:8080',
        changeOrigin: true,
      },
    }
  }
})
