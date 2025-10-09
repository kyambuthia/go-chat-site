import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  // Load env file based on mode (development, production)
  const env = loadEnv(mode, process.cwd(), '');

  return {
    plugins: [react()],
      server: {
        host: '0.0.0.0',
        proxy: {        '/api': {
          target: env.VITE_API_URL || 'http://localhost:8080',
          changeOrigin: true,
        },
        '/ws': {
          target: env.VITE_WS_URL || 'ws://localhost:8080',
          ws: true,
        },
      },
    },
  };
});