import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import AddWidgetDrawer from './AddWidgetDrawer'

vi.mock('../api/dashboard', () => ({
  listDataSources: vi.fn(),
}))

import * as dashboardApi from '../api/dashboard'

const source = (o: Partial<import('../api/dashboard').DataSourceMeta> = {}) => ({
  key: 'finflow:transactions',
  description: '交易列表',
  ...o,
})

beforeEach(() => {
  vi.clearAllMocks()
  ;(dashboardApi.listDataSources as ReturnType<typeof vi.fn>).mockResolvedValue([
    source(),
    source({ key: 'finflow:users', description: '用户列表' }),
  ])
})

describe('AddWidgetDrawer', () => {
  it('关闭时不渲染', () => {
    const { container } = render(<AddWidgetDrawer open={false} onClose={() => {}} onAdd={() => {}} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('打开时加载数据源列表', async () => {
    render(<AddWidgetDrawer open onClose={() => {}} onAdd={() => {}} />)
    expect(dashboardApi.listDataSources).toHaveBeenCalledOnce()
    expect(await screen.findByText('finflow:transactions')).toBeInTheDocument()
    expect(screen.getByText('finflow:users')).toBeInTheDocument()
  })

  it('搜索过滤数据源', async () => {
    render(<AddWidgetDrawer open onClose={() => {}} onAdd={() => {}} />)
    await screen.findByText('finflow:transactions')
    await userEvent.type(screen.getByPlaceholderText('搜索数据源…'), 'users')
    await waitFor(() => expect(screen.queryByText('finflow:transactions')).not.toBeInTheDocument())
    expect(screen.getByText('finflow:users')).toBeInTheDocument()
  })

  it('选择数据源、类型与标题后调用 onAdd', async () => {
    const onAdd = vi.fn()
    render(<AddWidgetDrawer open onClose={() => {}} onAdd={onAdd} />)
    await screen.findByText('finflow:transactions')
    // 默认类型为 stat；选择 chart
    await userEvent.click(screen.getByRole('button', { name: '图表' }))
    await userEvent.click(screen.getByText('finflow:transactions'))
    await userEvent.type(screen.getByPlaceholderText('widget 标题'), '我的图表')
    await userEvent.click(screen.getByRole('button', { name: '添加' }))
    await waitFor(() =>
      expect(onAdd).toHaveBeenCalledWith({
        type: 'chart',
        title: '我的图表',
        dataSource: 'finflow:transactions',
        gridW: 3,
        gridH: 2,
      })
    )
  })

  it('未选数据源时报错且不调用 onAdd', async () => {
    const onAdd = vi.fn()
    render(<AddWidgetDrawer open onClose={() => {}} onAdd={onAdd} />)
    await screen.findByText('finflow:transactions')
    await userEvent.click(screen.getByRole('button', { name: '添加' }))
    expect(await screen.findByText('请选择一个数据源')).toBeInTheDocument()
    expect(onAdd).not.toHaveBeenCalled()
  })

  it('点击背景关闭抽屉', async () => {
    const onClose = vi.fn()
    render(<AddWidgetDrawer open onClose={onClose} onAdd={() => {}} />)
    await screen.findByText('finflow:transactions')
    // 背景是 drawer-backdrop，点击标题文字外、panel 之外
    const backdrop = screen.getByRole('dialog', { name: '添加 widget' }).parentElement!
    await userEvent.click(backdrop)
    expect(onClose).toHaveBeenCalled()
  })
})
