import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { queryDataSource, type ChartData, type Widget } from '../api/dashboard'

const AXIS = '#71717A'
const GRID = 'rgba(255,255,255,0.06)'
const INDIGO = '#818CF8'
const PURPLE = '#A855F7'

const tooltipStyle = {
  background: 'rgba(9, 9, 11, 0.95)',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: 12,
  fontSize: 13,
  color: '#fff',
}

const PIE_COLORS = [INDIGO, PURPLE, '#F59E0B', '#10B981', '#EC4899', '#3B82F6']

interface WidgetCardProps {
  widget: Widget
  isEditing?: boolean
  onDelete?: () => void
}

type LoadState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'ok'; data: ChartData[] }

function formatValue(n: number): string {
  if (!Number.isFinite(n)) return '—'
  if (Math.abs(n) >= 10000) return n.toLocaleString('en-US', { maximumFractionDigits: 1 })
  return String(n)
}

/** Determine which chart sub-type to render from widget.config.chartType. */
function resolveChartKind(chartType: unknown): 'area' | 'pie' | 'bar' {
  const t = String(chartType ?? '').toLowerCase()
  if (['trend', 'area', 'line', 'time', 'timeseries'].includes(t)) return 'area'
  if (['breakdown', 'pie', 'donut', 'distribution'].includes(t)) return 'pie'
  if (['ranking', 'bar', 'rank'].includes(t)) return 'bar'
  return 'bar'
}

export default function WidgetCard({ widget, isEditing, onDelete }: WidgetCardProps) {
  const [state, setState] = useState<LoadState>({ status: 'loading' })

  useEffect(() => {
    let cancelled = false
    setState({ status: 'loading' })
    const params = widget.config?.params
    queryDataSource(widget.dataSource, params)
      .then(data => {
        if (cancelled) return
        setState({ status: 'ok', data: data ?? [] })
      })
      .catch((err: any) => {
        if (cancelled) return
        const message = err?.response?.data?.error || err?.message || '加载失败'
        setState({ status: 'error', message })
      })
    return () => {
      cancelled = true
    }
  }, [widget.dataSource, widget.config])

  return (
    <div className="glass-panel widget-card" style={{ padding: 16, height: '100%', position: 'relative', overflow: 'hidden' }}>
      <div className="widget-card-head" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <h3 style={{ margin: 0, fontSize: 15, fontWeight: 600 }}>{widget.title}</h3>
        {isEditing && onDelete && (
          <button
            className="btn-ghost"
            onClick={onDelete}
            aria-label={`删除 ${widget.title}`}
            title="删除 widget"
            style={{ padding: '4px 8px', borderRadius: 8, lineHeight: 1 }}
          >
            <X size={16} />
          </button>
        )}
      </div>

      <WidgetBody widget={widget} state={state} />
    </div>
  )
}

function WidgetBody({ widget, state }: { widget: Widget; state: LoadState }) {
  if (state.status === 'loading') {
    return <div className="widget-state" style={stateStyle}>加载中…</div>
  }
  if (state.status === 'error') {
    return <div className="widget-state" style={{ ...stateStyle, color: '#FCA5A5' }}>{state.message}</div>
  }
  const data = state.data
  if (!data || data.length === 0) {
    return <div className="widget-state" style={stateStyle}>暂无数据</div>
  }

  if (widget.type === 'stat') {
    const first = data[0]
    return (
      <div className="widget-stat" style={{ display: 'flex', alignItems: 'baseline', gap: 8, flexWrap: 'wrap' }}>
        <span className="stat-card-value" style={{ fontSize: 40, fontWeight: 700, fontFamily: 'Outfit, sans-serif' }}>
          {formatValue(Number(first.value))}
        </span>
        {first.label && <span style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{first.label}</span>}
      </div>
    )
  }

  if (widget.type === 'table') {
    return (
      <div className="table-container" style={{ maxHeight: 280, overflow: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            <tr>
              <th style={cellHead}>标签</th>
              <th style={{ ...cellHead, textAlign: 'right' }}>值</th>
            </tr>
          </thead>
          <tbody>
            {data.map((row, i) => (
              <tr key={i}>
                <td style={cellBody}>{row.label}</td>
                <td style={{ ...cellBody, textAlign: 'right' }}>{formatValue(Number(row.value))}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }

  // chart
  const kind = resolveChartKind(widget.config?.chartType)
  return (
    <ResponsiveContainer width="100%" height={220}>
      {kind === 'area' ? (
        <AreaChart data={data} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
          <defs>
            <linearGradient id={`widgetAreaFill-${widget.id}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={INDIGO} stopOpacity={0.35} />
              <stop offset="100%" stopColor={INDIGO} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis dataKey="label" stroke={AXIS} fontSize={11} tickLine={false} axisLine={{ stroke: GRID }} />
          <YAxis stroke={AXIS} fontSize={11} tickLine={false} axisLine={false} width={32} />
          <Tooltip contentStyle={tooltipStyle} cursor={{ stroke: GRID }} />
          <Area
            type="monotone"
            dataKey="value"
            stroke={INDIGO}
            strokeWidth={2}
            fill={`url(#widgetAreaFill-${widget.id})`}
            animationDuration={280}
          />
        </AreaChart>
      ) : kind === 'pie' ? (
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="label"
            innerRadius={48}
            outerRadius={76}
            paddingAngle={3}
            stroke="none"
            animationDuration={280}
          >
            {data.map((_, i) => (
              <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
            ))}
          </Pie>
          <Tooltip contentStyle={tooltipStyle} />
        </PieChart>
      ) : (
        <BarChart data={data} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
          <XAxis dataKey="label" stroke={AXIS} fontSize={11} tickLine={false} axisLine={{ stroke: GRID }} />
          <YAxis stroke={AXIS} fontSize={11} tickLine={false} axisLine={false} width={32} />
          <Tooltip contentStyle={tooltipStyle} cursor={{ fill: GRID }} />
          <Bar dataKey="value" radius={[4, 4, 0, 0]} animationDuration={280}>
            {data.map((_, i) => (
              <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
            ))}
          </Bar>
        </BarChart>
      )}
    </ResponsiveContainer>
  )
}

const stateStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  color: 'var(--text-tertiary)',
  fontSize: 14,
  height: 180,
}

const cellHead: React.CSSProperties = {
  textAlign: 'left',
  padding: '8px 10px',
  color: 'var(--text-secondary)',
  fontWeight: 500,
  fontSize: 12,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
  borderBottom: '1px solid var(--border-light)',
}

const cellBody: React.CSSProperties = {
  padding: '8px 10px',
  color: 'var(--text-primary)',
  fontSize: 13,
  borderBottom: '1px solid var(--border-light)',
}
