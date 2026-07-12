// web-wlsg Vite 开发与同源 API 代理配置。
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    // 明确使用 Hero3 当前监听的 IPv6 本机地址，避免其他项目占用 IPv4 8080 时代理串线。
    proxy: { '/api': 'http://[::1]:8080' },
  },
})
