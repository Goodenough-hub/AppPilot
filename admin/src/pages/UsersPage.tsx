import { useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import {
  createUser, deleteUser, getStats, listApps, listUsers,
  type AdminStats, type User
} from '../api/admin'

interface Form {
  username: string
  password: string
  role: 'user' | 'admin'
}

const initialForm: Form = { username: '', password: '', role: 'user' }

export default function UsersPage() {
  const [apps, setApps] = useState<string[]>([])
  const [app, setApp] = useState<string>('')
  const [users, setUsers] = useState<User[]>([])
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [form, setForm] = useState<Form>(initialForm)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [newUserId, setNewUserId] = useState<string | null>(null)
  const [leavingIds, setLeavingIds] = useState<Set<string>>(new Set())

  useEffect(() => {
    listApps()
      .then(list => {
        setApps(list)
        if (list.length > 0 && !list.includes(app)) {
          setApp(list[0])
        }
      })
      .catch(err => setError(err.response?.data?.error || '加载应用列表失败'))
  }, [])

  useEffect(() => {
    if (!app) return
    Promise.all([listUsers(app), getStats(app)])
      .then(([u, s]) => { setUsers(u); setStats(s) })
      .catch(err => setError(err.response?.data?.error || '加载失败'))
  }, [app])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!app) {
      setError('请先选择应用')
      return
    }
    setError('')
    setLoading(true)
    try {
      const created = await createUser({
        username: form.username,
        password: form.password,
        role: form.role,
        appScope: [app]
      })
      setForm(initialForm)
      const [u, s] = await Promise.all([listUsers(app), getStats(app)])
      setUsers(u); setStats(s)
      if (created?.id) {
        setNewUserId(String(created.id))
        setTimeout(() => setNewUserId(null), 2000)
      }
    } catch (err: any) {
      setError(err.response?.data?.error || '创建失败')
    } finally {
      setLoading(false)
    }
  }

  const remove = async (id: string, username: string) => {
    if (!confirm(`确认删除用户 ${username}？所有数据将一并删除。`)) return
    setLeavingIds(prev => new Set(prev).add(id))
    try {
      await deleteUser(id)
      // 等淡出动画结束再移除行
      setTimeout(async () => {
        const [u, s] = await Promise.all([listUsers(app), getStats(app)])
        setUsers(u); setStats(s)
        setLeavingIds(prev => {
          const next = new Set(prev)
          next.delete(id)
          return next
        })
      }, 250)
    } catch (err: any) {
      setLeavingIds(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
      setError(err.response?.data?.error || '删除失败')
    }
  }

  const cards = stats ? [
    { label: '用户数', value: stats.totalUsers },
    { label: '总交易数', value: stats.totalTransactions },
    { label: '本周活跃', value: stats.activeThisWeek ?? 0 },
    { label: '管理员', value: stats.admins }
  ] : []

  return (
    <div className="animate-fade-in-up">
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-end',
        marginBottom: 40
      }}>
        <div>
          <h1 style={{
            fontSize: 32,
            margin: 0,
            marginBottom: 8
          }}>用户管理</h1>
          <p style={{ color: 'var(--text-secondary)', fontSize: 14, margin: 0 }}>
            管理接入应用的用户账户
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>应用</span>
          <select
            value={app}
            onChange={e => setApp(e.target.value)}
            style={{ width: 160 }}
          >
            {apps.map(a => <option key={a} value={a}>{a}</option>)}
          </select>
        </div>
      </div>

      {error && (
        <div style={{
          color: '#FCA5A5',
          background: 'var(--danger-bg)',
          padding: '12px 16px',
          borderRadius: 12,
          marginBottom: 24,
          fontSize: 14,
          border: '1px solid rgba(239, 68, 68, 0.2)'
        }}>{error}</div>
      )}

      {cards.length > 0 && (
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 24,
          marginBottom: 40
        }}>
          {cards.map((c, i) => (
            <div key={c.label} className={`glass-panel animate-fade-in-up stagger-${i+1}`} style={{ padding: '24px' }}>
              <div style={{
                color: 'var(--text-secondary)',
                fontSize: 13,
                marginBottom: 12,
                textTransform: 'uppercase',
                letterSpacing: '0.05em'
              }}>{c.label}</div>
              <div style={{
                fontSize: 32,
                fontWeight: 700,
                color: 'var(--text-primary)',
                fontFamily: 'Outfit, sans-serif'
              }}>{c.value}</div>
            </div>
          ))}
        </div>
      )}

      <div className="glass-panel animate-fade-in-up stagger-2" style={{ marginBottom: 32, padding: 32 }}>
        <div style={{ marginBottom: 24 }}>
          <h2 style={{ fontSize: 20, margin: 0, marginBottom: 8 }}>创建新用户</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: 13, margin: 0 }}>
            新用户将自动绑定到当前应用「<span style={{color: 'var(--primary)'}}>{app || '—'}</span>」
          </p>
        </div>
        <form onSubmit={submit}>
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr 1fr',
            gap: 24,
            marginBottom: 24
          }}>
            <div>
              <label style={{ display: 'block', marginBottom: 8, color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>用户名</label>
              <input
                value={form.username}
                onChange={e => setForm({ ...form, username: e.target.value })}
                required
                minLength={3}
                placeholder="至少 3 个字符"
              />
            </div>
            <div>
              <label style={{ display: 'block', marginBottom: 8, color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>密码</label>
              <input
                type="password"
                value={form.password}
                onChange={e => setForm({ ...form, password: e.target.value })}
                required
                minLength={6}
                placeholder="至少 6 个字符"
              />
            </div>
            <div>
              <label style={{ display: 'block', marginBottom: 8, color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>角色</label>
              <select
                value={form.role}
                onChange={e => setForm({ ...form, role: e.target.value as 'user' | 'admin' })}
              >
                <option value="user">用户</option>
                <option value="admin">管理员</option>
              </select>
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <button
              type="submit"
              className="primary"
              disabled={loading || !app}
              title={app ? `自动绑定应用：${app}` : '请先选择应用'}
              style={{ minWidth: 140 }}
            >
              {loading ? '创建中...' : '创建用户'}
            </button>
          </div>
        </form>
      </div>

      <div className="glass-panel animate-fade-in-up stagger-3" style={{ overflow: 'hidden' }}>
        <div style={{ padding: '24px 32px' }}>
          <h2 style={{ fontSize: 20, margin: 0 }}>{app ? `${app} 用户` : '用户列表'}</h2>
        </div>
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>用户名</th>
                <th>角色</th>
                <th style={{ textAlign: 'right' }}>交易数</th>
                <th>最近活跃</th>
                <th>创建时间</th>
                <th style={{ textAlign: 'right' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => {
                const isNew = u.id === newUserId
                const rowClass = [
                  isNew ? 'animate-fade-in-up' : ''
                ].filter(Boolean).join(' ')
                return (
                <tr key={u.id} className={rowClass} style={{ opacity: leavingIds.has(u.id) ? 0.3 : 1, transition: 'opacity 0.2s' }}>
                  <td style={{ fontWeight: 500 }}>{u.username}</td>
                  <td>
                    <span className={u.role === 'admin' ? 'badge badge-admin' : 'badge badge-user'}>
                      {u.role === 'admin' ? '管理员' : '用户'}
                    </span>
                  </td>
                  <td style={{ textAlign: 'right', fontFamily: 'Outfit, sans-serif' }}>
                    {u.stats?.transactionCount ?? 0}
                  </td>
                  <td style={{ color: 'var(--text-tertiary)', fontSize: 13 }}>
                    {u.stats?.lastActiveAt ? new Date(u.stats.lastActiveAt).toLocaleString('zh-CN') : '—'}
                  </td>
                  <td style={{ color: 'var(--text-tertiary)', fontSize: 13 }}>
                    {new Date(u.createdAt).toLocaleString('zh-CN')}
                  </td>
                  <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                    <Link to={`/admin/users/${u.id}`} style={{ 
                      marginRight: 16,
                      padding: '6px 12px', 
                      background: 'var(--surface-hover)', 
                      borderRadius: 6,
                      fontSize: 12
                    }}>查看</Link>
                    <button
                      className="danger"
                      style={{ padding: '5px 14px', fontSize: 12 }}
                      onClick={() => remove(u.id, u.username)}
                      disabled={leavingIds.has(u.id)}
                    >删除</button>
                  </td>
                </tr>
                )
              })}
              {users.length === 0 && (
                <tr>
                  <td colSpan={6} style={{ textAlign: 'center', padding: 48, color: 'var(--text-tertiary)', fontSize: 14 }}>该应用暂无用户</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
