import { useEffect, useMemo, useState } from 'react'
import { X, Search } from 'lucide-react'
import { listDataSources, type DataSourceMeta } from '../api/dashboard'

interface AddWidgetDrawerProps {
  open: boolean
  onClose: () => void
  onAdd: (widget: {
    type: string
    title: string
    dataSource: string
    gridW: number
    gridH: number
  }) => void
}

type WidgetType = 'stat' | 'chart' | 'table'

const TYPE_META: Record<WidgetType, { label: string; gridW: number; gridH: number }> = {
  stat: { label: '数字', gridW: 2, gridH: 1 },
  chart: { label: '图表', gridW: 3, gridH: 2 },
  table: { label: '表格', gridW: 3, gridH: 2 },
}

export default function AddWidgetDrawer({ open, onClose, onAdd }: AddWidgetDrawerProps) {
  const [sources, setSources] = useState<DataSourceMeta[]>([])
  const [loadingSources, setLoadingSources] = useState(false)
  const [query, setQuery] = useState('')
  const [selectedSource, setSelectedSource] = useState<string>('')
  const [type, setType] = useState<WidgetType>('stat')
  const [title, setTitle] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    setLoadingSources(true)
    setError('')
    listDataSources()
      .then(setSources)
      .catch((err: any) => setError(err?.response?.data?.error || '数据源加载失败'))
      .finally(() => setLoadingSources(false))
  }, [open])

  // Reset transient state when re-opened.
  useEffect(() => {
    if (open) {
      setQuery('')
      setSelectedSource('')
      setType('stat')
      setTitle('')
      setError('')
    }
  }, [open])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return sources
    return sources.filter(
      s => s.key.toLowerCase().includes(q) || (s.description || '').toLowerCase().includes(q)
    )
  }, [sources, query])

  if (!open) return null

  const submit = () => {
    setError('')
    if (!selectedSource) {
      setError('请选择一个数据源')
      return
    }
    if (!title.trim()) {
      setError('请填写标题')
      return
    }
    const meta = TYPE_META[type]
    onAdd({
      type,
      title: title.trim(),
      dataSource: selectedSource,
      gridW: meta.gridW,
      gridH: meta.gridH,
    })
  }

  return (
    <div className="drawer-backdrop" onClick={onClose}>
      <div
        className="glass-panel drawer-panel"
        onClick={e => e.stopPropagation()}
        role="dialog"
        aria-label="添加 widget"
      >
        <div className="drawer-head">
          <h2>添加 widget</h2>
          <button className="btn-ghost" onClick={onClose} aria-label="关闭" style={{ padding: '6px 10px', borderRadius: 8 }}>
            <X size={18} />
          </button>
        </div>

        <div className="drawer-body">
          <section className="drawer-section">
            <label className="drawer-label">类型</label>
            <div className="type-selector">
              {(Object.keys(TYPE_META) as WidgetType[]).map(t => (
                <button
                  key={t}
                  className={`type-btn${type === t ? ' active' : ''}`}
                  onClick={() => setType(t)}
                  type="button"
                >
                  {TYPE_META[t].label}
                </button>
              ))}
            </div>
          </section>

          <section className="drawer-section">
            <label className="drawer-label">标题</label>
            <input
              className="form-input"
              placeholder="widget 标题"
              value={title}
              onChange={e => setTitle(e.target.value)}
              maxLength={64}
            />
          </section>

          <section className="drawer-section">
            <label className="drawer-label">数据源</label>
            <div className="search-box">
              <Search size={16} className="search-icon" />
              <input
                className="form-input search-input"
                placeholder="搜索数据源…"
                value={query}
                onChange={e => setQuery(e.target.value)}
              />
            </div>
            <div className="source-list">
              {loadingSources && <div className="drawer-empty">加载中…</div>}
              {!loadingSources && filtered.length === 0 && (
                <div className="drawer-empty">无匹配数据源</div>
              )}
              {filtered.map(s => (
                <button
                  key={s.key}
                  className={`source-item${selectedSource === s.key ? ' active' : ''}`}
                  onClick={() => setSelectedSource(s.key)}
                  type="button"
                >
                  <span className="source-key">{s.key}</span>
                  {s.description && <span className="source-desc">{s.description}</span>}
                </button>
              ))}
            </div>
          </section>
        </div>

        <div className="drawer-foot">
          {error && <div className="form-error" style={{ marginTop: 0, marginBottom: 8 }}>{error}</div>}
          <button className="btn-primary" onClick={submit} type="button">
            添加
          </button>
        </div>
      </div>

      <style>{`
        .drawer-backdrop {
          position: fixed;
          inset: 0;
          background: rgba(0,0,0,0.5);
          backdrop-filter: blur(4px);
          -webkit-backdrop-filter: blur(4px);
          z-index: 60;
          display: flex;
          justify-content: flex-end;
        }
        .drawer-panel {
          width: 380px;
          max-width: calc(100vw - 24px);
          height: 100%;
          border-radius: 0;
          border-left: 1px solid var(--border-glow);
          display: flex;
          flex-direction: column;
          animation: drawerSlideIn var(--dur-sheet) var(--ease-drawer);
        }
        @keyframes drawerSlideIn {
          from { transform: translateX(100%); opacity: 0.6; }
          to { transform: translateX(0); opacity: 1; }
        }
        .drawer-head {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 20px 20px 12px;
          border-bottom: 1px solid var(--border-light);
        }
        .drawer-head h2 { margin: 0; font-size: 18px; }
        .drawer-body {
          flex: 1;
          overflow-y: auto;
          padding: 16px 20px;
          display: flex;
          flex-direction: column;
          gap: 18px;
        }
        .drawer-section { display: flex; flex-direction: column; gap: 8px; }
        .drawer-label {
          font-size: 12px;
          text-transform: uppercase;
          letter-spacing: 0.05em;
          color: var(--text-secondary);
          font-weight: 500;
        }
        .type-selector { display: flex; gap: 8px; }
        .type-btn {
          flex: 1;
          padding: 10px 12px;
          border-radius: 10px;
          font-size: 14px;
        }
        .type-btn.active {
          background: var(--primary-gradient);
          border: none;
          color: #fff;
          box-shadow: var(--shadow-glow);
        }
        .search-box { position: relative; display: flex; align-items: center; }
        .search-icon {
          position: absolute;
          left: 12px;
          color: var(--text-tertiary);
          pointer-events: none;
        }
        .search-input { padding-left: 36px; }
        .source-list {
          display: flex;
          flex-direction: column;
          gap: 6px;
          max-height: 280px;
          overflow-y: auto;
          margin-top: 4px;
        }
        .source-item {
          text-align: left;
          padding: 10px 12px;
          border-radius: 10px;
          display: flex;
          flex-direction: column;
          gap: 2px;
          font-weight: 400;
        }
        .source-item.active {
          background: var(--surface-active);
          border-color: var(--border-glow);
        }
        .source-key { color: var(--text-primary); font-size: 14px; font-weight: 500; }
        .source-desc { color: var(--text-tertiary); font-size: 12px; }
        .drawer-empty {
          color: var(--text-tertiary);
          font-size: 14px;
          padding: 16px 12px;
          text-align: center;
        }
        .drawer-foot {
          padding: 16px 20px 20px;
          border-top: 1px solid var(--border-light);
          display: flex;
          flex-direction: column;
          align-items: stretch;
        }
        @media (prefers-reduced-motion: reduce) {
          .drawer-panel { animation: none; }
        }
      `}</style>
    </div>
  )
}
