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

  // 拖拽排序/移动：dragId 用 ref（dragstart→drop 同一帧内读写，避开 state 批处理竞态）；
  // dropHint 仅用于视觉指示线，用 state 即可
  const dragIdRef = useRef<number | null>(null)
  const [dragId, setDragId] = useState<number | null>(null) // 仅驱动「被拖条目半透明/抓手光标」视觉
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

  /** 落点处理：同文件夹=组内重排；跨文件夹=改 folder + 重排目标组原有条目（被拖条目落到组尾）。书签行/卡片共用。 */
  const dropOnItem = async (group: { folder: string; items: Item[] }, targetIndex: number, after: boolean) => {
    const dragId = dragIdRef.current
    if (dragId == null) return
    const from = group.items.findIndex((x) => x.id === dragId)
    let to = targetIndex + (after ? 1 : 0)
    if (from >= 0 && from < to) to -= 1
    if (from === to) return
    try {
      if (from >= 0) {
        // 同组重排：moveItem 的 to 是「移除 from 之后」的下标，直接给即可
        await reorder(tab, group.folder, moveItem(group.items.map((x) => x.id), from, to))
      } else {
        // 跨组移动：先 await 改 folder（PATCH 落库后被拖条目才属目标组），再 reorder
        // 含被拖条目按落点插入。串行保证 reorder 作用域校验（ids 须全属目标组）通过。
        const ids = group.items.map((x) => x.id)
        ids.splice(Math.max(0, Math.min(to, ids.length)), 0, dragId)
        await update(dragId, { folder: group.folder })
        await reorder(tab, group.folder, ids)
        toast.show(`已移动到「${group.folder || '未分类'}」`)
      }
    } catch (e: any) {
      toast.show(`移动失败：${e?.message ?? 'unknown'}`)
    }
  }

  /** 组级落点（拖到文件夹头/空白/空态区，未精确命中某条目卡）：移动到该文件夹组尾 */
  const dropOnGroup = async (group: { folder: string; items: Item[] }) => {
    const dragId = dragIdRef.current
    if (dragId == null) return
    const already = group.items.some((x) => x.id === dragId)
    try {
      if (already) {
        // 同组拖到空白：移到组尾
        const ids = group.items.map((x) => x.id)
        const from = ids.indexOf(dragId)
        await reorder(tab, group.folder, moveItem(ids, from, ids.length - 1))
      } else {
        // 跨组拖到空白：改 folder（落组尾）+ 重排目标组（含被拖条目在尾部）
        await update(dragId, { folder: group.folder })
        const ids = [...group.items.map((x) => x.id), dragId]
        await reorder(tab, group.folder, ids)
        toast.show(`已移动到「${group.folder || '未分类'}」`)
      }
    } catch (e: any) {
      toast.show(`移动失败：${e?.message ?? 'unknown'}`)
    }
  }

  /** 拖拽状态重置（dragend/drop 后统一清） */
  const resetDrag = () => { dragIdRef.current = null; setDragId(null); setDropHint(null) }

  /**
   * 计算落点 before/after。书签行（纵向列表）只看 y；prompt/skill 卡片是二维
   * grid（一行多张、换行），需按行主序压平成一维：先比所在行（y 中心），同行
   * 再比列（x 中心），与数组顺序严格一致，避免「放不对位置」。
   */
  const computeAfter = (e: React.DragEvent, halfAxis: 'y' | 'x'): boolean => {
    const rect = e.currentTarget.getBoundingClientRect()
    if (halfAxis === 'y') return e.clientY > rect.top + rect.height / 2
    // 卡片：二维行主序。行的容差取卡高的一半（行间距内也算同一行）
    const rowDelta = e.clientY - (rect.top + rect.height / 2)
    if (Math.abs(rowDelta) > rect.height / 2) return rowDelta > 0 // 跨行：下面的行=after
    return e.clientX > rect.left + rect.width / 2 // 同行：比 x
  }

  /** 条目本体（行/卡片）的拖拽 props；after 的判定轴由 halfAxis 决定 */
  const dragProps = (group: { folder: string; items: Item[] }, it: Item, i: number, halfAxis: 'y' | 'x') => ({
    draggable: true,
    onDragStart: (e: React.DragEvent) => {
      e.dataTransfer.effectAllowed = 'move'
      e.dataTransfer.setData('text/plain', String(it.id))
      dragIdRef.current = it.id
      setDragId(it.id) // 仅驱动视觉（被拖条目半透明）
    },
    onDragOver: (e: React.DragEvent) => {
      if (dragIdRef.current == null) return
      e.preventDefault()
      setDropHint({ id: it.id, after: computeAfter(e, halfAxis) })
    },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation() // 条目卡上的 drop 优先，不冒泡到 FolderSection 的组级落点
      // after 直接从 drop 事件自身坐标算；dragId 读 ref，均不依赖可能因批处理而过期的 state
      void dropOnItem(group, i, computeAfter(e, halfAxis))
      resetDrag()
    },
    onDragEnd: resetDrag
  })

  /** 组级落点 props：整个组区块（文件夹头/空白/空态区）接收拖入，移到该文件夹 */
  const emptyDropProps = (group: { folder: string; items: Item[] }) => ({
    onDragOver: (e: React.DragEvent) => { if (dragIdRef.current != null) e.preventDefault() },
    onDrop: (e: React.DragEvent) => { e.preventDefault(); void dropOnGroup(group); resetDrag() }
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
              emptyDrop={emptyDropProps(g)}
            >
              {g.items.map((it, i) => (
                <div
                  key={it.id}
                  {...dragProps(g, it, i, tab === 'bookmark' ? 'y' : 'x')}
                  style={{
                    cursor: dragId === it.id ? 'grabbing' : 'grab',
                    opacity: dragId === it.id ? 0.45 : 1,
                    boxShadow: dropHint?.id === it.id
                      ? tab === 'bookmark'
                        ? (dropHint.after ? '0 2px 0 var(--accent)' : '0 -2px 0 var(--accent)')
                        : (dropHint.after ? '2px 0 0 var(--accent)' : '-2px 0 0 var(--accent)')
                      : undefined,
                    borderRadius: tab === 'bookmark' ? 8 : 12
                  }}
                >
                  {tab === 'bookmark' ? (
                    <BookmarkRow
                      item={it}
                      onToggleFav={(id, next) => update(id, { favorite: next }).catch((e: any) => toast.show(`收藏失败：${e?.message ?? 'unknown'}`))}
                      onEdit={openEdit}
                      onDelete={setConfirmDeleteId}
                      onTagClick={setTag}
                    />
                  ) : (
                    <ItemCard
                      item={it}
                      index={i}
                      onToggleFav={(id, next) => update(id, { favorite: next }).catch((e: any) => toast.show(`收藏失败：${e?.message ?? 'unknown'}`))}
                      onEdit={openEdit}
                      onDelete={setConfirmDeleteId}
                      onTagClick={setTag}
                    />
                  )}
                </div>
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
