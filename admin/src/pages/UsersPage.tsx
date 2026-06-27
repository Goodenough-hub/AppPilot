import { useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { createUser, deleteUser, listUsers, type User } from '../api/admin'

interface Form {
  username: string
  password: string
  role: 'user' | 'admin'
  appScope: string
}

const initialForm: Form = { username: '', password: '', role: 'user', appScope: 'finflow' }

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [form, setForm] = useState<Form>(initialForm)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const load = () => {
    listUsers().then(setUsers).catch(err => setError(err.response?.data?.error || '加载失败'))
  }

  useEffect(load, [])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await createUser({
        username: form.username,
        password: form.password,
        role: form.role,
        appScope: form.appScope.split(',').map(s => s.trim()).filter(Boolean)
      })
      setForm(initialForm)
      load()
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
      load()
    } catch (err: any) {
      setError(err.response?.data?.error || '删除失败')
    }
  }

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 24 }}>用户管理</h1>
      {error && <div style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</div>}

      <form onSubmit={submit} style={{ background: 'var(--surface)', padding: 16, borderRadius: 8, marginBottom: 24, display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
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
        <div>
          <label style={{ display: 'block', marginBottom: 4, color: 'var(--text-dim)', fontSize: 12 }}>应用范围（逗号分隔）</label>
          <input value={form.appScope} onChange={e => setForm({ ...form, appScope: e.target.value })} />
        </div>
        <div style={{ gridColumn: '1 / -1' }}>
          <button type="submit" className="primary" disabled={loading}>创建用户</button>
        </div>
      </form>

      <div style={{ background: 'var(--surface)', borderRadius: 8, overflow: 'hidden' }}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>角色</th>
              <th>应用范围</th>
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
                <td>{u.appScope.join(', ')}</td>
                <td>{new Date(u.createdAt).toLocaleString('zh-CN')}</td>
                <td>
                  <Link to={`/admin/users/${u.id}`}>查看</Link>
                  <button className="danger" style={{ marginLeft: 8 }} onClick={() => remove(u.id, u.username)}>删除</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
