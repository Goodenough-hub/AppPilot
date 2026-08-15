import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import UserFormDrawer from './UserFormDrawer'
import type { User } from '../api/admin'

// Mock admin API so submit doesn't actually hit the network.
vi.mock('../api/admin', async () => {
  const actual = await vi.importActual<object>('../api/admin')
  return {
    ...actual,
    createUser: vi.fn(),
    updateUser: vi.fn()
  }
})
// Sonner toast is unused in these assertions; stub the surface it touches.
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() }
}))

const fakeUser: User = {
  id: '1',
  username: 'alice',
  role: 'user',
  appScope: ['finflow'],
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z'
}

describe('UserFormDrawer 密码可见性切换', () => {
  beforeEach(() => cleanup())

  it('create 模式：默认 password，点击后 text，再点回 password', () => {
    render(
      <UserFormDrawer open={true} mode="create" onClose={() => {}} onSaved={() => {}} />
    )
    const pwd = screen.getByPlaceholderText('至少 6 个字符') as HTMLInputElement
    expect(pwd.type).toBe('password')
    fireEvent.click(screen.getByRole('button', { name: '显示密码' }))
    expect(pwd.type).toBe('text')
    fireEvent.click(screen.getByRole('button', { name: '隐藏密码' }))
    expect(pwd.type).toBe('password')
  })

  it('edit 模式：占位符改成"不修改请留空"，切换仍生效', () => {
    render(
      <UserFormDrawer open={true} mode="edit" user={fakeUser} onClose={() => {}} onSaved={() => {}} />
    )
    const pwd = screen.getByPlaceholderText('不修改请留空') as HTMLInputElement
    expect(pwd.type).toBe('password')
    fireEvent.click(screen.getByRole('button', { name: '显示密码' }))
    expect(pwd.type).toBe('text')
  })

  it('切换按钮 type=button、tabIndex=-1', () => {
    render(
      <UserFormDrawer open={true} mode="create" onClose={() => {}} onSaved={() => {}} />
    )
    const btn = screen.getByRole('button', { name: '显示密码' }) as HTMLButtonElement
    expect(btn.type).toBe('button')
    expect(btn.tabIndex).toBe(-1)
  })
})
