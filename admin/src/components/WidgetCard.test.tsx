import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import WidgetCard from './WidgetCard'
import type { Widget } from '../api/dashboard'

vi.mock('../api/dashboard', () => ({
  queryDataSource: vi.fn(),
}))

import * as dashboardApi from '../api/dashboard'

const widget = (o: Partial<Widget> = {}): Widget => ({
  id: 'w1',
  dashboardId: 'd1',
  type: 'stat',
  title: '交易数',
  dataSource: 'finflow:transactions',
  config: {},
  gridX: 0,
  gridY: 0,
  gridW: 2,
  gridH: 1,
  sortOrder: 0,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  ...o,
})

beforeEach(() => {
  vi.clearAllMocks()
})

describe('WidgetCard', () => {
  it('加载中显示 loading 文案', async () => {
    let resolve: (v: any) => void
    ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockReturnValue(
      new Promise(r => {
        resolve = r
      })
    )
    render(<WidgetCard widget={widget()} />)
    expect(screen.getByText('加载中…')).toBeInTheDocument()
    resolve!([{ label: '总计', value: 42 }])
  })

  it('stat 类型展示首个数值', async () => {
    ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockResolvedValue([
      { label: '总计', value: 42 },
    ])
    render(<WidgetCard widget={widget({ type: 'stat' })} />)
    expect(await screen.findByText('42')).toBeInTheDocument()
    expect(screen.getByText('总计')).toBeInTheDocument()
  })

  it('table 类型渲染表格', async () => {
    ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockResolvedValue([
      { label: 'A', value: 10 },
      { label: 'B', value: 20 },
    ])
    render(<WidgetCard widget={widget({ type: 'table' })} />)
    expect(await screen.findByText('A')).toBeInTheDocument()
    expect(screen.getByText('B')).toBeInTheDocument()
  })

  it('空数据显示空态', async () => {
    ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockResolvedValue([])
    render(<WidgetCard widget={widget({ type: 'stat' })} />)
    expect(await screen.findByText('暂无数据')).toBeInTheDocument()
  })

  it('错误状态展示错误信息', async () => {
    ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockRejectedValue({
      message: 'boom',
    })
    render(<WidgetCard widget={widget({ type: 'stat' })} />)
    expect(await screen.findByText('boom')).toBeInTheDocument()
  })

  it('编辑模式显示删除按钮并触发 onDelete', async () => {
    ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockResolvedValue([
      { label: '总计', value: 1 },
    ])
    const onDelete = vi.fn()
    render(<WidgetCard widget={widget({ type: 'stat' })} isEditing onDelete={onDelete} />)
    const btn = await screen.findByRole('button', { name: '删除 交易数' })
    await userEvent.click(btn)
    expect(onDelete).toHaveBeenCalledOnce()
  })

  it('按 dataSource.config.params 查询数据源', async () => {
    ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockResolvedValue([
      { label: '总计', value: 1 },
    ])
    const params = { limit: 5 }
    render(<WidgetCard widget={widget({ type: 'stat', config: { params } })} />)
    await waitFor(() =>
      expect(dashboardApi.queryDataSource).toHaveBeenCalledWith('finflow:transactions', params)
    )
  })
})

