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
    <div style={{ display: 'flex', minHeight: '100vh', background: 'var(--bg)' }}>
      <aside style={{
        width: 220,
        background: 'var(--surface)',
        borderRight: '1px solid var(--border-soft)',
        padding: '32px 20px',
        display: 'flex',
        flexDirection: 'column'
      }}>
        <div style={{
          fontWeight: 600,
          fontSize: 17,
          marginBottom: 32,
          letterSpacing: '-0.022em',
          color: 'var(--text)'
        }}>AppPilot</div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: 2, flex: 1 }}>
          {navItems.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              style={({ isActive }) => ({
                padding: '9px 14px',
                borderRadius: 10,
                color: isActive ? 'var(--text)' : 'var(--text-dim)',
                background: isActive ? 'var(--surface-2)' : 'transparent',
                textDecoration: 'none',
                fontWeight: isActive ? 500 : 400,
                fontSize: 14,
                transition: 'all 0.15s ease'
              })}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div style={{ paddingTop: 20, borderTop: '1px solid var(--border-soft)' }}>
          <div style={{ fontSize: 12, color: 'var(--text-dim)', marginBottom: 10, letterSpacing: '-0.01em' }}>
            {user?.username} · {user?.role}
          </div>
          <button onClick={handleLogout} style={{ width: '100%', padding: '7px 14px', fontSize: 13 }}>退出</button>
        </div>
      </aside>
      <main style={{ flex: 1, padding: '48px 56px', overflow: 'auto', maxWidth: 1280 }}>
        <Outlet />
      </main>
    </div>
  )
}
