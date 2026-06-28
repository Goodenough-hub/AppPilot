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
    <div>
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-end',
        marginBottom: 40
      }}>
        <div>
          <h1 style={{
            fontSize: 40,
            fontWeight: 600,
            letterSpacing: '-0.025em',
            marginBottom: 6,
            lineHeight: 1.1
          }}>用户管理</h1>
          <p style={{ color: 'var(--text-dim)', fontSize: 15, letterSpacing: '-0.012em' }}>
            管理接入应用的用户账户
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span style={{ color: 'var(--text-dim)', fontSize: 13 }}>应用</span>
          <select
            value={app}
            onChange={e => setApp(e.target.value)}
            style={{
              minWidth: 160,
              padding: '8px 30px 8px 14px',
              borderRadius: 980,
              border: '1px solid var(--border)',
              background: 'var(--surface)',
              cursor: 'pointer'
            }}
          >
            {apps.map(a => <option key={a} value={a}>{a}</option>)}
          </select>
        </div>
      </div>

      {error && (
        <div style={{
          color: 'var(--danger)',
          background: 'rgba(255, 59, 48, 0.08)',
          padding: '12px 16px',
          borderRadius: 12,
          marginBottom: 24,
          fontSize: 14
        }}>{error}</div>
      )}

      {cards.length > 0 && (
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 16,
          marginBottom: 40
        }}>
          {cards.map(c => (
            <div key={c.label} style={{
              background: 'var(--surface)',
              padding: '28px 24px',
              borderRadius: 18,
              border: '1px solid var(--border-soft)',
              boxShadow: 'var(--shadow)'
            }}>
              <div style={{
                color: 'var(--text-dim)',
                fontSize: 13,
                marginBottom: 12,
                letterSpacing: '-0.005em'
              }}>{c.label}</div>
              <div style={{
                fontSize: 36,
                fontWeight: 600,
                color: 'var(--text)',
                letterSpacing: '-0.03em',
                fontVariantNumeric: 'tabular-nums'
              }}>{c.value}</div>
            </div>
          ))}
        </div>
      )}

      <div style={{
        background: 'var(--surface)',
        borderRadius: 20,
        border: '1px solid var(--border-soft)',
        boxShadow: 'var(--shadow)',
        marginBottom: 32,
        overflow: 'hidden'
      }}>
        <div style={{
          padding: '28px 32px 20px',
          borderBottom: '1px solid var(--border-soft)'
        }}>
          <h2 style={{
            fontSize: 22,
            fontWeight: 600,
            letterSpacing: '-0.022em',
            marginBottom: 4
          }}>创建新用户</h2>
          <p style={{
            color: 'var(--text-dim)',
            fontSize: 13,
            letterSpacing: '-0.005em'
          }}>
            新用户将自动绑定到当前应用「{app || '—'}」
          </p>
        </div>
        <form onSubmit={submit} style={{ padding: '24px 32px 32px' }}>
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr 1fr',
            gap: 20,
            marginBottom: 24
          }}>
            <div>
              <label style={{
                display: 'block',
                marginBottom: 8,
                color: 'var(--text)',
                fontSize: 13,
                fontWeight: 500,
                letterSpacing: '-0.005em'
              }}>用户名</label>
              <input
                value={form.username}
                onChange={e => setForm({ ...form, username: e.target.value })}
                required
                minLength={3}
                placeholder="至少 3 个字符"
                style={{ padding: '12px 16px', fontSize: 15 }}
              />
            </div>
            <div>
              <label style={{
                display: 'block',
                marginBottom: 8,
                color: 'var(--text)',
                fontSize: 13,
                fontWeight: 500,
                letterSpacing: '-0.005em'
              }}>密码</label>
              <input
                type="password"
                value={form.password}
                onChange={e => setForm({ ...form, password: e.target.value })}
                required
                minLength={6}
                placeholder="至少 6 个字符"
                style={{ padding: '12px 16px', fontSize: 15 }}
              />
            </div>
            <div>
              <label style={{
                display: 'block',
                marginBottom: 8,
                color: 'var(--text)',
                fontSize: 13,
                fontWeight: 500,
                letterSpacing: '-0.005em'
              }}>角色</label>
              <select
                value={form.role}
                onChange={e => setForm({ ...form, role: e.target.value as 'user' | 'admin' })}
                style={{ padding: '12px 16px', fontSize: 15, cursor: 'pointer' }}
              >
                <option value="user">用户</option>
                <option value="admin">管理员</option>
              </select>
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <button
              type="submit"
              className="primary apple-button-press"
              disabled={loading || !app}
              title={app ? `自动绑定应用：${app}` : '请先选择应用'}
              style={{ padding: '11px 28px', fontSize: 15, minWidth: 120 }}
            >
              {loading && <span className="apple-spinner" />}
              {loading ? '创建中' : '创建用户'}
            </button>
          </div>
        </form>
      </div>

      <div style={{
        background: 'var(--surface)',
        borderRadius: 20,
        border: '1px solid var(--border-soft)',
        boxShadow: 'var(--shadow)',
        overflow: 'hidden'
      }}>
        <div style={{
          padding: '24px 32px',
          borderBottom: '1px solid var(--border-soft)'
        }}>
          <h2 style={{
            fontSize: 19,
            fontWeight: 600,
            letterSpacing: '-0.022em'
          }}>{app ? `${app} 用户` : '用户列表'}</h2>
        </div>
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
              const isLeaving = leavingIds.has(u.id)
              const rowClass = [
                isNew ? 'apple-row-enter apple-row-highlight' : '',
                isLeaving ? 'apple-row-leaving' : ''
              ].filter(Boolean).join(' ')
              return (
              <tr key={u.id} className={rowClass}>
                <td style={{ fontWeight: 500 }}>{u.username}</td>
                <td>
                  <span style={{
                    display: 'inline-block',
                    padding: '2px 10px',
                    borderRadius: 980,
                    fontSize: 12,
                    fontWeight: 500,
                    background: u.role === 'admin' ? 'rgba(255, 149, 0, 0.12)' : 'var(--surface-2)',
                    color: u.role === 'admin' ? 'var(--warning)' : 'var(--text-dim)'
                  }}>
                    {u.role === 'admin' ? '管理员' : '用户'}
                  </span>
                </td>
                <td style={{ textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
                  {u.stats?.transactionCount ?? 0}
                </td>
                <td style={{ color: 'var(--text-dim)', fontSize: 13 }}>
                  {u.stats?.lastActiveAt ? new Date(u.stats.lastActiveAt).toLocaleString('zh-CN') : '—'}
                </td>
                <td style={{ color: 'var(--text-dim)', fontSize: 13 }}>
                  {new Date(u.createdAt).toLocaleString('zh-CN')}
                </td>
                <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                  <Link to={`/admin/users/${u.id}`} style={{ marginRight: 16 }}>查看</Link>
                  <button
                    className="danger apple-button-press"
                    style={{ padding: '5px 14px', fontSize: 13 }}
                    onClick={() => remove(u.id, u.username)}
                    disabled={isLeaving}
                  >删除</button>
                </td>
              </tr>
              )
            })}
            {users.length === 0 && (
              <tr>
                <td colSpan={6} style={{
                  textAlign: 'center',
                  padding: 48,
                  color: 'var(--text-dim)',
                  fontSize: 14
                }}>该应用暂无用户</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
