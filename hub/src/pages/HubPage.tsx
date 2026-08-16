import { useState, useRef, type ChangeEvent } from 'react'
import { Header } from '@/components/Header'
import { TabBar } from '@/components/TabBar'
import { TagCloud } from '@/components/TagCloud'
import { ItemCard } from '@/components/ItemCard'
import { SearchBar } from '@/components/SearchBar'
import { ItemDialog } from '@/components/ItemDialog'
import { EmptyState } from '@/components/EmptyState'
import { AlertDialog } from '@/components/ui/AlertDialog'
import { useItems } from '@/hooks/useItems'
import { useFilter } from '@/hooks/useFilter'
import { useToast } from '@/components/ui/Toast'
import { itemsApi, type Item } from '@/api/hub'

export default function HubPage() {
  const { items, loading, create, update, remove, reload } = useItems()
  const { tab, setTab, tag, setTag, query, setQuery, filtered, counts, allTags } = useFilter(items)
  const toast = useToast()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Item | undefined>(undefined)
  const [confirmDeleteId, setConfirmDeleteId] = useState<number | null>(null)
  const [importMode, setImportMode] = useState<'merge' | 'replace' | null>(null)
  const importFileRef = useRef<HTMLInputElement>(null)

  const openAdd = () => { setEditing(undefined); setDialogOpen(true) }
  const openEdit = (it: Item) => { setEditing(it); setDialogOpen(true) }

  const submitItem = async (input: Parameters<typeof create>[0]) => {
    if (editing) {
      await update(editing.id, input)
      toast.show('已保存')
    } else {
      await create(input)
      toast.show('已新增')
    }
  }

  const confirmDelete = async () => {
    if (confirmDeleteId == null) return
    try {
      await remove(confirmDeleteId)
      toast.show('已删除')
    } catch (e: any) {
      toast.show(`删除失败：${e?.message ?? 'unknown'}`)
    } finally {
      setConfirmDeleteId(null)
    }
  }

  const exportJson = async () => {
    try {
      const data = await itemsApi.exportJson()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `hub-export-${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(a.href)
      toast.show(`已导出 ${data.length} 条`)
    } catch (e: any) {
      toast.show(`导出失败：${e?.message ?? 'unknown'}`)
    }
  }

  const onImportClick = () => importFileRef.current?.click()

  const onImportChosen = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const text = await file.text()
    e.target.value = ''
    try {
      const parsed = JSON.parse(text)
      if (!Array.isArray(parsed)) throw new Error('文件不是 JSON 数组')
      setImportMode('merge')
      ;(window as any).__hubImportPayload = parsed // 由 confirm dialog 消费
    } catch (err: any) {
      toast.show(`解析失败：${err.message}`)
    }
  }

  const runImport = async (mode: 'merge' | 'replace') => {
    const payload = (window as any).__hubImportPayload as Item[] | undefined
    if (!payload) return
    try {
      const res = await itemsApi.importJson(payload, mode)
      toast.show(`导入完成：${mode} × ${res.affected}`)
      await reload()
    } catch (e: any) {
      toast.show(`导入失败：${e?.message ?? 'unknown'}`)
    } finally {
      ;(window as any).__hubImportPayload = undefined
      setImportMode(null)
    }
  }

  return (
    <div style={{ maxWidth: 880, margin: '0 auto', padding: '0 32px 96px' }}>
      <Header
        totalCount={counts.all}
        starredCount={counts.starred}
        typeCount={Object.keys(counts).filter(k => k !== 'all' && k !== 'starred' && counts[k as keyof typeof counts] > 0).length}
        onAdd={openAdd}
        onExport={exportJson}
        onImport={onImportClick}
      />
      <input ref={importFileRef} type="file" accept="application/json" style={{ display: 'none' }} onChange={onImportChosen} />

      <TabBar active={tab} onChange={setTab} />
      <div className="rule-b" />
      <TagCloud tags={allTags} active={tag} onSelect={setTag} />
      <div className="rule-b" style={{ marginBottom: 24 }} />

      {loading ? (
        <EmptyState message="loading…" />
      ) : filtered.length === 0 ? (
        <EmptyState message={items.length === 0 ? '空空如也。点右上 + 新增第一条' : '无匹配条目'} />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {filtered.map((it) => (
            <ItemCard
              key={it.id}
              item={it}
              onToggleFav={(id, next) => update(id, { favorite: next })}
              onEdit={openEdit}
              onDelete={setConfirmDeleteId}
              onTagClick={setTag}
            />
          ))}
        </div>
      )}

      <footer style={{ marginTop: 96, padding: '24px 0', textAlign: 'center', fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-dim)' }}>
        hub v0.1 · built for one · goodenough, mmxxvi
      </footer>

      <SearchBar value={query} onChange={setQuery} />

      <ItemDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        initial={editing}
        onSubmit={submitItem}
      />

      <AlertDialog
        open={confirmDeleteId !== null}
        onOpenChange={(v) => !v && setConfirmDeleteId(null)}
        title="删除条目"
        description="删除后不可撤销，确认删除？"
        confirmText="删除"
        destructive
        onConfirm={confirmDelete}
      />

      <AlertDialog
        open={importMode !== null}
        onOpenChange={(v) => !v && setImportMode(null)}
        title="导入 JSON"
        description="merge：按 ID 合并（默认，安全）；replace：清空后全插入（不可撤销）。"
        confirmText="merge"
        cancelText="replace / 取消"
        onConfirm={() => runImport('merge')}
      />
    </div>
  )
}
