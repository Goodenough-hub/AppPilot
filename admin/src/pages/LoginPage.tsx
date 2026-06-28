import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import Logo from '../components/Logo'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(username, password)
      navigate('/admin')
    } catch (err: any) {
      setError(err.response?.data?.error || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', minHeight: '100vh', alignItems: 'center', justifyContent: 'center', position: 'relative' }}>
      <div style={{ position: 'absolute', width: 400, height: 400, background: 'var(--primary)', filter: 'blur(120px)', opacity: 0.2, borderRadius: '50%' }}></div>
      <form onSubmit={handleSubmit} className="glass-panel animate-fade-in-up" style={{ width: 380, padding: '40px 32px', position: 'relative', zIndex: 1 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{
            boxShadow: 'var(--shadow-glow)',
            marginBottom: 16,
            display: 'inline-flex',
            borderRadius: 12
          }}>
            <Logo size={48} />
          </div>
          <h1 style={{ fontSize: 24, margin: 0, fontFamily: 'Outfit, sans-serif' }}>欢迎回来</h1>
          <div style={{ color: 'var(--text-secondary)', fontSize: 14, marginTop: 8 }}>登录 AppPilot 管理后台</div>
        </div>
        {error && <div style={{ color: '#FCA5A5', background: 'var(--danger-bg)', padding: '10px 14px', borderRadius: 8, marginBottom: 20, fontSize: 13, border: '1px solid rgba(239, 68, 68, 0.2)' }}>{error}</div>}
        <div style={{ marginBottom: 16 }}>
          <label style={{ display: 'block', marginBottom: 8, color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>用户名</label>
          <input value={username} onChange={e => setUsername(e.target.value)} autoFocus required placeholder="请输入用户名" />
        </div>
        <div style={{ marginBottom: 24 }}>
          <label style={{ display: 'block', marginBottom: 8, color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>密码</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} required placeholder="请输入密码" />
        </div>
        <button type="submit" className="primary" disabled={loading} style={{ width: '100%', padding: '12px', fontSize: 15 }}>
          {loading ? '登录中…' : '登录'}
        </button>
      </form>
    </div>
  )
}
