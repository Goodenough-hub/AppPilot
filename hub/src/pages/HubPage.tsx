import { useState, useRef, useMemo, type ChangeEvent } from 'react'
import { Header } from '@/components/Header'
import { TabBar } from '@/components/TabBar'
import { TagCloud } from '@/components/TagCloud'
import { ItemCard } from '@/components/ItemCard'
import { BookmarkRow } from '@/components/BookmarkRow'
import { SearchBar } from '@/components/SearchBar'
import { ItemDialog } from '@/components/ItemDialog'
import { FolderDialog } from '@/components/FolderDialog'
import { FolderSection } from '@/components/FolderSection'
import { EmptyState } from '@/components/EmptyState'
import { AlertDialog } from '@/components/ui/AlertDialog'
import { Dialog, DialogTitle } from '@/components/ui/Dialog'
import { useItems } from '@/hooks/useItems'
import { useFilter } from '@/hooks/useFilter'
import { useFolders } from '@/hooks/useFolders'
import { useCollapsedFolders } from '@/hooks/useCollapsedFolders'
import { useToast } from '@/components/ui/Toast'
import { groupByFolder } from '@/utils/group'
import { moveItem } from '@/utils/reorder'
import { itemsApi, type Item } from '@/api/hub'

/** 文件夹对话框状态：新建 / 重命名 */
type FolderDialogState = { mode: 'create' } | { mode: 'rename'; id: number; name: string } | null

