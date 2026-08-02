import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // Route each prefix directly to its own microservice.
      // This lets the frontend work in local dev without the api-gateway
      // (port 8080) running. Start each service individually instead:
      //   auth-service        → :8081
      //   claim-service       → :8082
      //   notification-service→ :8083  (HTTP + WebSocket)
      //   ai-service-py       → :8090  (or ai-service-go on :8093)
      '/auth': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/claims': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/notifications': {
        target: 'http://localhost:8083',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8083',
        ws: true,
        changeOrigin: true,
      },
      '/ai': {
        target: 'http://localhost:8093',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          charts: ['recharts'],
          table: ['@tanstack/react-table'],
          query: ['@tanstack/react-query'],
        },
      },
    },
  },
})
