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
}

export interface ItemPatch {
  type?: ItemType
  title?: string
  url?: string | null
  content?: string | null
  tags?: string[]
  favorite?: boolean
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