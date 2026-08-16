import { Star, Pencil, Trash2, Globe } from 'lucide-react'
import type { Item } from '@/api/hub'
import { domainOf } from '@/utils/format'

/** 书签紧凑行：图标 + 标题链接 + 域名 + hover 显现的操作（收藏/编辑/删除）。
 *  书签只为跳转服务，不用卡片；prompt/skill 仍用 ItemCard。 */
export function BookmarkRow({
  item, onToggleFav, onEdit, onDelete, onTagClick
}: {
  item: Item
  onToggleFav: (id: number, next: boolean) => void
  onEdit: (item: Item) => void
  onDelete: (id: number) => void
  onTagClick: (tag: string) => void
}) {
  const domain = domainOf(item.url)

  return (
    <div
      className="bookmark-row"
      style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 10px' }}
    >
      <Globe size={14} style={{ color: 'var(--ink-dim)', flexShrink: 0 }} />

      {item.url ? (
        <a
          href={item.url}
          target="_blank"
          rel="noopener noreferrer"
          title={item.title}
          style={{
            fontSize: 'var(--fs-sm)', color: 'var(--ink)', minWidth: 0,
            overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'
          }}
        >
          {item.title}
        </a>
      ) : (
        <span
          style={{
            fontSize: 'var(--fs-sm)', color: 'var(--ink)', minWidth: 0,
            overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'
          }}
        >
          {item.title}
        </span>
      )}

      {item.tags.map((t) => (
        <button
          key={t}
          onClick={() => onTagClick(t)}
          className="chip"
          style={{
            fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', flexShrink: 0,
            color: 'var(--ink-dim)', background: 'var(--paper-sink)',
            border: '1px solid transparent', borderRadius: 999, padding: '1px 8px'
          }}
        >
          #{t}
        </button>
      ))}

      <span style={{ flex: 1 }} />

      {domain && (
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-xs)', color: 'var(--ink-dim)', flexShrink: 0 }}>
          {domain}
        </span>
      )}

      {item.favorite && (
        <Star aria-label="已收藏" size={13} fill="currentColor" style={{ color: 'var(--accent)', flexShrink: 0 }} />
      )}

      <div
        className="card-actions"
        style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-mid)', flexShrink: 0 }}
      >
        <button
          aria-label={item.favorite ? '取消收藏' : '收藏'}
          aria-pressed={item.favorite}
          onClick={() => onToggleFav(item.id, !item.favorite)}
          className="card-action"
          style={{ color: item.favorite ? 'var(--accent)' : undefined }}
        >
          <Star size={13} fill={item.favorite ? 'currentColor' : 'none'} />
        </button>
        <button aria-label={`编辑 ${item.title}`} onClick={() => onEdit(item)} className="card-action">
          <Pencil size={13} />
        </button>
        <button aria-label={`删除 ${item.title}`} onClick={() => onDelete(item.id)} className="card-action danger">
          <Trash2 size={13} />
        </button>
      </div>
    </div>
  )
}
