import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import dotenv from 'dotenv';
dotenv.config();

export default defineConfig({

  plugins: [vue()],
  server: {
    port: Number(process.env.VITE_APP_PORT) || 5173, // Use process.env directly
  },
})
