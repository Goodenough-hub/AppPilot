import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import Logo from './Logo'

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
    <div style={{ display: 'flex', minHeight: '100vh', padding: '24px', gap: '24px' }}>
      <aside className="glass-panel" style={{
        width: 260,
        padding: '32px 20px',
        display: 'flex',
        flexDirection: 'column',
        height: 'calc(100vh - 48px)',
        position: 'sticky',
        top: 24
      }}>
        <div style={{
          fontWeight: 700,
          fontSize: 22,
          marginBottom: 40,
          color: 'var(--text-primary)',
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          fontFamily: 'Outfit, sans-serif'
        }}>
          <div style={{ boxShadow: 'var(--shadow-glow)', borderRadius: 8, display: 'flex' }}>
            <Logo size={32} />
          </div>
          AppPilot
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: 8, flex: 1 }}>
          {navItems.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              style={({ isActive }) => ({
                padding: '12px 16px',
                borderRadius: 12,
                color: isActive ? '#FFFFFF' : 'var(--text-secondary)',
                background: isActive ? 'var(--surface-active)' : 'transparent',
                border: `1px solid ${isActive ? 'var(--border-glow)' : 'transparent'}`,
                textDecoration: 'none',
                fontWeight: isActive ? 500 : 400,
                fontSize: 14,
                transition: 'all 0.2s ease',
                display: 'flex',
                alignItems: 'center',
                boxShadow: isActive ? '0 4px 12px rgba(0,0,0,0.1)' : 'none'
              })}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div style={{ paddingTop: 24, borderTop: '1px solid var(--border-light)' }}>
          <div style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12 }}>
            <div style={{ width: 36, height: 36, borderRadius: 18, background: 'var(--surface-active)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
               {user?.username?.[0]?.toUpperCase()}
            </div>
            <div>
              <div style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{user?.username}</div>
              <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{user?.role}</div>
            </div>
          </div>
          <button onClick={handleLogout} style={{ width: '100%', fontSize: 14 }}>退出系统</button>
        </div>
      </aside>
      <main style={{ flex: 1, padding: '16px 32px 48px', overflow: 'auto', maxWidth: 1400 }}>
        <Outlet />
      </main>
    </div>
  )
}
