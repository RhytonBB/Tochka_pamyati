import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/uploads': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/thumbnails': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/previews': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/originals': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/signal_thumbnails': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/signal_previews': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/signal_originals': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      }
    }
  }
})
