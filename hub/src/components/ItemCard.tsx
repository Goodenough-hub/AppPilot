import { useState } from 'react'
import { Star, Copy, Check, Pencil, Trash2 } from 'lucide-react'
import type { Item } from '@/api/hub'
import { timeAgo, domainOf } from '@/utils/format'
import { copyText } from '@/utils/copy'

const TYPE_LABEL: Record<Item['type'], string> = {
  bookmark: 'BOOKMARK',
  prompt: 'PROMPT',
  skill: 'SKILL'
}

export function ItemCard({
  item, index = 0, onToggleFav, onEdit, onDelete, onTagClick
}: {
  item: Item
  index?: number
  onToggleFav: (id: number, next: boolean) => void
  onEdit: (item: Item) => void
  onDelete: (id: number) => void
  onTagClick: (tag: string) => void
}) {
  const [copied, setCopied] = useState(false)
  const domain = domainOf(item.url)

  const doCopy = async () => {
    if (!item.content) return
    await copyText(item.content)
    setCopied(true)
    setTimeout(() => setCopied(false), 1200)
  }

  return (
    <article
      className="item-card"
      style={{
        background: 'var(--paper-lift)',
        border: '1px solid var(--rule)',
        borderRadius: 12,
        padding: '20px 24px',
        display: 'flex', flexDirection: 'column', gap: 12,
        animationDelay: `${Math.min(index, 8) * 35}ms`
      }}
    >
      {/* header row */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
        <div style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', letterSpacing: '0.08em' }}>
          <span className="text-accent">{TYPE_LABEL[item.type]}</span>
          <span style={{ letterSpacing: 0 }}> · {timeAgo(item.updatedAt)}</span>
        </div>
        <button
          aria-label={item.favorite ? '取消收藏' : '收藏'}
          aria-pressed={item.favorite}
          onClick={() => onToggleFav(item.id, !item.favorite)}
          className="fav-btn"
          style={{ display: 'flex', padding: 2, color: item.favorite ? 'var(--accent)' : 'var(--ink-dim)', transition: 'color 140ms ease' }}
        >
          <Star size={15} fill={item.favorite ? 'currentColor' : 'none'} />
        </button>
      </div>

      {/* title */}
      <div style={{ fontSize: 'var(--fs-md)', color: 'var(--ink)', fontWeight: 500 }}>
        {item.title}
      </div>

      {/* url subtitle */}
      {item.url && (
        <a href={item.url} target="_blank" rel="noopener noreferrer"
          style={{ fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-dim)' }}>
          ↗ {domain || item.url}
        </a>
      )}

      {/* content preview */}
      {item.content && (
        <div style={{
          fontSize: 'var(--fs-sm)', color: 'var(--ink-mid)',
          display: '-webkit-box', WebkitLineClamp: 3, WebkitBoxOrient: 'vertical',
          overflow: 'hidden'
        }}>
          {item.content}
        </div>
      )}

      {/* tags */}
      {item.tags.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {item.tags.map((t) => (
            <button key={t} onClick={() => onTagClick(t)}
              className="chip"
              style={{
                fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)',
                color: 'var(--ink-dim)', background: 'var(--paper-sink)',
                border: '1px solid transparent',
                borderRadius: 999, padding: '3px 10px'
              }}>
              #{t}
            </button>
          ))}
        </div>
      )}

      {/* actions: hover 时显现（触屏常驻） */}
      <div className="card-actions rule-t" style={{ paddingTop: 12, display: 'flex', gap: 18, fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-mid)' }}>
        {item.content && (
          <button onClick={doCopy} className="card-action" style={{ color: copied ? 'var(--accent)' : undefined }}>
            {copied ? <><Check size={13} /> copied</> : <><Copy size={13} /> copy</>}
          </button>
        )}
        <button onClick={() => onEdit(item)} className="card-action"><Pencil size={13} /> edit</button>
        <button onClick={() => onDelete(item.id)} className="card-action danger" style={{ marginLeft: 'auto' }}><Trash2 size={13} /> delete</button>
      </div>
    </article>
  )
}
