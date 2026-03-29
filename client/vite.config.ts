import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { PrimeVueResolver } from "@primevue/auto-import-resolver";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  const DEV_PORT = Number(env.VITE_DEV_PORT) || 5000;
  const API_PROXY_TARGET = env.VITE_API_PROXY_TARGET || "http://localhost:2000";

  return {
    plugins: [
      vue(),
      Components({
        resolvers: [PrimeVueResolver()],
      }),
    ],
    build: {
      chunkSizeWarningLimit: 800,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes("node_modules/chart.js") ||
                id.includes("node_modules/vue-chart-3") ||
                id.includes("node_modules/chartjs-") ||
                id.includes("node_modules/date-fns")) {
              return "charts";
            }
            if (id.includes("node_modules/primevue") ||
                id.includes("node_modules/@primevue") ||
                id.includes("node_modules/primeicons")) {
              return "primevue";
            }
            if (id.includes("node_modules/vue") ||
                id.includes("node_modules/vue-router") ||
                id.includes("node_modules/pinia")) {
              return "vue-vendor";
            }
            if (id.includes("node_modules/")) {
              return "vendor";
            }
          },
        },
      },
    },
    server: {
      host: true,
      port: DEV_PORT,
      strictPort: true,
      proxy: {
        "/api": {
          target: API_PROXY_TARGET,
          changeOrigin: true,
        },
      },
    },
  };
});
