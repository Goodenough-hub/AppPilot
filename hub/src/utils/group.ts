import type { Folder, Item } from '@/api/hub'

/** 一个文件夹分组。folder 为 '' 表示「未分类」；folderId 为 null 表示未在 hub_folders 登记（孤儿/未分类） */
export interface ItemGroup {
  folder: string
  folderId: number | null
  items: Item[]
}

/**
 * 把条目按 folder 分组。分组顺序：
 * 1. 已登记的文件夹（保持 folders 入参顺序，后端按创建时间升序给出；空文件夹也保留）
 * 2. 仅被条目引用的孤儿文件夹名（字典序）
 * 3. 未分类（''）固定最后（且无条目时不出现）
 * 组内条目保持传入顺序。
 */
export function groupByFolder(items: Item[], folders: Folder[]): ItemGroup[] {
  const byFolder = new Map<string, Item[]>()
  for (const it of items) {
    const arr = byFolder.get(it.folder)
    if (arr) arr.push(it)
    else byFolder.set(it.folder, [it])
  }

  const groups: ItemGroup[] = []
  const registered = new Set<string>()
  for (const f of folders) {
    registered.add(f.name)
    groups.push({ folder: f.name, folderId: f.id, items: byFolder.get(f.name) ?? [] })
  }

  const orphans = Array.from(byFolder.keys())
    .filter((k) => k !== '' && !registered.has(k))
    .sort((a, b) => a.localeCompare(b))
  for (const name of orphans) {
    groups.push({ folder: name, folderId: null, items: byFolder.get(name)! })
  }

  const uncategorized = byFolder.get('')
  if (uncategorized && uncategorized.length > 0) {
    groups.push({ folder: '', folderId: null, items: uncategorized })
  }
  return groups
}
