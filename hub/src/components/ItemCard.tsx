import { useState } from 'react'
import type { Item } from '@/api/hub'
import { timeAgo, domainOf } from '@/utils/format'
import { copyText } from '@/utils/copy'

const TYPE_LABEL: Record<Item['type'], string> = {
  bookmark: 'BOOKMARK',
  prompt: 'PROMPT',
  skill: 'SKILL'
}

export function ItemCard({
  item, onToggleFav, onEdit, onDelete, onTagClick
}: {
  item: Item
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
      style={{
        background: 'var(--paper-lift)',
        border: '1px solid var(--rule)',
        borderRadius: 8,
        padding: '20px 24px',
        display: 'flex', flexDirection: 'column', gap: 12
      }}
    >
      {/* header row */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
        <div style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)' }}>
          {TYPE_LABEL[item.type]} · {timeAgo(item.updatedAt)}
        </div>
        <button
          aria-label={item.favorite ? '取消收藏' : '收藏'}
          onClick={() => onToggleFav(item.id, !item.favorite)}
          style={{ color: item.favorite ? 'var(--accent)' : 'var(--ink-mid)', fontSize: 16 }}
        >
          {item.favorite ? '◆' : '◇'}
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
              style={{
                fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)',
                color: 'var(--ink-dim)', background: 'var(--paper-sink)',
                borderRadius: 4, padding: '2px 8px'
              }}>
              #{t}
            </button>
          ))}
        </div>
      )}

      {/* actions */}
      <div className="rule-t" style={{ paddingTop: 12, display: 'flex', gap: 16, fontSize: 'var(--fs-sm)', color: 'var(--ink-mid)' }}>
        {item.content && (
          <button onClick={doCopy} style={{ color: copied ? 'var(--accent)' : 'inherit' }}>
            {copied ? 'copied' : 'copy'}
          </button>
        )}
        <button onClick={() => onEdit(item)}>edit</button>
        <button onClick={() => onDelete(item.id)}>delete</button>
      </div>
    </article>
  )
}
