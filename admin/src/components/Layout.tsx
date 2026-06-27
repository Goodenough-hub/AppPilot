import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

const navItems = [
  { to: '/admin', label: '概览', end: true },
  { to: '/admin/users', label: '用户管理', end: false }
]

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const handleLogout = () => {
    logout()
    navigate('/admin/login')
  }
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <aside style={{ width: 200, background: 'var(--surface)', borderRight: '1px solid var(--border)', padding: 16 }}>
        <div style={{ fontWeight: 600, fontSize: 16, marginBottom: 24 }}>AppPilot</div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {navItems.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              style={({ isActive }) => ({
                padding: '8px 12px',
                borderRadius: 6,
                color: isActive ? 'white' : 'var(--text-dim)',
                background: isActive ? 'var(--primary)' : 'transparent',
                textDecoration: 'none'
              })}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div style={{ paddingTop: 24, borderTop: '1px solid var(--border)', marginTop: 200 }}>
          <div style={{ fontSize: 12, color: 'var(--text-dim)', marginBottom: 8 }}>
            {user?.username} ({user?.role})
          </div>
          <button onClick={handleLogout} style={{ width: '100%' }}>退出</button>
        </div>
      </aside>
      <main style={{ flex: 1, padding: 24, overflow: 'auto' }}>
        <Outlet />
      </main>
    </div>
  )
}
