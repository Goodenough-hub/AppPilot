import { useEffect, useState, type FormEvent } from 'react'
import { Dialog, DialogTitle, DialogClose } from '@/components/ui/Dialog'
import { useToast } from '@/components/ui/Toast'

interface Props {
  open: boolean
  onOpenChange: (v: boolean) => void
  /** 对话框标题，如「新建文件夹」/「重命名文件夹」 */
  title: string
  /** 重命名时的初始名 */
  initial?: string
  onSubmit: (name: string) => Promise<void>
}

/** 文件夹新建/重命名共用的单输入框对话框 */
export function FolderDialog({ open, onOpenChange, title, initial, onSubmit }: Props) {
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)
  const toast = useToast()

  useEffect(() => {
    if (open) setName(initial ?? '')
  }, [open, initial])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    setSaving(true)
    try {
      await onSubmit(trimmed)
      onOpenChange(false)
    } catch (err: any) {
      const status = err?.response?.status
      const msg = status === 409 ? '已存在同名文件夹' : (err?.response?.data?.error ?? err?.message ?? 'unknown')
      toast.show(`保存失败：${msg}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogClose />
      <DialogTitle>{title}</DialogTitle>
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div>
          <label htmlFor="folder-name" style={{ fontSize: 'var(--fs-xs)', color: 'var(--ink-mid)', display: 'block', marginBottom: 4 }}>
            名称
          </label>
          <input
            id="folder-name"
            required
            maxLength={200}
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            style={{ width: '100%' }}
          />
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 8 }}>
          <button type="button" onClick={() => onOpenChange(false)} style={{ color: 'var(--ink-mid)' }}>取消</button>
          <button type="submit" className="btn-primary" disabled={saving}>{saving ? '保存中…' : '保存'}</button>
        </div>
      </form>
    </Dialog>
  )
}
