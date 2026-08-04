import { useEffect, useState, type FormEvent } from 'react'
import { toast } from 'sonner'
import {
  createBlogUser,
  deleteBlogUser,
  listBlogUsers,
  resetBlogUserPassword,
  updateBlogUser,
  type BlogUser,
} from '../api/admin'

const emptyForm = { username: '', password: '' }

interface ResetTarget {
  id: string
  username: string
  password: string
}

export default function BlogUsersPage() {
  const [users, setUsers] = useState<BlogUser[]>([])
  const [form, setForm] = useState(emptyForm)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [leavingIds, setLeavingIds] = useState<Set<string>>(new Set())
  const [resetTarget, setResetTarget] = useState<ResetTarget | null>(null)

  const reload = () =>
    listBlogUsers()
      .then(setUsers)
      .catch(err => setError(err.response?.data?.error || '加载失败'))

  useEffect(() => {
    reload()
  }, [])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await createBlogUser(form)
      setForm(emptyForm)
      toast.success(`已创建博客账号 ${form.username}`)
      await reload()
    } catch (err: any) {
      const msg = err.response?.data?.error || '创建失败'
      setError(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const toggle = async (u: BlogUser) => {
    try {
      await updateBlogUser(u.id, { isEnabled: !u.isEnabled })
      toast.success(`${u.username} 已${u.isEnabled ? '停用' : '启用'}`)
      await reload()
    } catch (err: any) {
      toast.error(err.response?.data?.error || '操作失败')
    }
  }

  const remove = async (u: BlogUser) => {
    if (!confirm(`确认软删除博客账号 ${u.username}？令牌立即失效，草稿与历史保留。`)) return
    setLeavingIds(prev => new Set(prev).add(u.id))
    try {
      await deleteBlogUser(u.id)
      toast.success(`已删除 ${u.username}`)
      setTimeout(reload, 250)
    } catch (err: any) {
      toast.error(err.response?.data?.error || '删除失败')
      setLeavingIds(prev => {
        const next = new Set(prev)
        next.delete(u.id)
        return next
      })
    }
  }

  const submitReset = async (e: FormEvent) => {
    e.preventDefault()
    if (!resetTarget) return
    try {
      await resetBlogUserPassword(resetTarget.id, resetTarget.password)
      toast.success(`${resetTarget.username} 密码已重置（旧令牌失效）`)
      setResetTarget(null)
      await reload()
    } catch (err: any) {
      toast.error(err.response?.data?.error || '重置失败')
    }
  }

  return (
    <div className="page-container">
      <header className="page-header">
        <h1>博客账号</h1>
        <p className="page-subtitle">FluxBlog 独立账号，与 FinFlow 用户隔离。停用/删除令牌立即失效。</p>
      </header>

      <section className="glass-panel" style={{ padding: 20, marginBottom: 24 }}>
        <h2 style={{ marginTop: 0 }}>新建账号</h2>
        <form onSubmit={submit} className="admin-form-row">
          <input
            className="form-input"
            placeholder="用户名（≥3）"
            value={form.username}
            onChange={e => setForm({ ...form, username: e.target.value })}
            minLength={3}
            maxLength={64}
            required
          />
          <input
            className="form-input"
            type="password"
            placeholder="密码（≥6）"
            value={form.password}
            onChange={e => setForm({ ...form, password: e.target.value })}
            minLength={6}
            required
          />
          <button className="btn-primary" type="submit" disabled={loading}>
            {loading ? '创建中…' : '创建'}
          </button>
        </form>
        {error && <div className="form-error">{error}</div>}
      </section>

      <section className="glass-panel" style={{ padding: 0, overflow: 'hidden' }}>
        <table className="admin-table responsive-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>状态</th>
              <th>tokenVersion</th>
              <th>更新时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {users.map(u => (
              <tr
                key={u.id}
                className={leavingIds.has(u.id) ? 'row-leaving' : ''}
                style={{ opacity: u.deletedAt ? 0.5 : 1 }}
              >
                <td data-label="ID">{u.id}</td>
                <td data-label="用户名">{u.username}{u.deletedAt && <span className="muted">（已删除）</span>}</td>
                <td data-label="状态">
                  <span className={`badge ${u.isEnabled ? 'badge-ok' : 'badge-off'}`}>
                    {u.isEnabled ? '启用' : '停用'}
                  </span>
                </td>
                <td data-label="tokenVersion">{u.tokenVersion}</td>
                <td data-label="更新时间">{u.updatedAt ? new Date(u.updatedAt).toLocaleString() : '—'}</td>
                <td data-label="操作" className="actions">
                  <button className="btn-ghost" onClick={() => toggle(u)} disabled={!!u.deletedAt}>
                    {u.isEnabled ? '停用' : '启用'}
                  </button>
                  <button
                    className="btn-ghost"
                    onClick={() =>
                      setResetTarget({ id: u.id, username: u.username, password: '' })
                    }
                    disabled={!!u.deletedAt}
                  >
                    重置密码
                  </button>
                  <button className="btn-danger-ghost" onClick={() => remove(u)} disabled={!!u.deletedAt}>
                    删除
                  </button>
                </td>
              </tr>
            ))}
            {users.length === 0 && (
              <tr>
                <td colSpan={6} className="empty-row">暂无博客账号</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>

      {resetTarget && (
        <div className="modal-backdrop" onClick={() => setResetTarget(null)}>
          <div className="glass-panel modal-card" onClick={e => e.stopPropagation()}>
            <h3 style={{ marginTop: 0 }}>重置「{resetTarget.username}」密码</h3>
            <form onSubmit={submitReset} className="admin-form-row">
              <input
                className="form-input"
                type="password"
                placeholder="新密码（≥6）"
                value={resetTarget.password}
                onChange={e => setResetTarget({ ...resetTarget, password: e.target.value })}
                minLength={6}
                required
                autoFocus
              />
              <button className="btn-primary" type="submit">重置</button>
              <button className="btn-ghost" type="button" onClick={() => setResetTarget(null)}>取消</button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
