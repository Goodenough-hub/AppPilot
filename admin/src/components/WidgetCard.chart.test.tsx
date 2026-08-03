import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import React from 'react'
import WidgetCard from './WidgetCard'
import type { Widget } from '../api/dashboard'

// recharts ResponsiveContainer depends on ResizeObserver (absent in jsdom) and
// never measures a size, so the inner chart never mounts in this environment.
// Mock recharts with lightweight stubs that tag the chosen chart kind.
vi.mock('recharts', () => {
  const tag = (name: string) => () =>
    React.createElement('div', { 'data-chart': name })
  return {
    ResponsiveContainer: (props: { children: React.ReactNode }) =>
      React.createElement('div', { 'data-testid': 'rc' }, props.children),
    AreaChart: tag('area'),
    Area: tag('series'),
    BarChart: tag('bar'),
    Bar: tag('series'),
    PieChart: tag('pie'),
    Pie: tag('series'),
    Cell: tag('cell'),
    XAxis: tag('xaxis'),
    YAxis: tag('yaxis'),
    Tooltip: tag('tooltip'),
  }
})

vi.mock('../api/dashboard', () => ({
  queryDataSource: vi.fn(),
}))

import * as dashboardApi from '../api/dashboard'

const widget = (o: Partial<Widget> = {}): Widget => ({
  id: 'w1',
  dashboardId: 'd1',
  type: 'chart',
  title: '图表',
  dataSource: 'finflow:transactions',
  config: {},
  gridX: 0,
  gridY: 0,
  gridW: 3,
  gridH: 2,
  sortOrder: 0,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  ...o,
})

beforeEach(() => {
  vi.clearAllMocks()
  ;(dashboardApi.queryDataSource as ReturnType<typeof vi.fn>).mockResolvedValue([
    { label: '1月', value: 10 },
    { label: '2月', value: 20 },
  ])
})

describe('WidgetCard chart 子类型选择', () => {
  it('trend → AreaChart', async () => {
    const { container } = render(
      <WidgetCard widget={widget({ config: { chartType: 'trend' } })} />
    )
    await waitFor(() =>
      expect(container.querySelector('[data-chart="area"]')).toBeInTheDocument()
    )
  })

  it('breakdown → PieChart', async () => {
    const { container } = render(
      <WidgetCard widget={widget({ config: { chartType: 'breakdown' } })} />
    )
    await waitFor(() =>
      expect(container.querySelector('[data-chart="pie"]')).toBeInTheDocument()
    )
  })

  it('ranking → BarChart', async () => {
    const { container } = render(
      <WidgetCard widget={widget({ config: { chartType: 'ranking' } })} />
    )
    await waitFor(() =>
      expect(container.querySelector('[data-chart="bar"]')).toBeInTheDocument()
    )
  })

  it('未知 chartType 默认 → BarChart', async () => {
    const { container } = render(<WidgetCard widget={widget({ config: {} })} />)
    await waitFor(() =>
      expect(container.querySelector('[data-chart="bar"]')).toBeInTheDocument()
    )
  })
})
