import { useEffect, useState, useCallback } from 'react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import { listApps } from '../api/admin'
import {
  getActiveSessions,
  getAnalyticsSummary,
  getPV,
  getTopPages,
  type AnalyticsSummary,
  type PVDailyRow,
  type TopPageRow,
} from '../api/analytics'
import StatCard from '../components/ui/StatCard'

const AXIS = '#71717A'
const GRID = 'rgba(255,255,255,0.06)'
const BLUE = '#818CF8'
const PURPLE = '#A855F7'

const tooltipStyle = {
  background: 'rgba(9, 9, 11, 0.95)',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: 12,
  fontSize: 13,
  color: '#fff',
}

const RANGES: { label: string; days: number }[] = [
  { label: '今天', days: 0 },
  { label: '近 7 天', days: 7 },
  { label: '近 30 天', days: 30 },
]

export function getAnalyticsRange(days: number, now = new Date()): { start: string; end: string } {
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  if (days > 0) start.setDate(start.getDate() - days + 1)
  return { start: start.toISOString(), end: now.toISOString() }
}

export default function AnalyticsPage() {
  const [apps, setApps] = useState<string[]>([])
  const [app, setApp] = useState<string>('')
  const [days, setDays] = useState(7)
  const [pvData, setPvData] = useState<PVDailyRow[]>([])
  const [summary, setSummary] = useState<AnalyticsSummary>({ pageViews: 0, uniqueSessions: 0 })
  const [topPages, setTopPages] = useState<TopPageRow[]>([])
  const [activeSessions, setActiveSessions] = useState(0)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  // 加载应用列表
  useEffect(() => {
    listApps()
      .then(list => {
        setApps(list)
        if (list.length > 0 && !app) setApp(list[0])
      })
      .catch(err => setError(err.response?.data?.error || '加载应用列表失败'))
  }, [])

  // 加载数据
  const load = useCallback(async () => {
    if (!app) return
    setLoading(true)
    setError('')
    try {
      const { start, end } = getAnalyticsRange(days)
      const [pv, aggregate, top] = await Promise.all([
        getPV({ app, start, end }),
        getAnalyticsSummary({ app, start, end }),
        getTopPages({ app, start, end, limit: 20 }),
      ])
      setPvData(pv)
      setSummary(aggregate)
      setTopPages(top)
      // 活跃会话独立加载，失败不影响主数据
      getActiveSessions(app).then(setActiveSessions).catch(() => {})
    } catch (err: any) {
      setError(err.response?.data?.error || '加载失败')
    } finally {
      setLoading(false)
    }
  }, [app, days])

  useEffect(() => { load() }, [load])

  const averagePages = summary.uniqueSessions > 0
    ? (summary.pageViews / summary.uniqueSessions).toFixed(1)
    : '0'

  return (
    <div className="animate-fade-in-up">
      <header className="admin-page-header">
        <div>
          <h1>页面分析</h1>
          <div className="subtitle">按应用查看页面浏览与访问会话</div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ color: 'var(--text-secondary)', fontSize: 13, fontWeight: 500 }}>应用</span>
          <select value={app} onChange={e => setApp(e.target.value)} style={{ width: 160 }}>
            {apps.map(a => <option key={a} value={a}>{a}</option>)}
          </select>
        </div>
      </header>

      {error && (
        <div style={{
          color: '#FCA5A5', background: 'var(--danger-bg)', padding: '12px 16px',
          borderRadius: 12, marginBottom: 24, fontSize: 14,
          border: '1px solid rgba(239, 68, 68, 0.2)'
        }}>{error}</div>
      )}

      <div className="analytics-metric-help">
        <strong>页面浏览次数</strong>：每次打开或切换页面计 1 次。
        <strong>独立会话数</strong>：同一标签页会话在所选时间范围内只计 1 次。
      </div>

      {/* 概览卡片 */}
      <div className="stat-grid">
        <StatCard label="页面浏览次数" value={summary.pageViews} gradient="linear-gradient(135deg, #818CF8, #6366F1)" glow="#818CF8" gradientValue className="animate-fade-in-up stagger-1" />
        <StatCard label="独立会话数" value={summary.uniqueSessions} gradient="linear-gradient(135deg, #A855F7, #7C3AED)" glow="#A855F7" gradientValue className="animate-fade-in-up stagger-2" />
        <StatCard label="每次访问平均浏览页数" value={averagePages} gradient="linear-gradient(135deg, #10B981, #059669)" glow="#10B981" gradientValue className="animate-fade-in-up stagger-3" />
        <StatCard label="近 5 分钟活跃会话" value={activeSessions} gradient="linear-gradient(135deg, #F59E0B, #D97706)" glow="#F59E0B" gradientValue className="animate-fade-in-up stagger-4" />
      </div>

      {/* 时间范围 */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
        {RANGES.map(r => (
          <button
            key={r.days}
            onClick={() => setDays(r.days)}
            className={days === r.days ? 'primary' : ''}
            style={{ padding: '8px 16px', fontSize: 13 }}
          >
            {r.label}
          </button>
        ))}
      </div>

      {/* 页面访问趋势图 */}
      <div className="charts-row" style={{ marginBottom: 32 }}>
        <div className="glass-panel chart-card" style={{ gridColumn: '1 / -1' }}>
          <h2>每日页面访问趋势</h2>
          <div className="subtitle">按天展示页面浏览次数和独立会话数</div>
          {loading ? (
            <div style={{ height: 260, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-tertiary)' }}>加载中…</div>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <AreaChart data={pvData} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
                <defs>
                  <linearGradient id="pvFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={BLUE} stopOpacity={0.3} />
                    <stop offset="100%" stopColor={BLUE} stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="uvFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={PURPLE} stopOpacity={0.25} />
                    <stop offset="100%" stopColor={PURPLE} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="date" stroke={AXIS} fontSize={12} tickLine={false} axisLine={{ stroke: GRID }} />
                <YAxis stroke={AXIS} fontSize={12} tickLine={false} axisLine={false} allowDecimals={false} width={36} />
                <Tooltip contentStyle={tooltipStyle} cursor={{ stroke: GRID }} />
                <Area type="monotone" dataKey="pv" name="页面浏览次数" stroke={BLUE} strokeWidth={2} fill="url(#pvFill)" animationDuration={280} />
                <Area type="monotone" dataKey="uv" name="独立会话数" stroke={PURPLE} strokeWidth={2} fill="url(#uvFill)" animationDuration={280} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {/* 热门页面排行 */}
      <div className="glass-panel animate-fade-in-up stagger-4" style={{ overflow: 'hidden' }}>
        <div style={{ padding: '24px 32px' }}>
          <h2 style={{ fontSize: 20, margin: 0 }}>热门页面</h2>
        </div>
        <div className="table-container">
          <table className="responsive-table">
            <thead>
              <tr>
                <th>#</th>
                <th>页面路径</th>
                <th style={{ textAlign: 'right' }}>页面浏览次数</th>
                <th style={{ textAlign: 'right' }}>独立会话数</th>
              </tr>
            </thead>
            <tbody>
              {topPages.map((p, i) => (
                <tr key={p.path}>
                  <td data-label="#" style={{ color: 'var(--text-tertiary)', fontSize: 13, fontWeight: 600 }}>{i + 1}</td>
                  <td data-label="页面路径" style={{ fontWeight: 500 }}>{p.path}</td>
                  <td data-label="页面浏览次数" style={{ textAlign: 'right', fontFamily: 'Outfit, sans-serif' }}>{p.pv}</td>
                  <td data-label="独立会话数" style={{ textAlign: 'right', fontFamily: 'Outfit, sans-serif' }}>{p.uv}</td>
                </tr>
              ))}
              {topPages.length === 0 && (
                <tr>
                  <td colSpan={4} style={{ textAlign: 'center', padding: 48, color: 'var(--text-tertiary)', fontSize: 14 }}>
                    暂无数据
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
