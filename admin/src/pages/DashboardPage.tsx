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
    { label: '总用户数', value: stats.totalUsers, color: 'var(--primary)', gradient: 'linear-gradient(135deg, #6366F1, #A855F7)', delay: 'stagger-1' },
    { label: '管理员', value: stats.admins, color: 'var(--warning)', gradient: 'linear-gradient(135deg, #F59E0B, #EF4444)', delay: 'stagger-2' },
    { label: '普通用户', value: stats.regularUsers, color: 'var(--success)', gradient: 'linear-gradient(135deg, #10B981, #3B82F6)', delay: 'stagger-3' },
    { label: '总交易数', value: stats.totalTransactions, color: 'var(--danger)', gradient: 'linear-gradient(135deg, #EC4899, #F43F5E)', delay: 'stagger-4' }
  ]

  return (
    <div className="animate-fade-in-up">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 32 }}>
        <div>
          <h1 style={{ fontSize: 28, margin: 0 }}>系统概览</h1>
          <div style={{ color: 'var(--text-secondary)', fontSize: 14, marginTop: 4 }}>欢迎回来，这里是系统实时运行数据。</div>
        </div>
      </div>
      
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 24, marginBottom: 40 }}>
        {cards.map((c) => (
          <div key={c.label} className={`glass-panel animate-fade-in-up ${c.delay}`} style={{ padding: '24px' }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
               <div style={{ width: 8, height: 8, borderRadius: '50%', background: c.gradient, marginRight: 8, boxShadow: `0 0 10px ${c.color}` }}></div>
               <div style={{ color: 'var(--text-secondary)', fontSize: 13, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{c.label}</div>
            </div>
            <div style={{ 
              fontSize: 36, 
              fontWeight: 700, 
              background: c.gradient,
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
              fontFamily: 'Outfit, sans-serif'
            }}>{c.value}</div>
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 16 }}>
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
                    <Link to={`/admin/users/${u.id}`} style={{ 
                      padding: '6px 12px', 
                      background: 'var(--surface-hover)', 
                      borderRadius: 6,
                      fontSize: 12
                    }}>详情</Link>
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
