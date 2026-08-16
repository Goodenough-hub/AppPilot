import { useState } from 'react'
import { Star, Copy, Check, Pencil, Trash2 } from 'lucide-react'
import type { Item } from '@/api/hub'
import { timeAgo, domainOf } from '@/utils/format'
import { copyText } from '@/utils/copy'
import { useToast } from '@/components/ui/Toast'

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
  const toast = useToast()

  const doCopy = async () => {
    if (!item.content) return
    try {
      await copyText(item.content)
      setCopied(true)
      setTimeout(() => setCopied(false), 1200)
      toast.show('已复制')
    } catch {
      toast.show('复制失败')
    }
  }

  return (
    <article
      className="item-card"
      style={{
        background: 'var(--paper-lift)',
        border: '1px solid var(--rule)',
        borderRadius: 12,
        padding: '16px',
        height: '100%',
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

      {/* title：四列窄卡内截断单行，完整名见 tooltip */}
      <div
        title={item.title}
        style={{
          fontSize: 'var(--fs-sm)', color: 'var(--ink)', fontWeight: 500,
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'
        }}
      >
        {item.title}
      </div>

      {/* url subtitle */}
      {item.url && (
        <a href={item.url} target="_blank" rel="noopener noreferrer"
          style={{ fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-dim)' }}>
          ↗ {domain || item.url}
        </a>
      )}

      {/* content block：整块可点击复制，右上角常驻 copy 按钮 */}
      {item.content && (
        <div
          className="content-block"
          role="button"
          tabIndex={0}
          title="点击复制内容"
          onClick={() => void doCopy()}
          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); void doCopy() } }}
          style={{
            position: 'relative',
            background: 'var(--paper-sink)',
            border: '1px solid var(--rule)',
            borderRadius: 8,
            padding: '12px 40px 12px 14px',
            cursor: 'pointer',
            fontSize: 'var(--fs-sm)',
            fontFamily: 'var(--font-mono)',
            color: 'var(--ink-mid)',
            whiteSpace: 'pre-wrap',
            display: '-webkit-box', WebkitLineClamp: 3, WebkitBoxOrient: 'vertical',
            overflow: 'hidden'
          }}
        >
          {item.content}
          <button
            aria-label={copied ? '已复制' : '复制内容'}
            onClick={(e) => { e.stopPropagation(); void doCopy() }}
            style={{
              position: 'absolute', top: 8, right: 8, display: 'flex',
              padding: 4, borderRadius: 6, background: 'var(--paper-lift)',
              color: copied ? 'var(--accent)' : 'var(--ink-dim)', transition: 'color 140ms ease'
            }}
          >
            {copied ? <Check size={13} /> : <Copy size={13} />}
          </button>
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

      {/* actions: hover 时显现（触屏常驻）；copy 在内容块上常驻，不在这里重复 */}
      <div className="card-actions rule-t" style={{ paddingTop: 12, display: 'flex', gap: 18, fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-mid)' }}>
        <button onClick={() => onEdit(item)} className="card-action"><Pencil size={13} /> edit</button>
        <button onClick={() => onDelete(item.id)} className="card-action danger" style={{ marginLeft: 'auto' }}><Trash2 size={13} /> delete</button>
      </div>
    </article>
  )
}
