import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // 关键配置：将所有 /api 开头的请求，代理到后端的 8080 端口
      '/api': {
        target: 'http://localhost:8080', // 你的 Go 后端服务地址
        changeOrigin: true, // 需要改变源
      },
    }
  }
})
