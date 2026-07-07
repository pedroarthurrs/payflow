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
        // Support WebSocket proxying (e.g. live-reload or future WS features)
        ws: true,
        // Protect against slow-loris and hung requests
        timeout: 30000,
        proxyTimeout: 30000,
        onProxyReq: (_proxyReq, req) => {
          // Log outgoing proxy requests in development for easier debugging
          console.log(`[Proxy] ${req.method} ${req.url}`)
        },
        onError: (err, _req, res) => {
          console.error('[Proxy Error]', err.message)
          if (!res.headersSent) {
            res.writeHead(503, { 'Content-Type': 'application/json' })
          }
          res.end(JSON.stringify({ error: 'API unavailable — backend may be down' }))
        },
      }
    }
  },
  test: {
    environment: 'jsdom',
    globals: true
  }
})
