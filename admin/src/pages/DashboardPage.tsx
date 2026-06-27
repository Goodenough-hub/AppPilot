import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getStats, listUsers, type AdminStats, type User } from '../api/admin'

export default function DashboardPage() {
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [users, setUsers] = useState<User[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([getStats(), listUsers()])
      .then(([s, u]) => {
        setStats(s)
        setUsers(u)
      })
      .catch(err => setError(err.response?.data?.error || '加载失败'))
  }, [])

  if (error) return <div style={{ color: 'var(--danger)' }}>{error}</div>
  if (!stats) return <div>加载中…</div>

  const cards = [
    { label: '总用户数', value: stats.totalUsers, color: 'var(--primary)' },
    { label: '管理员', value: stats.admins, color: 'var(--warning)' },
    { label: '普通用户', value: stats.regularUsers, color: 'var(--success)' },
    { label: '总交易数', value: stats.totalTransactions, color: 'var(--danger)' }
  ]

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 24 }}>概览</h1>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 32 }}>
        {cards.map(c => (
          <div key={c.label} style={{ background: 'var(--surface)', padding: 20, borderRadius: 8 }}>
            <div style={{ color: 'var(--text-dim)', fontSize: 13, marginBottom: 8 }}>{c.label}</div>
            <div style={{ fontSize: 28, fontWeight: 600, color: c.color }}>{c.value}</div>
          </div>
        ))}
      </div>
      <h2 style={{ fontSize: 16, marginBottom: 12 }}>最近用户</h2>
      <div style={{ background: 'var(--surface)', borderRadius: 8, overflow: 'hidden' }}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>角色</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {users.slice(-10).reverse().map(u => (
              <tr key={u.id}>
                <td>{u.id}</td>
                <td>{u.username}</td>
                <td>{u.role === 'admin' ? '管理员' : '用户'}</td>
                <td>{new Date(u.createdAt).toLocaleString('zh-CN')}</td>
                <td><Link to={`/admin/users/${u.id}`}>查看</Link></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
