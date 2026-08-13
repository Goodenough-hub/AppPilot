import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { UserPlus, Search } from 'lucide-react'
import {
  deleteUser, getStats, listUsers,
  type AdminStats, type User
} from '../api/admin'
import { SUPPORTED_APPS } from '../lib/apps'
import StatCard from '../components/ui/StatCard'
import UserFormDrawer from '../components/UserFormDrawer'

type DrawerMode = 'create' | 'edit'

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [error, setError] = useState('')
  const [appFilter, setAppFilter] = useState<string>('') // '' = 全部
  const [search, setSearch] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerMode, setDrawerMode] = useState<DrawerMode>('create')
  const [editingUser, setEditingUser] = useState<User | undefined>(undefined)
  const [newUserId, setNewUserId] = useState<string | null>(null)
  const [leavingIds, setLeavingIds] = useState<Set<string>>(new Set())

  const refresh = async () => {
    try {
      const [u, s] = await Promise.all([listUsers(), getStats()])
      setUsers(u); setStats(s)
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '加载失败'
      setError(msg)
    }
  }

  useEffect(() => { refresh() }, [])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return users.filter(u => {
      if (appFilter && !u.appScope.includes(appFilter)) return false
      if (q && !u.username.toLowerCase().includes(q)) return false
      return true
    })
  }, [users, appFilter, search])

  const openCreate = () => {
    setDrawerMode('create')
    setEditingUser(undefined)
    setDrawerOpen(true)
  }

  const openEdit = (u: User) => {
    setDrawerMode('edit')
    setEditingUser(u)
    setDrawerOpen(true)
  }

  const onSaved = async (saved: User) => {
    setDrawerOpen(false)
    if (drawerMode === 'create' && saved?.id) {
      setNewUserId(String(saved.id))
      setTimeout(() => setNewUserId(null), 2000)
    }
    await refresh()
  }

  const remove = async (id: string, username: string) => {
    if (!confirm(`确认删除用户 ${username}？所有数据将一并删除。`)) return
    setLeavingIds(prev => new Set(prev).add(id))
    try {
      await deleteUser(id)
      toast.success(`已删除用户 ${username}`)
      setTimeout(async () => {
        await refresh()
        setLeavingIds(prev => {
          const next = new Set(prev)
          next.delete(id)
          return next
        })
      }, 250)
    } catch (err: unknown) {
      setLeavingIds(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '删除失败'
      setError(msg)
      toast.error(msg)
    }
  }

  const cards = stats ? [
    { label: '用户数', value: stats.totalUsers },
    { label: '总交易数', value: stats.totalTransactions },
    { label: '本周活跃', value: stats.activeThisWeek ?? 0 },
    { label: '管理员', value: stats.admins }
  ] : []

  const filterChips = [
    { code: '', name: '全部' },
    ...SUPPORTED_APPS,
  ]

  return (
    <div className="animate-fade-in-up">
      <header className="admin-page-header">
        <div>
          <h1 style={{ fontSize: 32 }}>用户管理</h1>
          <p className="subtitle">一个账号可同时授权多个应用</p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <button className="btn-primary" onClick={openCreate} type="button" style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
            <UserPlus size={16} />
            新建用户
          </button>
        </div>
      </header>

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
        <div className="stat-grid">
          {cards.map((c, i) => (
            <StatCard
              key={c.label}
              label={c.label}
              value={c.value}
              className={`animate-fade-in-up stagger-${i + 1}`}
            />
          ))}
        </div>
      )}

      <div className="glass-panel animate-fade-in-up stagger-3" style={{ overflow: 'hidden', marginTop: 24 }}>
        <div style={{ padding: '20px 32px', display: 'flex', gap: 16, alignItems: 'center', flexWrap: 'wrap', borderBottom: '1px solid var(--border-light)' }}>
          <div className="app-filter-chips">
            {filterChips.map(c => (
              <button
                key={c.code || 'all'}
                type="button"
                className={`chip${appFilter === c.code ? ' active' : ''}`}
                onClick={() => setAppFilter(c.code)}
              >
                {c.name}
              </button>
            ))}
          </div>
          <div className="users-search">
            <Search size={14} className="users-search-icon" />
            <input
              className="form-input users-search-input"
              placeholder="搜索用户名…"
              value={search}
              onChange={e => setSearch(e.target.value)}
            />
          </div>
          <div style={{ marginLeft: 'auto', color: 'var(--text-tertiary)', fontSize: 13 }}>
            共 {filtered.length} / {users.length}
          </div>
        </div>
        <div className="table-container">
          <table className="responsive-table">
            <thead>
              <tr>
                <th>用户名</th>
                <th>角色</th>
                <th>应用授权</th>
                <th style={{ textAlign: 'right' }}>交易数</th>
                <th>最近活跃</th>
                <th>创建时间</th>
                <th style={{ textAlign: 'right' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(u => {
                const isNew = u.id === newUserId
                const rowClass = [isNew ? 'animate-fade-in-up' : ''].filter(Boolean).join(' ')
                return (
                  <tr key={u.id} className={rowClass} style={{ opacity: leavingIds.has(u.id) ? 0.3 : 1, transition: 'opacity 0.2s' }}>
                    <td data-label="用户名" style={{ fontWeight: 500 }}>{u.username}</td>
                    <td data-label="角色">
                      <span className={u.role === 'admin' ? 'badge badge-admin' : 'badge badge-user'}>
                        {u.role === 'admin' ? '管理员' : '用户'}
                      </span>
                    </td>
                    <td data-label="应用授权">
                      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                        {SUPPORTED_APPS.map(a => {
                          const granted = u.appScope.includes(a.code)
                          return (
                            <span
                              key={a.code}
                              className="badge"
                              style={{
                                opacity: granted ? 1 : 0.32,
                                borderStyle: granted ? 'solid' : 'dashed',
                              }}
                              title={granted ? `已授权访问 ${a.name}` : `未授权 ${a.name}`}
                            >
                              {a.name}
                            </span>
                          )
                        })}
                      </div>
                    </td>
                    <td data-label="交易数" style={{ textAlign: 'right', fontFamily: 'Outfit, sans-serif' }}>
                      {u.stats?.transactionCount ?? 0}
                    </td>
                    <td data-label="最近活跃" style={{ color: 'var(--text-tertiary)', fontSize: 13 }}>
                      {u.stats?.lastActiveAt ? new Date(u.stats.lastActiveAt).toLocaleString('zh-CN') : '—'}
                    </td>
                    <td data-label="创建时间" style={{ color: 'var(--text-tertiary)', fontSize: 13 }}>
                      {new Date(u.createdAt).toLocaleString('zh-CN')}
                    </td>
                    <td data-label="操作" className="actions" style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                      <Link to={`/admin/users/${u.id}`} className="pill-link" style={{ marginRight: 12 }}>查看</Link>
                      <button
                        className="pill-link"
                        style={{ marginRight: 12, background: 'none', border: 'none', cursor: 'pointer' }}
                        onClick={() => openEdit(u)}
                      >编辑</button>
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
              {filtered.length === 0 && (
                <tr>
                  <td colSpan={7} style={{ textAlign: 'center', padding: 48, color: 'var(--text-tertiary)', fontSize: 14 }}>
                    {users.length === 0 ? '暂无用户' : '无符合筛选的用户'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      <UserFormDrawer
        open={drawerOpen}
        mode={drawerMode}
        user={editingUser}
        onClose={() => setDrawerOpen(false)}
        onSaved={onSaved}
      />

      <style>{`
        .app-filter-chips { display: inline-flex; gap: 6px; align-items: center; }
        .chip {
          padding: 6px 14px;
          border-radius: 999px;
          border: 1px solid var(--border-light);
          background: transparent;
          color: var(--text-secondary);
          font-size: 13px;
          cursor: pointer;
          transition: background 0.15s, color 0.15s, border-color 0.15s;
        }
        .chip:hover { color: var(--text-primary); border-color: var(--border-glow); }
        .chip.active {
          background: var(--primary-gradient, linear-gradient(135deg,#6366F1,#8B5CF6));
          color: #fff;
          border-color: transparent;
          box-shadow: var(--shadow-glow);
        }
        .users-search { position: relative; display: inline-flex; align-items: center; min-width: 200px; }
        .users-search-icon {
          position: absolute;
          left: 12px;
          color: var(--text-tertiary);
          pointer-events: none;
        }
        .users-search-input { padding-left: 34px; min-width: 200px; }
      `}</style>
    </div>
  )
}
