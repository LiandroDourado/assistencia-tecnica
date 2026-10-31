import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  // Em desenvolvimento (npm run dev), a API é o servidor Go na porta 8089.
  server: { proxy: { '/api': 'http://127.0.0.1:8089' } }
});
