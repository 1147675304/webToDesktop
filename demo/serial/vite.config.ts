import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: '/',
  server: { port: 5174 },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    target: 'es2015',
    cssTarget: 'safari12',
  }
})
