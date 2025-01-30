import {defineConfig, loadEnv} from 'vite'
import vue from '@vitejs/plugin-vue'

export default ({ mode }) => {
  // Load environment variables
  const env = loadEnv(mode, process.cwd());

  // Determine if HTTPS should be enabled
  const isHttps = env.VITE_APP_PRODUCTION_MODE === 'true';

  return defineConfig({
    plugins: [vue()],
    build: {
      minify: 'esbuild',
      chunkSizeWarningLimit: 1600,
      rollupOptions: {
        output: {
          manualChunks: undefined, // Disable manualChunks to allow Vite to manage chunks
        },
      },
    },
    server: {
      port: parseInt(env.VITE_APP_FRONTEND_PORT),
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: "modern",
        },
      },
    },
  });
}
