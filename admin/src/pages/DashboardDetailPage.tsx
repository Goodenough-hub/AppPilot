import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Pencil, Check, Plus } from 'lucide-react'
import {
  Responsive as ResponsiveGridBase,
  WidthProvider,
  type Layout,
  type LayoutItem,
} from 'react-grid-layout/legacy'
import 'react-grid-layout/css/styles.css'
import {
  listDashboards,
  getDashboard,
  updateWidget,
  createWidget,
  deleteWidget,
  type Dashboard,
  type Widget,
} from '../api/dashboard'
import WidgetCard from '../components/WidgetCard'
import AddWidgetDrawer from '../components/AddWidgetDrawer'

const ResponsiveGridLayout = WidthProvider(ResponsiveGridBase)

const BREAKPOINTS = { lg: 0 }
const COLS = { lg: 12 }
const ROW_HEIGHT = 60
const MARGIN: [number, number] = [12, 12]

export default function DashboardDetailPage() {
  const { app } = useParams<{ app: string }>()
  const [dashboards, setDashboards] = useState<Dashboard[]>([])
  const [dashboard, setDashboard] = useState<Dashboard | null>(null)
  const [widgets, setWidgets] = useState<Widget[]>([])
  const [isEditing, setIsEditing] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  // react-grid-layout fires onLayoutChange once after mount (compacted layout).
  // Ignore that initial fire so we don't overwrite stored positions.
  const skipLayoutSave = useRef(true)

  useEffect(() => {
    if (!app) return
    setLoading(true)
    setError('')
    skipLayoutSave.current = true
    let cancelled = false
    listDashboards()
      .then(list => {
        if (cancelled) return
        setDashboards(list)
        const match = list.find(d => d.app === app) ?? null
        setDashboard(match)
        if (!match) {
          setWidgets([])
          setLoading(false)
          return null
        }
        return getDashboard(match.id).then(({ dashboard: d, widgets: ws }) => {
          if (cancelled) return
          setDashboard(d)
          setWidgets(ws)
          setLoading(false)
        })
      })
      .catch(err => {
        if (cancelled) return
        setError(err?.response?.data?.error || err?.message || '加载失败')
        setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [app])

  const layout: Layout = useMemo(
    () =>
      widgets.map(w => ({
        i: w.id,
        x: w.gridX,
        y: w.gridY,
        w: w.gridW,
        h: w.gridH,
      })),
    [widgets]
  )

  const handleLayoutChange = (next: Layout) => {
    if (skipLayoutSave.current) {
      skipLayoutSave.current = false
      return
    }
    if (!dashboard || !isEditing) return
    const changes: { widget: Widget; item: LayoutItem }[] = []
    for (const w of widgets) {
      const item = next.find(n => n.i === w.id)
      if (!item) continue
      if (
        item.x === w.gridX &&
        item.y === w.gridY &&
        item.w === w.gridW &&
        item.h === w.gridH
      ) {
        continue
      }
      changes.push({ widget: w, item })
    }
    if (changes.length === 0) return
    setWidgets(prev =>
      prev.map(w => {
        const c = changes.find(c => c.widget.id === w.id)
        return c
          ? { ...w, gridX: c.item.x, gridY: c.item.y, gridW: c.item.w, gridH: c.item.h }
          : w
      })
    )
    for (const { widget, item } of changes) {
      updateWidget(dashboard.id, widget.id, {
        gridX: item.x,
        gridY: item.y,
        gridW: item.w,
        gridH: item.h,
      }).catch(err => setError(err?.response?.data?.error || '布局保存失败'))
    }
  }

  const handleDelete = (widget: Widget) => {
    if (!dashboard) return
    if (!window.confirm(`确认删除「${widget.title}」？`)) return
    const id = dashboard.id
    deleteWidget(id, widget.id)
      .then(() => setWidgets(prev => prev.filter(w => w.id !== widget.id)))
      .catch(err => setError(err?.response?.data?.error || '删除失败'))
  }

  const handleAdd = (input: {
    type: string
    title: string
    dataSource: string
    gridW: number
    gridH: number
  }) => {
    if (!dashboard) return
    const id = dashboard.id
    const bottomY = widgets.reduce((m, w) => Math.max(m, w.gridY + w.gridH), 0)
    createWidget(id, {
      type: input.type as Widget['type'],
      title: input.title,
      dataSource: input.dataSource,
      config: {},
      gridX: 0,
      gridY: bottomY,
      gridW: input.gridW,
      gridH: input.gridH,
      sortOrder: widgets.length,
    })
      .then(w => {
        setWidgets(prev => [...prev, w])
        setDrawerOpen(false)
      })
      .catch(err => setError(err?.response?.data?.error || '添加失败'))
  }

  return (
    <div className="animate-fade-in-up">
      <header className="admin-page-header">
        <div>
          <h1>{dashboard?.title || app || '应用看板'}</h1>
          <div className="subtitle">
            {dashboard?.description || '按应用维度查看关键指标与趋势。'}
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <button
            type="button"
            className={isEditing ? 'btn-primary' : 'btn-ghost'}
            onClick={() => setIsEditing(e => !e)}
            style={{ display: 'inline-flex', alignItems: 'center', gap: 6, padding: '8px 14px', borderRadius: 10, fontSize: 14 }}
          >
            {isEditing ? <Check size={16} /> : <Pencil size={16} />}
            {isEditing ? '完成' : '编辑布局'}
          </button>
          {isEditing && (
            <button
              type="button"
              className="btn-primary"
              onClick={() => setDrawerOpen(true)}
              style={{ display: 'inline-flex', alignItems: 'center', gap: 6, padding: '8px 14px', borderRadius: 10, fontSize: 14 }}
            >
              <Plus size={16} />
              添加 Widget
            </button>
          )}
        </div>
      </header>

      {dashboards.length > 0 && (
        <div className="app-tabs" role="tablist">
          {dashboards.map(d => (
            <Link
              key={d.id}
              to={`/admin/dashboards/${d.app}`}
              role="tab"
              aria-selected={d.app === app}
              className={`app-tab${d.app === app ? ' active' : ''}`}
            >
              {d.title}
            </Link>
          ))}
        </div>
      )}

      {error && <div className="form-error" style={{ marginBottom: 12 }}>{error}</div>}
      {loading && <div style={{ color: 'var(--text-tertiary)' }}>加载中…</div>}
      {!loading && !dashboard && (
        <div className="glass-panel" style={{ padding: 24, color: 'var(--text-tertiary)' }}>
          未找到应用「{app}」的看板。
        </div>
      )}

      {dashboard && !loading && (
        <div className="glass-panel" style={{ padding: 16, minHeight: 200 }}>
          {widgets.length === 0 ? (
            <div style={{ padding: 24, color: 'var(--text-tertiary)', textAlign: 'center' }}>
              暂无 widget。{isEditing && '点击「添加 Widget」创建第一个。'}
            </div>
          ) : (
            <ResponsiveGridLayout
              className="layout"
              layouts={{ lg: layout }}
              breakpoints={BREAKPOINTS}
              cols={COLS}
              rowHeight={ROW_HEIGHT}
              margin={MARGIN}
              containerPadding={[0, 0]}
              isDraggable={isEditing}
              isResizable={isEditing}
              compactType="vertical"
              useCSSTransforms
              onLayoutChange={handleLayoutChange}
            >
              {widgets.map(w => (
                <div key={w.id}>
                  <WidgetCard
                    widget={w}
                    isEditing={isEditing}
                    onDelete={() => handleDelete(w)}
                  />
                </div>
              ))}
            </ResponsiveGridLayout>
          )}
        </div>
      )}

      <AddWidgetDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        onAdd={handleAdd}
      />

      <style>{`
        .app-tabs {
          display: flex;
          gap: 8px;
          margin-bottom: 16px;
          flex-wrap: wrap;
        }
        .app-tab {
          padding: 8px 16px;
          border-radius: 10px;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-secondary);
          background: var(--surface);
          border: 1px solid var(--border-light);
          text-decoration: none;
          transition: all 0.18s ease;
        }
        .app-tab:hover {
          color: var(--text-primary);
          border-color: var(--border-glow);
        }
        .app-tab.active {
          background: var(--primary-gradient);
          color: #fff;
          border: none;
          box-shadow: var(--shadow-glow);
        }
      `}</style>
    </div>
  )
}
