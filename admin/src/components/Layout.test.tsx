import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import Layout from './Layout'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { username: 'admin', role: 'admin' }, logout: vi.fn() }),
}))

function renderLayout() {
  return render(
    <MemoryRouter initialEntries={['/admin']}>
      <Layout />
    </MemoryRouter>,
  )
}

describe('Layout 抽屉导航', () => {
  it('点击汉堡按钮打开抽屉，点击关闭按钮收起', async () => {
    const u = userEvent.setup()
    const { container } = renderLayout()
    const sidebar = container.querySelector('.admin-sidebar') as HTMLElement
    expect(sidebar.className).not.toContain('open')

    await u.click(screen.getByLabelText('打开菜单'))
    expect(sidebar.className).toContain('open')

    await u.click(screen.getByLabelText('关闭菜单'))
    expect(sidebar.className).not.toContain('open')
  })

  it('点击遮罩关闭抽屉', async () => {
    const u = userEvent.setup()
    const { container } = renderLayout()
    await u.click(screen.getByLabelText('打开菜单'))

    const sidebar = container.querySelector('.admin-sidebar') as HTMLElement
    expect(sidebar.className).toContain('open')

    const backdrop = container.querySelector('.admin-drawer-backdrop') as HTMLElement
    await u.click(backdrop)
    expect(sidebar.className).not.toContain('open')
  })
})
