import { apiClient } from './client'

export interface TrackRequest {
  app: string
  eventType: string
  path: string
  title?: string
  referrer?: string
  sessionId?: string
}

export interface PVDailyRow {
  date: string
  pv: number
  uv: number
}

export interface TopPageRow {
  path: string
  pv: number
  uv: number
}

export interface PVAggregateParams {
  app: string
  start?: string  // RFC3339
  end?: string    // RFC3339
  limit?: number
}

function buildParams(params: PVAggregateParams): Record<string, string> {
  const p: Record<string, string> = { app: params.app }
  if (params.start) p.start = params.start
  if (params.end) p.end = params.end
  if (params.limit) p.limit = String(params.limit)
  return p
}

/** 埋点上报（fire-and-forget，失败不抛异常） */
export function track(req: TrackRequest): void {
  apiClient.post('/analytics/track', req).catch(() => {
    // fire-and-forget: 上报失败不影响主流程
  })
}

/** 获取 PV/UV 日聚合数据 */
export async function getPV(params: PVAggregateParams): Promise<PVDailyRow[]> {
  const { data } = await apiClient.get<PVDailyRow[]>('/admin/analytics/pv', {
    params: buildParams(params),
  })
  return data
}

/** 获取热门页面排行 */
export async function getTopPages(params: PVAggregateParams): Promise<TopPageRow[]> {
  const { data } = await apiClient.get<TopPageRow[]>('/admin/analytics/top-pages', {
    params: buildParams(params),
  })
  return data
}

/** 获取实时在线用户数（近5分钟） */
export async function getRealtime(app: string): Promise<number> {
  const { data } = await apiClient.get<{ online: number }>('/admin/analytics/realtime', {
    params: { app },
  })
  return data.online
}

/** 生成会话 ID（浏览器 session 级别，同标签页共享） */
export function getSessionId(): string {
  const key = 'apppilot_analytics_sid'
  let sid = sessionStorage.getItem(key)
  if (!sid) {
    sid = crypto.randomUUID()
    sessionStorage.setItem(key, sid)
  }
  return sid
}
