import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  base: '/',
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3019,
    proxy: {
      '/api': 'http://localhost:8019',
      '/ws': {
        target: 'ws://localhost:8019',
        ws: true,
      },
      '/static': 'http://localhost:8019',
    },
  },
})
