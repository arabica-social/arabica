import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [svelte()],
  build: {
    emptyOutDir: false,
    lib: {
      entry: 'frontend/svelte/src/main.ts',
      formats: ['iife'],
      name: 'ArabicaSvelteIslands',
      fileName: () => 'svelte-islands.js'
    },
    outDir: 'internal/web/assets/js',
    rollupOptions: {
      output: {
        extend: true
      }
    }
  }
});
