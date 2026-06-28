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
      await createUser({
        username: form.username,
        password: form.password,
        role: form.role,
        appScope: [app]
      })
      setForm(initialForm)
      const [u, s] = await Promise.all([listUsers(app), getStats(app)])
      setUsers(u); setStats(s)
    } catch (err: any) {
      setError(err.response?.data?.error || '创建失败')
    } finally {
      setLoading(false)
    }
  }

  const remove = async (id: string, username: string) => {
    if (!confirm(`确认删除用户 ${username}？所有数据将一并删除。`)) return
    try {
      await deleteUser(id)
      const [u, s] = await Promise.all([listUsers(app), getStats(app)])
      setUsers(u); setStats(s)
    } catch (err: any) {
      setError(err.response?.data?.error || '删除失败')
    }
  }

  const cards = stats ? [
    { label: '用户数', value: stats.totalUsers, color: 'var(--primary)' },
    { label: '总交易数', value: stats.totalTransactions, color: 'var(--danger)' },
    { label: '本周活跃', value: stats.activeThisWeek ?? 0, color: 'var(--success)' },
    { label: '管理员', value: stats.admins, color: 'var(--warning)' }
  ] : []

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h1 style={{ fontSize: 22 }}>用户管理</h1>
        <div>
          <label style={{ marginRight: 8, color: 'var(--text-dim)', fontSize: 13 }}>应用：</label>
          <select value={app} onChange={e => setApp(e.target.value)} style={{ minWidth: 140 }}>
            {apps.map(a => <option key={a} value={a}>{a}</option>)}
          </select>
        </div>
      </div>
      {error && <div style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</div>}

      {cards.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 24 }}>
          {cards.map(c => (
            <div key={c.label} style={{ background: 'var(--surface)', padding: 20, borderRadius: 8 }}>
              <div style={{ color: 'var(--text-dim)', fontSize: 13, marginBottom: 8 }}>{c.label}</div>
              <div style={{ fontSize: 28, fontWeight: 600, color: c.color }}>{c.value}</div>
            </div>
          ))}
        </div>
      )}

      <form onSubmit={submit} style={{ background: 'var(--surface)', padding: 16, borderRadius: 8, marginBottom: 24, display: 'grid', gridTemplateColumns: 'repeat(3, 1fr) auto', gap: 12, alignItems: 'end' }}>
        <div>
          <label style={{ display: 'block', marginBottom: 4, color: 'var(--text-dim)', fontSize: 12 }}>用户名</label>
          <input value={form.username} onChange={e => setForm({ ...form, username: e.target.value })} required minLength={3} />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 4, color: 'var(--text-dim)', fontSize: 12 }}>密码</label>
          <input value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} required minLength={6} />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 4, color: 'var(--text-dim)', fontSize: 12 }}>角色</label>
          <select value={form.role} onChange={e => setForm({ ...form, role: e.target.value as 'user' | 'admin' })}>
            <option value="user">用户</option>
            <option value="admin">管理员</option>
          </select>
        </div>
        <button type="submit" className="primary" disabled={loading || !app} title={app ? `自动绑定应用：${app}` : '请先选择应用'}>创建用户</button>
        <div style={{ gridColumn: '1 / -1', color: 'var(--text-dim)', fontSize: 12 }}>
          新用户将自动绑定到当前应用「{app || '—'}」
        </div>
      </form>

      <div style={{ background: 'var(--surface)', borderRadius: 8, overflow: 'hidden' }}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>角色</th>
              <th>交易数</th>
              <th>最近活跃</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {users.map(u => (
              <tr key={u.id}>
                <td>{u.id}</td>
                <td>{u.username}</td>
                <td>{u.role === 'admin' ? '管理员' : '用户'}</td>
                <td>{u.stats?.transactionCount ?? 0}</td>
                <td>{u.stats?.lastActiveAt ? new Date(u.stats.lastActiveAt).toLocaleString('zh-CN') : '—'}</td>
                <td>{new Date(u.createdAt).toLocaleString('zh-CN')}</td>
                <td>
                  <Link to={`/admin/users/${u.id}`}>查看</Link>
                  <button className="danger" style={{ marginLeft: 8 }} onClick={() => remove(u.id, u.username)}>删除</button>
                </td>
              </tr>
            ))}
            {users.length === 0 && (
              <tr><td colSpan={7} style={{ textAlign: 'center', padding: 24, color: 'var(--text-dim)' }}>该应用暂无用户</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
