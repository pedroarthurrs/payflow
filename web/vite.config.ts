/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, type PluginOption } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), ...(tailwindcss() as unknown as PluginOption[])],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 3000,
    host: true,
      allowedHosts: ['luetta-overlaudatory-unausterely.ngrok-free.dev', 'localhost'],
      proxy: {
      '/api': {
        target: process.env.VITE_API_PROXY_TARGET || process.env.VITE_API_URL || 'http://localhost:8080',
        rewrite: (path: string) => path.replace(/^\/api/, ''),
        changeOrigin: true,
        // Timeout to prevent hanging requests and protect against slow-loris attacks
        timeout: 30000,
        proxyTimeout: 30000,
      }
    }
  },
  test: {
    environment: 'jsdom',
    globals: true
  }
})
