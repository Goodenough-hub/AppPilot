import type { Item, ItemType } from '@/api/hub'

export type TabKey = ItemType

export interface FilterInput {
  tab: TabKey
  tag?: string | null
  query?: string | null
}

export function filterItems(items: Item[], f: FilterInput): Item[] {
  const q = (f.query ?? '').toLowerCase().trim()
  const tag = (f.tag ?? '').toLowerCase().trim()
  return items.filter((it) => {
    if (it.type !== f.tab) return false
    if (tag && !it.tags.some((t) => t.toLowerCase() === tag)) return false
    if (q) {
      const hay = [
        it.title,
        it.content ?? '',
        it.url ?? '',
        it.tags.join(' ')
      ].join(' ').toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })
}