import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [svelte()],
  root: __dirname,
  publicDir: 'public',
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:18910',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:18910',
        changeOrigin: true,
      },
      '/oauth': {
        target: 'http://localhost:18910',
        changeOrigin: true,
      },
      '/login': {
        target: 'http://localhost:18910',
        changeOrigin: true,
      },
      '/logout': {
        target: 'http://localhost:18910',
        changeOrigin: true,
      },
      '/static': {
        target: 'http://localhost:18910',
        changeOrigin: true,
      },
    },
  },
  base: '/static/app/',
  build: {
    outDir: path.resolve(__dirname, '../static/app'),
    emptyOutDir: true,
    assetsDir: 'assets',
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
});