export default function HubPage() {
  const { items, loading, create, update, remove, reload, reorder } = useItems()
  const { tab, setTab, tag, setTag, query, setQuery, filtered, counts, allTags } = useFilter(items)
  const { folders, create: createFolder, rename: renameFolder, remove: removeFolder, reload: reloadFolders } = useFolders()
  const { isCollapsed, toggle: toggleCollapsed } = useCollapsedFolders()
  const toast = useToast()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Item | undefined>(undefined)
  const [confirmDeleteId, setConfirmDeleteId] = useState<number | null>(null)
  const [importMode, setImportMode] = useState<'merge' | 'replace' | null>(null)
  const [importPayload, setImportPayload] = useState<Item[] | null>(null)
  const importFileRef = useRef<HTMLInputElement>(null)

  const [folderDialog, setFolderDialog] = useState<FolderDialogState>(null)
  const [confirmDeleteFolder, setConfirmDeleteFolder] = useState<{ id: number; name: string; count: number } | null>(null)

  // 书签拖拽排序：dragId 为被拖条目，dropHint 指示落点（某行的上/下半区）
  const [dragId, setDragId] = useState<number | null>(null)
  const [dropHint, setDropHint] = useState<{ id: number; after: boolean } | null>(null)

  // 当前 tab 的条目按文件夹分组；搜索/标签过滤激活时隐藏空分组（平时保留，让新建的空文件夹可见）
  const groups = useMemo(() => groupByFolder(filtered, folders[tab]), [filtered, folders, tab])
  const filtering = tag !== null || query.trim() !== ''
  const displayGroups = filtering ? groups.filter((g) => g.items.length > 0) : groups

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
    // 条目的 folder 可能被后端自动登记为新文件夹，刷新目录
    reloadFolders().catch(() => {})
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

  /** 书签行落点换算 + 持久化（只支持同文件夹内重排；跨组拖动直接忽略） */
  const dropOnRow = async (group: { folder: string; items: Item[] }, targetIndex: number, after: boolean) => {
    if (dragId == null) return
    const from = group.items.findIndex((x) => x.id === dragId)
    if (from < 0) return // 拖的是别的文件夹的条目
    let to = targetIndex + (after ? 1 : 0)
    if (from < to) to -= 1
    if (from === to) return
    const ids = moveItem(group.items.map((x) => x.id), from, to)
    try {
      await reorder(tab, group.folder, ids)
    } catch (e: any) {
      toast.show(`排序失败：${e?.message ?? 'unknown'}`)
    }
  }

  const rowDragProps = (group: { folder: string; items: Item[] }, it: Item, i: number) => ({
    draggable: true,
    onDragStart: (e: React.DragEvent) => {
      e.dataTransfer.effectAllowed = 'move'
      e.dataTransfer.setData('text/plain', String(it.id))
      setDragId(it.id)
    },
    onDragOver: (e: React.DragEvent) => {
      if (dragId == null) return
      e.preventDefault()
      const rect = e.currentTarget.getBoundingClientRect()
      setDropHint({ id: it.id, after: e.clientY > rect.top + rect.height / 2 })
    },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault()
      void dropOnRow(group, i, dropHint?.id === it.id ? dropHint.after : false)
      setDragId(null)
      setDropHint(null)
    },
    onDragEnd: () => { setDragId(null); setDropHint(null) }
  })

  const submitFolder = async (name: string) => {
    if (!folderDialog) return
    if (folderDialog.mode === 'create') {
      await createFolder(tab, name)
      toast.show('已新建文件夹')
    } else {
      await renameFolder(folderDialog.id, name)
      toast.show('已重命名')
      // 重命名级联更新了条目 folder，刷新条目
      await reload()
    }
  }

  const confirmDeleteFolderAction = async () => {
    if (!confirmDeleteFolder) return
    try {
      await removeFolder(confirmDeleteFolder.id)
      toast.show('已删除文件夹，条目已移至未分类')
      // 条目 folder 被服务端置空，刷新条目
      await reload()
    } catch (e: any) {
      toast.show(`删除失败：${e?.message ?? 'unknown'}`)
    } finally {
      setConfirmDeleteFolder(null)
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
      setImportPayload(parsed)
    } catch (err: any) {
      toast.show(`解析失败：${err.message}`)
    }
  }

  const runImport = async (mode: 'merge' | 'replace') => {
    const payload = importPayload
    if (!payload) return
    try {
      const res = await itemsApi.importJson(payload, mode)
      toast.show(`导入完成：${mode} × ${res.affected}`)
      await reload()
      reloadFolders().catch(() => {})
    } catch (e: any) {
      toast.show(`导入失败：${e?.message ?? 'unknown'}`)
    } finally {
      setImportPayload(null)
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
        onAddFolder={() => setFolderDialog({ mode: 'create' })}
        onExport={exportJson}
        onImport={onImportClick}
      />
      <input ref={importFileRef} type="file" accept="application/json" style={{ display: 'none' }} onChange={onImportChosen} />

      <TabBar active={tab} onChange={setTab} counts={counts} />
      <div className="rule-b" />
      <TagCloud tags={allTags} active={tag} onSelect={setTag} />
      <div className="rule-b" style={{ marginBottom: 24 }} />

      {loading ? (
        <EmptyState message="loading…" />
      ) : displayGroups.length === 0 ? (
        <EmptyState message={items.length === 0 ? '空空如也。点右上 + 新增第一条' : '无匹配条目'} />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 32 }}>
          {displayGroups.map((g) => (
            <FolderSection
              key={g.folderId ?? (g.folder === '' ? '__uncategorized__' : `orphan:${g.folder}`)}
              name={g.folder}
              count={g.items.length}
              collapsed={isCollapsed(tab, g.folder)}
              onToggle={() => toggleCollapsed(tab, g.folder)}
              onRename={g.folderId != null ? () => setFolderDialog({ mode: 'rename', id: g.folderId!, name: g.folder }) : undefined}
              onDelete={g.folderId != null ? () => setConfirmDeleteFolder({ id: g.folderId!, name: g.folder, count: g.items.length }) : undefined}
              dense={tab === 'bookmark'}
            >
              {g.items.map((it, i) => (
                tab === 'bookmark' ? (
                  <div
                    key={it.id}
                    {...rowDragProps(g, it, i)}
                    style={{
                      cursor: dragId === it.id ? 'grabbing' : 'grab',
                      opacity: dragId === it.id ? 0.45 : 1,
                      boxShadow: dropHint?.id === it.id
                        ? (dropHint.after ? '0 2px 0 var(--accent)' : '0 -2px 0 var(--accent)')
                        : undefined,
                      borderRadius: 8
                    }}
                  >
                    <BookmarkRow
                      item={it}
                      onToggleFav={(id, next) => update(id, { favorite: next }).catch((e: any) => toast.show(`收藏失败：${e?.message ?? 'unknown'}`))}
                      onEdit={openEdit}
                      onDelete={setConfirmDeleteId}
                      onTagClick={setTag}
                    />
                  </div>
                ) : (
                  <ItemCard
                    key={it.id}
                    item={it}
                    index={i}
                    onToggleFav={(id, next) => update(id, { favorite: next }).catch((e: any) => toast.show(`收藏失败：${e?.message ?? 'unknown'}`))}
                    onEdit={openEdit}
                    onDelete={setConfirmDeleteId}
                    onTagClick={setTag}
                  />
                )
              ))}
            </FolderSection>
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
        defaultType={tab}
        foldersByType={folders}
        onSubmit={submitItem}
      />

      <FolderDialog
        open={folderDialog !== null}
        onOpenChange={(v) => !v && setFolderDialog(null)}
        title={folderDialog?.mode === 'rename' ? '重命名文件夹' : '新建文件夹'}
        initial={folderDialog?.mode === 'rename' ? folderDialog.name : undefined}
        onSubmit={submitFolder}
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
        open={confirmDeleteFolder !== null}
        onOpenChange={(v) => !v && setConfirmDeleteFolder(null)}
        title="删除文件夹"
        description={`文件夹「${confirmDeleteFolder?.name ?? ''}」内的 ${confirmDeleteFolder?.count ?? 0} 个条目将移至未分类，条目本身不删除。`}
        confirmText="删除"
        destructive
        onConfirm={confirmDeleteFolderAction}
      />

      {/* import mode selection dialog */}
      <Dialog open={importMode !== null} onOpenChange={(v) => !v && setImportMode(null)}>
        <DialogTitle>导入 JSON</DialogTitle>
        <div style={{ fontSize: 'var(--fs-sm)', color: 'var(--ink-mid)', marginBottom: 24 }}>
          <p style={{ margin: '0 0 8px' }}><strong>merge</strong>：按 id 合并（既有则更新，缺失则新增）。安全。</p>
          <p style={{ margin: 0 }}><strong>replace</strong>：先清空所有条目再全部插入。不可撤销。</p>
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
          <button onClick={() => setImportMode(null)} style={{ color: 'var(--ink-mid)' }}>取消</button>
          <button onClick={() => runImport('replace')} style={{ color: 'var(--accent)', fontWeight: 500 }}>replace</button>
          <button onClick={() => runImport('merge')} className="btn-primary">merge</button>
        </div>
      </Dialog>
    </div>
  )
}
