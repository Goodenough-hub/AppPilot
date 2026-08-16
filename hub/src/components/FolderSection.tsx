import type { DragEvent, ReactNode } from 'react'
import { ChevronRight, ChevronDown, Folder, Inbox, Pencil, Trash2 } from 'lucide-react'

interface Props {
  /** 文件夹名；'' 表示未分类 */
  name: string
  count: number
  collapsed: boolean
  onToggle: () => void
  /** 仅在文件夹已登记（有 id）时提供；未分类/孤儿名无管理操作 */
  onRename?: () => void
  onDelete?: () => void
  /** 紧凑模式（书签行列表）：缩小条目间距 */
  dense?: boolean
  /** 整个组区块（含头部/空态区）的拖放接收：拖入=移动到该文件夹。条目卡上的 drop 优先（e.stopPropagation） */
  emptyDrop?: {
    onDragOver: (e: DragEvent) => void
    onDrop: (e: DragEvent) => void
  }
  children: ReactNode
}

/** 文件夹分组：头部点击收起/展开；hover 显现重命名/删除操作；组区块可接收跨组拖放 */
export function FolderSection({ name, count, collapsed, onToggle, onRename, onDelete, dense, emptyDrop, children }: Props) {
  const uncategorized = name === ''
  return (
    <section
      {...emptyDrop}
      style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
    >
      <div
        className="folder-header"
        style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 2px' }}
      >
        <button
          onClick={onToggle}
          aria-expanded={!collapsed}
          aria-label={collapsed ? `展开 ${name || '未分类'}` : `收起 ${name || '未分类'}`}
          style={{
            display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0,
            color: 'var(--ink)', textAlign: 'left'
          }}
        >
          {collapsed ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
          {uncategorized
            ? <Inbox size={15} style={{ color: 'var(--ink-dim)', flexShrink: 0 }} />
            : <Folder size={15} style={{ color: 'var(--accent)', flexShrink: 0 }} />}
          <span style={{ fontSize: 'var(--fs-md)', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {uncategorized ? '未分类' : name}
          </span>
          <sup style={{ fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-dim)' }}>
            {count}
          </sup>
        </button>
        {(onRename || onDelete) && (
          <div className="folder-actions" style={{ display: 'flex', gap: 14, fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)' }}>
            {onRename && (
              <button onClick={onRename} aria-label={`重命名 ${name}`} className="card-action" style={{ color: 'var(--ink-dim)' }}>
                <Pencil size={13} /> rename
              </button>
            )}
            {onDelete && (
              <button onClick={onDelete} aria-label={`删除 ${name}`} className="card-action danger" style={{ color: 'var(--ink-dim)' }}>
                <Trash2 size={13} />
              </button>
            )}
          </div>
        )}
      </div>

      {!collapsed && (
        count === 0 ? (
          /* 空文件夹：给足可命中的 drop 高度（HTML5 拖拽 hit-test 需要真实可见的元素区域），
             虚线框暗示可拖入 */
          <div style={{
            padding: '8px 2px 8px 26px', minHeight: 64, display: 'flex', alignItems: 'center',
            fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-dim)',
            border: '1px dashed var(--rule)', borderRadius: 8
          }}>
            空文件夹 · 可拖入条目
          </div>
        ) : dense ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {children}
          </div>
        ) : (
          /* prompt/skill 卡片：桌面一行四列，窄屏自适应降列 */
          <div className="card-grid">
            {children}
          </div>
        )
      )}
    </section>
  )
}
