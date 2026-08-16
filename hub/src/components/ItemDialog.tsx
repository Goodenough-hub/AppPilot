import { useEffect, useState, type FormEvent } from 'react'
import { Dialog, DialogTitle, DialogClose } from '@/components/ui/Dialog'
import type { Folder, Item, ItemInput, ItemType } from '@/api/hub'
import { parseRepo, fetchRepoInfo } from '@/utils/github'
import { useToast } from '@/components/ui/Toast'

interface Props {
  open: boolean
  onOpenChange: (v: boolean) => void
  initial?: Item
  /** 各类型下的文件夹目录（文件夹候选随对话框内所选 type 联动） */
  foldersByType: Record<ItemType, Folder[]>
  onSubmit: (input: ItemInput) => Promise<void>
}

const TYPES: { value: ItemType; label: string }[] = [
  { value: 'bookmark', label: 'Bookmark' },
  { value: 'prompt', label: 'Prompt' },
  { value: 'skill', label: 'Skill' }
]

function parseTags(input: string): string[] {
  return input.split(',').map(s => s.trim()).filter(Boolean)
}

export function ItemDialog({ open, onOpenChange, initial, foldersByType, onSubmit }: Props) {
  const [type, setType] = useState<ItemType>('bookmark')
  const [title, setTitle] = useState('')
  const [url, setUrl] = useState('')
  const [content, setContent] = useState('')
  const [tags, setTags] = useState('')
  const [folder, setFolder] = useState('')
  const [icon, setIcon] = useState('')
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
      setFolder(initial.folder)
      setIcon(initial.icon)
      setFavorite(initial.favorite)
    } else {
      setType('bookmark'); setTitle(''); setUrl(''); setContent(''); setTags(''); setFolder(''); setIcon(''); setFavorite(false)
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
      const existing = parseTags(tags)
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
        tags: parseTags(tags),
        folder: folder.trim(),
        icon: icon.trim(),
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
                onChange={() => { setType(t.value); setFolder(''); setIcon('') }} style={{
                  position: 'absolute',
                  width: 1,
                  height: 1,
                  padding: 0,
                  margin: -1,
                  overflow: 'hidden',
                  clip: 'rect(0, 0, 0, 0)',
                  whiteSpace: 'nowrap',
                  border: 0
                }} />
              {t.label}
            </label>
          ))}
        </div>

        <div>
          <label htmlFor="item-title" style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>标题</label>
          <input id="item-title" required value={title} onChange={(e) => setTitle(e.target.value)} style={{ width: '100%' }} />
        </div>

        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 4 }}>
            <label htmlFor="item-url" style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)' }}>URL</label>
            {canGh && (
              <button type="button" onClick={runGh} disabled={fetching}
                style={{ fontSize: 'var(--fs-xs)', color: 'var(--accent)' }}>
                {fetching ? '拉取中…' : '从 GitHub 拉取'}
              </button>
            )}
          </div>
          <input id="item-url" type="url" value={url} onChange={(e) => setUrl(e.target.value)} style={{ width: '100%' }} />
        </div>

        <div>
          <label htmlFor="item-content" style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>内容 / 描述</label>
          <textarea id="item-content" rows={5} value={content} onChange={(e) => setContent(e.target.value)}
            style={{ width: '100%', fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-xs)' }} />
        </div>

        <div>
          <label htmlFor="item-folder" style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>文件夹（留空为未分类）</label>
          <input id="item-folder" list="item-folder-candidates" value={folder} onChange={(e) => setFolder(e.target.value)} style={{ width: '100%' }} />
          <datalist id="item-folder-candidates">
            {foldersByType[type].map((f) => <option key={f.id} value={f.name} />)}
          </datalist>
        </div>

        {type === 'bookmark' && (
          <div>
            <label htmlFor="item-icon" style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>图标 URL（留空按站点自动探测）</label>
            <input id="item-icon" type="url" value={icon} onChange={(e) => setIcon(e.target.value)} style={{ width: '100%' }} />
          </div>
        )}

        <div>
          <label htmlFor="item-tags" style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>标签（逗号分隔）</label>
          <input id="item-tags" value={tags} onChange={(e) => setTags(e.target.value)} style={{ width: '100%' }} />
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
