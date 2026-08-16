import axios from 'axios'

export const TOKEN_KEY = 'hub_token'

export const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' }
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

apiClient.interceptors.response.use(
  (r) => r,
  (err) => {
    const status = err.response?.status
    // 401 直接清 token 跳 login（Hub 内 login 是同域内部路由）
    if (status === 401) {
      localStorage.removeItem(TOKEN_KEY)
      if (!window.location.pathname.endsWith('/login')) {
        window.location.href = '/hub/login'
      }
    }
    return Promise.reject(err)
  }
)

export const tokenStorage = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY)
}