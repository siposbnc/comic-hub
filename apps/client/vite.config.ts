import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Vite config tuned for Tauri: fixed dev port, no clearScreen so Rust logs stay visible.
export default defineConfig({
  plugins: [react()],
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
  },
  build: {
    target: 'es2022',
    sourcemap: true,
  },
});
