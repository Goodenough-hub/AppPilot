import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'

export default function LoginPage() {
  const { login } = useAuth()
  const nav = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setErr(null)
    setLoading(true)
    try {
      await login(username, password)
      nav('/', { replace: true })
    } catch (e: any) {
      setErr(e?.response?.data?.error ?? e?.message ?? '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 32 }}>
      <div style={{ width: '100%', maxWidth: 360 }}>
        <h1 className="font-serif italic" style={{ fontSize: 'var(--fs-xl)', margin: 0, marginBottom: 32, textAlign: 'center' }}>Hub</h1>
        <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="用户名" autoFocus required />
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="密码" required />
          {err && <div style={{ color: 'var(--accent)', fontSize: 'var(--fs-xs)' }}>{err}</div>}
          <button type="submit" className="btn-primary" disabled={loading}>{loading ? 'loading…' : '登录'}</button>
        </form>
        <div style={{ marginTop: 32, textAlign: 'center', fontSize: 'var(--fs-xs)', color: 'var(--ink-dim)', fontFamily: 'var(--font-mono)' }}>
          hub v0.1 · 授权用户可用
        </div>
      </div>
    </div>
  )
}