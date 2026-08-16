import { Sun, Moon, Download, Upload, Plus } from 'lucide-react'
import { useTheme } from '@/hooks/useTheme'

export interface HeaderProps {
  totalCount: number
  starredCount: number
  typeCount: number
  onAdd: () => void
  onExport: () => void
  onImport: () => void
}

export function Header({ totalCount, starredCount, typeCount, onAdd, onExport, onImport }: HeaderProps) {
  const { theme, toggle } = useTheme()
  return (
    <header style={{ padding: '32px 0 16px' }}>
      <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: 24, marginBottom: 24 }}>
        <h1 className="font-serif italic" style={{ fontSize: 'var(--fs-xl)', margin: 0, lineHeight: 1.1 }}>Hub</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: 24, fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)' }}>
          <span>{totalCount}</span>
          <span>·</span>
          <span>{starredCount} ◆</span>
          <span>·</span>
          <span>{typeCount}</span>
          <button aria-label="切换主题" onClick={toggle} className="text-mid" style={{ display: 'flex', padding: 4 }}>
            {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
          </button>
          <button aria-label="导出" onClick={onExport} className="text-mid" style={{ display: 'flex', padding: 4 }}><Download size={16} /></button>
          <button aria-label="导入" onClick={onImport} className="text-mid" style={{ display: 'flex', padding: 4 }}><Upload size={16} /></button>
          <button onClick={onAdd} className="btn-primary" style={{ padding: '6px 12px', fontSize: 'var(--fs-sm)', display: 'flex', alignItems: 'center', gap: 6 }}><Plus size={14} /> 新增</button>
        </div>
      </div>
      <div className="rule-b" />
    </header>
  )
}
