import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { apiClient, tokenStorage } from '@/api/client'

// Hub access rule (mirrors backend AppScopeRequired): scope contains "hub",
// or role is admin (the backend grants the "admin" pseudo-scope pass-through).
export function canAccessHub(appScope: string[] | null, role: string | null): boolean {
  return role === 'admin' || (appScope?.includes('hub') ?? false)
}

interface AuthState {
  token: string | null
  role: string | null
  appScope: string[] | null
  username: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

const Ctx = createContext<AuthState>({ token: null, role: null, appScope: null, username: null, login: async () => {}, logout: () => {} })

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(tokenStorage.get())
  const [role, setRole] = useState<string | null>(null)
  const [appScope, setAppScope] = useState<string[] | null>(null)
  const [username, setUsername] = useState<string | null>(null)

  // On mount, decode JWT payload to extract role/scope/username for display (no signature verification)
  useEffect(() => {
    if (!token) return
    try {
      // JWT base64url 解码：换掉 - _ 并补 = padding；atob 只吃标准 base64
      const b64 = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
      const padded = b64 + '='.repeat((4 - b64.length % 4) % 4)
      const payload = JSON.parse(atob(padded))
      setRole(payload.role ?? null)
      setAppScope(payload.scope ?? null)
      setUsername(payload.usr ?? null)
    } catch { /* invalid token 结构 */ }
  }, [token])

  const login = useCallback(async (u: string, p: string) => {
    const { data } = await apiClient.post('/auth/login', { username: u, password: p })
    if (!canAccessHub(data.appScope ?? null, data.role ?? null)) {
      throw new Error('未授权访问 Hub')
    }
    tokenStorage.set(data.token)
    setToken(data.token)
    setRole(data.role)
    setAppScope(data.appScope ?? null)
    setUsername(data.username)
  }, [])

  const logout = useCallback(() => {
    tokenStorage.clear()
    setToken(null)
    setRole(null)
    setAppScope(null)
    setUsername(null)
    window.location.href = '/hub/login'
  }, [])

  return <Ctx.Provider value={{ token, role, appScope, username, login, logout }}>{children}</Ctx.Provider>
}

export const useAuth = () => useContext(Ctx)
