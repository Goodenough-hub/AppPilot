import { useCallback, useState } from 'react'
import type { ItemType } from '@/api/hub'

const STORAGE_KEY = 'hub_folder_collapsed'

/** 折叠状态的存储 key。命名空间按类型隔离，用 JSON 数组避免名字里含分隔符造成歧义 */
export function collapseKey(type: ItemType, folder: string): string {
  return JSON.stringify([type, folder])
}

function load(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed: unknown = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? (parsed as Record<string, boolean>) : {}
  } catch {
    return {}
  }
}

/** 文件夹收起/展开状态，持久化到 localStorage（缺省 = 展开） */
export function useCollapsedFolders() {
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(load)

  const toggle = useCallback((type: ItemType, folder: string) => {
    setCollapsed((prev) => {
      const key = collapseKey(type, folder)
      const next = { ...prev, [key]: !prev[key] }
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
      } catch { /* 存储不可用时退化为会话内状态 */ }
      return next
    })
  }, [])

  const isCollapsed = useCallback(
    (type: ItemType, folder: string) => collapsed[collapseKey(type, folder)] === true,
    [collapsed]
  )

  return { isCollapsed, toggle }
}
