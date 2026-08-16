import { useEffect, useState, type FormEvent } from 'react'
import { Dialog, DialogTitle, DialogClose } from '@/components/ui/Dialog'
import type { Item, ItemInput, ItemType } from '@/api/hub'
import { parseRepo, fetchRepoInfo } from '@/utils/github'
import { useToast } from '@/components/ui/Toast'

interface Props {
  open: boolean
  onOpenChange: (v: boolean) => void
  initial?: Item
  onSubmit: (input: ItemInput) => Promise<void>
}

const TYPES: { value: ItemType; label: string }[] = [
  { value: 'bookmark', label: 'Bookmark' },
  { value: 'prompt', label: 'Prompt' },
  { value: 'skill', label: 'Skill' }
]

export function ItemDialog({ open, onOpenChange, initial, onSubmit }: Props) {
  const [type, setType] = useState<ItemType>('bookmark')
  const [title, setTitle] = useState('')
  const [url, setUrl] = useState('')
  const [content, setContent] = useState('')
  const [tags, setTags] = useState('')
  const [favorite, setFavorite] = useState(false)
  const [saving, setSaving] = useState(false)
  const [fetching, setFetching] = useState(false)
  const toast = useToast()

  useEffect(() => {
    if (!open) return
    if (initial) {
      setType(initial.type)
      setTitle(initial.title)
      setUrl(initial.url ?? '')
      setContent(initial.content ?? '')
      setTags(initial.tags.join(', '))
      setFavorite(initial.favorite)
    } else {
      setType('bookmark'); setTitle(''); setUrl(''); setContent(''); setTags(''); setFavorite(false)
    }
  }, [open, initial])

  const canGh = type === 'skill' && !!parseRepo(url)

  const runGh = async () => {
    const parsed = parseRepo(url)
    if (!parsed) return
    setFetching(true)
    try {
      const info = await fetchRepoInfo(parsed.owner, parsed.repo)
      setTitle(info.title)
      if (info.content) setContent(info.content)
      const existing = tags.split(',').map(s => s.trim()).filter(Boolean)
      const merged = Array.from(new Set([...existing, ...info.tags]))
      setTags(merged.join(', '))
      toast.show('已从 GitHub 拉取')
    } catch (e: any) {
      toast.show(`GitHub 拉取失败：${e.message}`)
    } finally {
      setFetching(false)
    }
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      await onSubmit({
        type, title,
        url: url.trim() || null,
        content: content.trim() || null,
        tags: tags.split(',').map(s => s.trim()).filter(Boolean),
        favorite
      })
      onOpenChange(false)
    } catch (e: any) {
      toast.show(`保存失败：${e?.response?.data?.error ?? e?.message ?? 'unknown'}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogClose />
      <DialogTitle>{initial ? '编辑条目' : '新增条目'}</DialogTitle>
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8 }}>
          {TYPES.map((t) => (
            <label key={t.value} style={{
              padding: '8px 12px', borderRadius: 6, textAlign: 'center', cursor: 'pointer',
              border: `1px solid ${type === t.value ? 'var(--accent)' : 'var(--rule)'}`,
              color: type === t.value ? 'var(--accent)' : 'var(--ink-mid)',
              fontSize: 'var(--fs-sm)'
            }}>
              <input type="radio" name="type" value={t.value} checked={type === t.value}
                onChange={() => setType(t.value)} style={{ display: 'none' }} />
              {t.label}
            </label>
          ))}
        </div>

        <div>
          <label style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>标题</label>
          <input required value={title} onChange={(e) => setTitle(e.target.value)} style={{ width: '100%' }} />
        </div>

        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 4 }}>
            <label style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)' }}>URL</label>
            {canGh && (
              <button type="button" onClick={runGh} disabled={fetching}
                style={{ fontSize: 'var(--fs-xs)', color: 'var(--accent)' }}>
                {fetching ? '拉取中…' : '从 GitHub 拉取'}
              </button>
            )}
          </div>
          <input type="url" value={url} onChange={(e) => setUrl(e.target.value)} style={{ width: '100%' }} />
        </div>

        <div>
          <label style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>内容 / 描述</label>
          <textarea rows={5} value={content} onChange={(e) => setContent(e.target.value)}
            style={{ width: '100%', fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-xs)' }} />
        </div>

        <div>
          <label style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>标签（逗号分隔）</label>
          <input value={tags} onChange={(e) => setTags(e.target.value)} style={{ width: '100%' }} />
        </div>

        <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 'var(--fs-sm)', color: 'var(--ink-mid)' }}>
          <input type="checkbox" checked={favorite} onChange={(e) => setFavorite(e.target.checked)} style={{ width: 'auto' }} />
          加入收藏
        </label>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 8 }}>
          <button type="button" onClick={() => onOpenChange(false)} style={{ color: 'var(--ink-mid)' }}>取消</button>
          <button type="submit" className="btn-primary" disabled={saving}>{saving ? '保存中…' : '保存'}</button>
        </div>
      </form>
    </Dialog>
  )
}
