import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getStats, listUsers, type AdminStats, type User } from '../api/admin'
import StatCard from '../components/ui/StatCard'
import { UserGrowthChart, RoleDonut } from '../components/DashboardCharts'

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
    { label: '总用户数', value: stats.totalUsers, glow: '#6366F1', gradient: 'var(--accent-indigo)' },
    { label: '管理员', value: stats.admins, glow: '#F59E0B', gradient: 'var(--accent-amber)' },
    { label: '普通用户', value: stats.regularUsers, glow: '#10B981', gradient: 'var(--accent-emerald)' },
    { label: '总交易数', value: stats.totalTransactions, glow: '#EC4899', gradient: 'var(--accent-pink)' }
  ]

  return (
    <div className="animate-fade-in-up">
      <header className="admin-page-header">
        <div>
          <h1>系统概览</h1>
          <div className="subtitle">欢迎回来，这里是系统实时运行数据。</div>
        </div>
      </header>

      <div className="stat-grid">
        {cards.map((c, i) => (
          <StatCard
            key={c.label}
            label={c.label}
            value={c.value}
            gradient={c.gradient}
            glow={c.glow}
            gradientValue
            className={`animate-fade-in-up stagger-${i + 1}`}
          />
        ))}
      </div>

      <div className="charts-row">
        <UserGrowthChart users={users} />
        <RoleDonut admins={stats.admins} regular={stats.regularUsers} />
      </div>

      <div className="admin-page-header" style={{ marginBottom: 16 }}>
        <h2 style={{ fontSize: 20, margin: 0 }}>最近用户</h2>
        <Link to="/admin/users" style={{ fontSize: 13, color: 'var(--primary)', fontWeight: 500 }}>查看全部 &rarr;</Link>
      </div>

      <div className="glass-panel animate-fade-in-up stagger-4" style={{ overflow: 'hidden' }}>
        <div className="table-container">
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
                  <td style={{ color: 'var(--text-tertiary)', fontSize: 13 }}>#{u.id}</td>
                  <td>
                    <div style={{ fontWeight: 500, color: 'var(--text-primary)' }}>{u.username}</div>
                  </td>
                  <td>
                    <span className={u.role === 'admin' ? 'badge badge-admin' : 'badge badge-user'}>
                      {u.role === 'admin' ? '管理员' : '用户'}
                    </span>
                  </td>
                  <td style={{ color: 'var(--text-secondary)' }}>{new Date(u.createdAt).toLocaleString('zh-CN')}</td>
                  <td>
                    <Link to={`/admin/users/${u.id}`} className="pill-link">详情</Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
