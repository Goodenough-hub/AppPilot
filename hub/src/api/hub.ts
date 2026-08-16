import { apiClient } from './client'

export type ItemType = 'bookmark' | 'prompt' | 'skill'

export interface Item {
  id: number
  type: ItemType
  title: string
  url: string | null
  content: string | null
  tags: string[]
  favorite: boolean
  /** 文件夹名；'' 表示未分类。命名空间随 item.type */
  folder: string
  /** 自定义图标 URL；'' 表示按站点 favicon 自动探测 */
  icon: string
  createdAt: string
  updatedAt: string
}

export interface ItemInput {
  type: ItemType
  title: string
  url?: string | null
  content?: string | null
  tags?: string[]
  favorite?: boolean
  folder?: string
  icon?: string
}

export interface ItemPatch {
  type?: ItemType
  title?: string
  url?: string | null
  content?: string | null
  tags?: string[]
  favorite?: boolean
  folder?: string
  icon?: string
}

export const itemsApi = {
  list: async (): Promise<Item[]> => (await apiClient.get<Item[]>('/hub/items')).data,
  create: async (input: ItemInput): Promise<Item> => (await apiClient.post<Item>('/hub/items', input)).data,
  update: async (id: number, patch: ItemPatch): Promise<Item> =>
    (await apiClient.patch<Item>(`/hub/items/${id}`, patch)).data,
  remove: async (id: number): Promise<void> => { await apiClient.delete(`/hub/items/${id}`) },
  exportJson: async (): Promise<Item[]> => (await apiClient.get<Item[]>('/hub/export')).data,
  importJson: async (items: Item[], mode: 'merge' | 'replace' = 'merge'): Promise<{ affected: number; mode: string }> =>
    (await apiClient.post(`/hub/import`, items, { params: { mode } })).data
}

/** 文件夹（按类型隔离命名空间；同名在不同 type 下互不相干） */
export interface Folder {
  id: number
  type: ItemType
  name: string
  itemCount: number
  createdAt: string
}

export const foldersApi = {
  list: async (type: ItemType): Promise<Folder[]> =>
    (await apiClient.get<Folder[]>('/hub/folders', { params: { type } })).data,
  create: async (type: ItemType, name: string): Promise<Folder> =>
    (await apiClient.post<Folder>('/hub/folders', { type, name })).data,
  rename: async (id: number, name: string): Promise<Folder> =>
    (await apiClient.patch<Folder>(`/hub/folders/${id}`, { name })).data,
  remove: async (id: number): Promise<void> => { await apiClient.delete(`/hub/folders/${id}`) }
}