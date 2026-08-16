import { useCallback, useEffect, useState } from 'react'
import { foldersApi, type Folder, type ItemType } from '@/api/hub'

const ALL_TYPES: ItemType[] = ['bookmark', 'prompt', 'skill']

const EMPTY: Record<ItemType, Folder[]> = { bookmark: [], prompt: [], skill: [] }

/**
 * 拉取全部三个类型的文件夹目录（一次并行三请求，之后不再随 tab 切换重拉）。
 * 文件夹的新建/重命名/删除与条目的写入（后端自动登记）都可能改变目录，由调用方触发 reload。
 */
export function useFolders() {
  const [folders, setFolders] = useState<Record<ItemType, Folder[]>>(EMPTY)

  const reload = useCallback(async () => {
    const [bookmark, prompt, skill] = await Promise.all(ALL_TYPES.map((t) => foldersApi.list(t)))
    setFolders({ bookmark, prompt, skill })
  }, [])

  useEffect(() => {
    reload().catch(() => { /* 列表失败不阻塞页面，保持空目录 */ })
  }, [reload])

  const create = useCallback(async (type: ItemType, name: string) => {
    const created = await foldersApi.create(type, name)
    await reload()
    return created
  }, [reload])

  const rename = useCallback(async (id: number, name: string) => {
    const updated = await foldersApi.rename(id, name)
    await reload()
    return updated
  }, [reload])

  const remove = useCallback(async (id: number) => {
    await foldersApi.remove(id)
    await reload()
  }, [reload])

  return { folders, reload, create, rename, remove }
}
