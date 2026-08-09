import { useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom'
import { LayoutDashboard, Users, Activity, BarChart3, LogOut, Menu, X, PencilLine, Eye, type LucideIcon } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { startBlogSession } from '../api/admin'
import Logo from './Logo'

const navItems: { to: string; label: string; end: boolean; icon: LucideIcon }[] = [
  { to: '/admin', label: '概览', end: true, icon: LayoutDashboard },
  { to: '/admin/dashboards/finflow', label: '应用看板', end: false, icon: BarChart3 },
  { to: '/admin/users', label: '用户管理', end: false, icon: Users },
  { to: '/admin/analytics', label: '页面分析', end: false, icon: Activity },
]

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [blogLoading, setBlogLoading] = useState(false)
  const [blogError, setBlogError] = useState<string | null>(null)

  const handleLogout = () => {
    logout()
    navigate('/admin/login')
  }

  // 进入 FluxBlog：后端用 admin 身份签发 blog JWT cookie（startBlogSession）。
  // cookie 下发后浏览器可自由跳转任意 /blog/* 路径，前端按 target 决定去哪。
  // 「博客写作」→ /blog/studio/；「博客预览」→ /blog/preview/。新开 tab，保留 admin 面板。
  const enterBlog = async (target: string) => {
    setBlogError(null)
    setBlogLoading(true)
    try {
      await startBlogSession()
      window.open(target, '_blank', 'noopener')
    } catch (err: any) {
      setBlogLoading(false)
      const status = err?.response?.status
      setBlogError(status === 403 ? '博客账号已停用，请在「博客账号」页重新启用' : '进入博客失败，请稍后重试')
    }
  }

  // 路由变化时自动关闭抽屉
  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

  // 抽屉打开时锁定 body 滚动
  useEffect(() => {
    if (!drawerOpen) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = prev }
  }, [drawerOpen])

  return (
    <div className="admin-shell">
      {/* 移动端顶栏（≤1024 显示） */}
      <header className="admin-topbar">
        <button
          className="admin-topbar-toggle"
          aria-label="打开菜单"
          aria-expanded={drawerOpen}
          onClick={() => setDrawerOpen(true)}
        >
          <Menu size={22} strokeWidth={2} />
        </button>
        <div className="admin-brand">
          <div className="admin-brand-mark"><Logo size={26} /></div>
          AppPilot
        </div>
      </header>

      {/* 抽屉遮罩（移动端打开时显示） */}
      <div
        className={`admin-drawer-backdrop ${drawerOpen ? 'open' : ''}`}
        onClick={() => setDrawerOpen(false)}
        aria-hidden="true"
      />

      <aside className={`glass-panel admin-sidebar ${drawerOpen ? 'open' : ''}`}>
        <div className="admin-sidebar-head">
          <div className="admin-brand">
            <div className="admin-brand-mark">
              <Logo size={32} />
            </div>
            AppPilot
          </div>
          <button
            className="admin-drawer-close"
            aria-label="关闭菜单"
            onClick={() => setDrawerOpen(false)}
          >
            <X size={20} strokeWidth={2} />
          </button>
        </div>
        <nav className="admin-nav">
          {navItems.map(item => {
            const Icon = item.icon
            return (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) => `admin-nav-link ${isActive ? 'active' : ''}`}
              >
                <Icon size={18} strokeWidth={2} />
                {item.label}
              </NavLink>
            )
          })}
          <button
            type="button"
            className="admin-nav-link admin-nav-action"
            onClick={() => enterBlog('/blog/studio/')}
            disabled={blogLoading}
            aria-label="进入博客写作后台"
          >
            <PencilLine size={18} strokeWidth={2} />
            {blogLoading ? '正在进入…' : '博客写作'}
          </button>
          <button
            type="button"
            className="admin-nav-link admin-nav-action"
            onClick={() => enterBlog('/blog/preview/')}
            disabled={blogLoading}
            aria-label="进入博客预览页"
          >
            <Eye size={18} strokeWidth={2} />
            {blogLoading ? '正在进入…' : '博客预览'}
          </button>
          {blogError && (
            <div className="admin-nav-hint" role="alert">{blogError}</div>
          )}
        </nav>
        <div className="admin-sidebar-footer">
          <div className="admin-user">
            <div className="admin-user-avatar">
              {user?.username?.[0]?.toUpperCase()}
            </div>
            <div>
              <div className="admin-user-name">{user?.username}</div>
              <div className="admin-user-role">{user?.role}</div>
            </div>
          </div>
          <button onClick={handleLogout} style={{ width: '100%', fontSize: 14, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
            <LogOut size={16} strokeWidth={2} />
            退出系统
          </button>
        </div>
      </aside>
      <main className="admin-main">
        <Outlet />
      </main>
    </div>
  )
}
