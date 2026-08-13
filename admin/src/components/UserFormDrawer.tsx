import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { toast } from 'sonner'
import { createUser, updateUser, type CreateUserRequest, type UpdateUserRequest, type User } from '../api/admin'
import { SUPPORTED_APPS } from '../lib/apps'

interface UserFormDrawerProps {
  open: boolean
  mode: 'create' | 'edit'
  user?: User
  onClose: () => void
  onSaved: (u: User) => void
}

type Role = 'user' | 'admin'

// Intersection of user.appScope with the two user-facing apps; the "admin"
// pseudo-scope is intentionally hidden — backend syncs it based on role.
function intersectSupported(scope: string[] | undefined): string[] {
  if (!scope) return []
  return SUPPORTED_APPS.map(a => a.code).filter(c => scope.includes(c))
}

export default function UserFormDrawer({ open, mode, user, onClose, onSaved }: UserFormDrawerProps) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('user')
  const [appScope, setAppScope] = useState<string[]>([])
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // Reset transient state whenever the drawer opens.
  useEffect(() => {
    if (!open) return
    setError('')
    setPassword('')
    setSubmitting(false)
    if (mode === 'edit' && user) {
      setUsername(user.username)
      setRole((user.role === 'admin' ? 'admin' : 'user'))
      setAppScope(intersectSupported(user.appScope))
    } else {
      setUsername('')
      setRole('user')
      setAppScope(SUPPORTED_APPS.map(a => a.code)) // default: all apps selected
    }
  }, [open, mode, user])

  if (!open) return null

  const toggleApp = (code: string) => {
    setAppScope(prev => (prev.includes(code) ? prev.filter(c => c !== code) : [...prev, code]))
  }

  const submit = async () => {
    setError('')
    if (mode === 'create') {
      if (username.trim().length < 3) { setError('用户名至少 3 个字符'); return }
      if (password.length < 6) { setError('密码至少 6 个字符'); return }
    }
    if (appScope.length === 0) { setError('至少选择一个应用'); return }
    if (mode === 'edit' && password.length > 0 && password.length < 6) {
      setError('新密码至少 6 个字符'); return
    }

    if (mode === 'edit' && password.length > 0) {
      if (!window.confirm(`确认修改 ${username} 的密码？此操作立即生效。`)) return
    }

    setSubmitting(true)
    try {
      let saved: User
      if (mode === 'create') {
        const body: CreateUserRequest = {
          username: username.trim(),
          password,
          role,
          appScope: [...appScope],
        }
        saved = await createUser(body)
        toast.success(`已创建用户 ${saved.username}`)
      } else if (user) {
        const body: UpdateUserRequest = {
          role,
          appScope: [...appScope],
        }
        if (password.length > 0) body.password = password
        saved = await updateUser(user.id, body)
        toast.success(`已保存 ${saved.username}`)
      } else {
        return
      }
      onSaved(saved)
    } catch (err: unknown) {
      const anyErr = err as { response?: { data?: { error?: string } } }
      const msg = anyErr?.response?.data?.error || (mode === 'create' ? '创建失败' : '保存失败')
      setError(msg)
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="drawer-backdrop" onClick={onClose}>
      <div
        className="glass-panel drawer-panel"
        onClick={e => e.stopPropagation()}
        role="dialog"
        aria-label={mode === 'create' ? '新建用户' : `编辑用户 ${user?.username ?? ''}`}
      >
        <div className="drawer-head">
          <h2>{mode === 'create' ? '新建用户' : '编辑用户'}</h2>
          <button className="btn-ghost" onClick={onClose} aria-label="关闭" style={{ padding: '6px 10px', borderRadius: 8 }}>
            <X size={18} />
          </button>
        </div>

        <div className="drawer-body">
          <section className="drawer-section">
            <label className="drawer-label">用户名</label>
            <input
              className="form-input"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder="至少 3 个字符"
              maxLength={64}
              disabled={mode === 'edit'}
              autoComplete="off"
            />
            {mode === 'edit' && (
              <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
                用户名是登录标识，创建后不可修改。
              </span>
            )}
          </section>

          <section className="drawer-section">
            <label className="drawer-label">
              {mode === 'create' ? '密码' : '新密码（留空则不修改）'}
            </label>
            <input
              className="form-input"
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder={mode === 'create' ? '至少 6 个字符' : '不修改请留空'}
              autoComplete="new-password"
            />
            {mode === 'edit' && password.length > 0 && (
              <span style={{ fontSize: 12, color: '#FCA5A5' }}>
                保存后旧密码立即失效。
              </span>
            )}
          </section>

          <section className="drawer-section">
            <label className="drawer-label">角色</label>
            <select
              className="form-input"
              value={role}
              onChange={e => setRole(e.target.value as Role)}
            >
              <option value="user">用户</option>
              <option value="admin">管理员</option>
            </select>
          </section>

          <section className="drawer-section">
            <label className="drawer-label">应用授权</label>
            <div className="app-check-list">
              {SUPPORTED_APPS.map(a => {
                const checked = appScope.includes(a.code)
                return (
                  <label key={a.code} className={`app-check-item${checked ? ' active' : ''}`}>
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => toggleApp(a.code)}
                    />
                    <span className="app-check-name">{a.name}</span>
                    <span className="app-check-code">{a.code}</span>
                  </label>
                )
              })}
            </div>
            <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
              账户创建后同一用户名密码可用于所勾选的所有应用。
            </span>
          </section>
        </div>

        <div className="drawer-foot">
          {error && <div className="form-error" style={{ marginTop: 0, marginBottom: 8 }}>{error}</div>}
          <button
            className="btn-primary"
            onClick={submit}
            type="button"
            disabled={submitting}
          >
            {submitting ? '保存中…' : (mode === 'create' ? '创建用户' : '保存变更')}
          </button>
        </div>
      </div>

      <style>{`
        .drawer-backdrop {
          position: fixed;
          inset: 0;
          background: rgba(0,0,0,0.5);
          backdrop-filter: blur(4px);
          -webkit-backdrop-filter: blur(4px);
          z-index: 60;
          display: flex;
          justify-content: flex-end;
        }
        .drawer-panel {
          width: 420px;
          max-width: calc(100vw - 24px);
          height: 100%;
          border-radius: 0;
          border-left: 1px solid var(--border-glow);
          display: flex;
          flex-direction: column;
          animation: drawerSlideIn var(--dur-sheet) var(--ease-drawer);
        }
        @keyframes drawerSlideIn {
          from { transform: translateX(100%); opacity: 0.6; }
          to { transform: translateX(0); opacity: 1; }
        }
        .drawer-head {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 20px 20px 12px;
          border-bottom: 1px solid var(--border-light);
        }
        .drawer-head h2 { margin: 0; font-size: 18px; }
        .drawer-body {
          flex: 1;
          overflow-y: auto;
          padding: 16px 20px;
          display: flex;
          flex-direction: column;
          gap: 18px;
        }
        .drawer-section { display: flex; flex-direction: column; gap: 8px; }
        .drawer-label {
          font-size: 12px;
          text-transform: uppercase;
          letter-spacing: 0.05em;
          color: var(--text-secondary);
          font-weight: 500;
        }
        .app-check-list {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }
        .app-check-item {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 10px 14px;
          border-radius: 10px;
          border: 1px solid var(--border-light);
          cursor: pointer;
          transition: background 0.15s, border-color 0.15s;
        }
        .app-check-item:hover { background: var(--surface-hover, rgba(255,255,255,0.03)); }
        .app-check-item.active {
          background: var(--surface-active, rgba(99,102,241,0.08));
          border-color: var(--border-glow);
        }
        .app-check-item input[type="checkbox"] { accent-color: var(--primary, #6366F1); width: 16px; height: 16px; }
        .app-check-name { font-size: 14px; font-weight: 500; color: var(--text-primary); }
        .app-check-code {
          margin-left: auto;
          font-family: 'JetBrains Mono', monospace;
          font-size: 11px;
          color: var(--text-tertiary);
        }
        .drawer-foot {
          padding: 16px 20px 20px;
          border-top: 1px solid var(--border-light);
          display: flex;
          flex-direction: column;
          align-items: stretch;
        }
        @media (prefers-reduced-motion: reduce) {
          .drawer-panel { animation: none; }
        }
      `}</style>
    </div>
  )
}
