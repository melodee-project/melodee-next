import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0', // Listen on all interfaces (IPv4 and IPv6)
    proxy: {
      '/api': {
        target: 'https://localhost:8080',
        changeOrigin: true,
        secure: false, // Set to false to allow self-signed certificates
      },
      '/healthz': {
        target: 'https://localhost:8080',
        changeOrigin: true,
        secure: false, // Set to false to allow self-signed certificates
      },
      '/metrics': {
        target: 'https://localhost:8080',
        changeOrigin: true,
        secure: false, // Set to false to allow self-signed certificates
      },
    },
  },
  build: {
    outDir: 'dist',
  },
})