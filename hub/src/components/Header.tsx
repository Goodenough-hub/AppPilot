import { Sun, Moon, Download, Upload, Plus, FolderPlus } from 'lucide-react'
import { useTheme } from '@/hooks/useTheme'

export interface HeaderProps {
  totalCount: number
  starredCount: number
  typeCount: number
  onAdd: () => void
  onAddFolder: () => void
  onExport: () => void
  onImport: () => void
}

export function Header({ totalCount, starredCount, typeCount, onAdd, onAddFolder, onExport, onImport }: HeaderProps) {
  const { theme, toggle } = useTheme()
  return (
    <header style={{ padding: '32px 0 16px' }}>
      <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: 24, marginBottom: 24, flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 16 }}>
          <h1 className="font-serif italic font-display" style={{ fontSize: 'var(--fs-xl)', margin: 0 }}>Hub</h1>
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-xs)', color: 'var(--ink-dim)', whiteSpace: 'nowrap' }}>
            {totalCount} items · {starredCount} starred · {typeCount} types
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <button aria-label="切换主题" onClick={toggle} className="icon-btn">
            {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
          </button>
          <button aria-label="导出" onClick={onExport} className="icon-btn"><Download size={16} /></button>
          <button aria-label="导入" onClick={onImport} className="icon-btn"><Upload size={16} /></button>
          <button aria-label="新建文件夹" onClick={onAddFolder} className="icon-btn" style={{ marginLeft: 8 }}><FolderPlus size={16} /></button>
          <button onClick={onAdd} className="btn-primary" style={{ padding: '6px 14px', fontSize: 'var(--fs-sm)', display: 'flex', alignItems: 'center', gap: 6 }}><Plus size={14} /> 新增</button>
        </div>
      </div>
      <div className="rule-b" />
    </header>
  )
}
