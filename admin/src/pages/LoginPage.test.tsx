import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import LoginPage from './LoginPage'

// Isolate LoginPage from the real AuthContext (which touches localStorage and the API).
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    login: vi.fn(),
    logout: vi.fn(),
    user: null,
    loading: false
  })
}))

function renderPage() {
  return render(
    <MemoryRouter>
      <LoginPage />
    </MemoryRouter>
  )
}

describe('AppPilot admin LoginPage 密码可见性切换', () => {
  beforeEach(() => cleanup())

  it('密码 input 默认 type=password', () => {
    renderPage()
    const pwd = screen.getByPlaceholderText('请输入密码') as HTMLInputElement
    expect(pwd.type).toBe('password')
  })

  it('点击眼睛后 type=text，再点击回到 password', () => {
    renderPage()
    const pwd = screen.getByPlaceholderText('请输入密码') as HTMLInputElement
    fireEvent.click(screen.getByRole('button', { name: '显示密码' }))
    expect(pwd.type).toBe('text')
    fireEvent.click(screen.getByRole('button', { name: '隐藏密码' }))
    expect(pwd.type).toBe('password')
  })

  it('切换按钮 type=button、tabIndex=-1', () => {
    renderPage()
    const btn = screen.getByRole('button', { name: '显示密码' }) as HTMLButtonElement
    expect(btn.type).toBe('button')
    expect(btn.tabIndex).toBe(-1)
  })
})
