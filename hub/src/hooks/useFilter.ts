import { useMemo, useState } from 'react'
import { filterItems, type TabKey } from '@/utils/filter'
import type { Item } from '@/api/hub'

export function useFilter(items: Item[]) {
  const [tab, setTab] = useState<TabKey>('all')
  const [tag, setTag] = useState<string | null>(null)
  const [query, setQuery] = useState('')

  const filtered = useMemo(() => filterItems(items, { tab, tag, query }), [items, tab, tag, query])

  const counts = useMemo(() => ({
    all: items.length,
    starred: items.filter(i => i.favorite).length,
    bookmark: items.filter(i => i.type === 'bookmark').length,
    prompt: items.filter(i => i.type === 'prompt').length,
    skill: items.filter(i => i.type === 'skill').length,
  }), [items])

  const allTags = useMemo(() => {
    const s = new Set<string>()
    for (const it of items) for (const t of it.tags) s.add(t)
    return Array.from(s).sort()
  }, [items])

  return { tab, setTab, tag, setTag, query, setQuery, filtered, counts, allTags }
}