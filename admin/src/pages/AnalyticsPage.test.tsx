import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import AnalyticsPage, { getAnalyticsRange } from './AnalyticsPage'

vi.mock('../api/admin', () => ({ listApps: vi.fn() }))
vi.mock('../api/analytics', () => ({
  getPV: vi.fn(),
  getAnalyticsSummary: vi.fn(),
  getTopPages: vi.fn(),
  getActiveSessions: vi.fn(),
}))
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  AreaChart: ({ children }: { children: ReactNode }) => <svg>{children}</svg>,
  Area: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
}))

import * as adminApi from '../api/admin'
import * as analyticsApi from '../api/analytics'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(adminApi.listApps).mockResolvedValue(['finflow'])
  vi.mocked(analyticsApi.getPV).mockResolvedValue([
    { date: '2026-08-23', pv: 12, uv: 3 },
  ])
  vi.mocked(analyticsApi.getAnalyticsSummary).mockResolvedValue({ pageViews: 12, uniqueSessions: 3 })
  vi.mocked(analyticsApi.getTopPages).mockResolvedValue([
    { path: '/finflow/', pv: 12, uv: 3 },
  ])
  vi.mocked(analyticsApi.getActiveSessions).mockResolvedValue(2)
})

describe('AnalyticsPage 页面分析', () => {
  it('用中文全称解释并展示页面访问指标', async () => {
    render(<AnalyticsPage />)

    expect(await screen.findByText('每日页面访问趋势')).toBeInTheDocument()
    expect(screen.getAllByText('页面浏览次数').length).toBeGreaterThan(0)
    expect(screen.getAllByText('独立会话数').length).toBeGreaterThan(0)
    expect(screen.getByText('每次访问平均浏览页数')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('近 5 分钟活跃会话')).toBeInTheDocument())
    expect(screen.queryByText('PV')).not.toBeInTheDocument()
    expect(screen.queryByText('UV')).not.toBeInTheDocument()
  })

  it('使用范围汇总计算每次访问平均浏览页数', async () => {
    render(<AnalyticsPage />)

    await waitFor(() => expect(analyticsApi.getAnalyticsSummary).toHaveBeenCalledOnce())
    expect(screen.getByText('4.0')).toBeInTheDocument()
  })
})

describe('getAnalyticsRange', () => {
  const now = new Date(2026, 7, 23, 10, 30, 0)

  it('今天从本地零点开始', () => {
    const range = getAnalyticsRange(0, now)
    expect(new Date(range.start)).toEqual(new Date(2026, 7, 23, 0, 0, 0))
    expect(new Date(range.end)).toEqual(now)
  })

  it('近 7 天包含今天且只有 7 个自然日', () => {
    const range = getAnalyticsRange(7, now)
    expect(new Date(range.start)).toEqual(new Date(2026, 7, 17, 0, 0, 0))
  })

  it('近 30 天包含今天且只有 30 个自然日', () => {
    const range = getAnalyticsRange(30, now)
    expect(new Date(range.start)).toEqual(new Date(2026, 6, 25, 0, 0, 0))
  })
})
