import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite';
import {PrimeVueResolver} from '@primevue/auto-import-resolver';

export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, process.cwd(), '')

    const DEV_PORT = Number(env.VITE_DEV_PORT)
    const API_PROXY_TARGET = env.VITE_API_PROXY_TARGET

    return {
        plugins: [
            vue(),
            Components({
                resolvers: [PrimeVueResolver()]
            }),
        ],
        server: {
            host: true,
            port: DEV_PORT,
            strictPort: true,
            proxy: {
                '/api': {
                    target: API_PROXY_TARGET,
                    changeOrigin: true,
                },
            },
        }
    }
})
