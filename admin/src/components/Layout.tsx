import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { LayoutDashboard, Users, LogOut, type LucideIcon } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import Logo from './Logo'

const navItems: { to: string; label: string; end: boolean; icon: LucideIcon }[] = [
  { to: '/admin', label: '概览', end: true, icon: LayoutDashboard },
  { to: '/admin/users', label: '用户管理', end: false, icon: Users }
]

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const handleLogout = () => {
    logout()
    navigate('/admin/login')
  }
  return (
    <div className="admin-shell">
      <aside className="glass-panel admin-sidebar">
        <div className="admin-brand">
          <div className="admin-brand-mark">
            <Logo size={32} />
          </div>
          AppPilot
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
