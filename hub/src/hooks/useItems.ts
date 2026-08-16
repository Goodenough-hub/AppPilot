import { useCallback, useEffect, useState } from 'react'
import { itemsApi, type Item, type ItemInput, type ItemPatch } from '@/api/hub'

export function useItems() {
  const [items, setItems] = useState<Item[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await itemsApi.list()
      setItems(data)
    } catch (e) {
      setError(e as Error)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void reload() }, [reload])

  const create = useCallback(async (input: ItemInput) => {
    const created = await itemsApi.create(input)
    setItems((prev) => [created, ...prev])
    return created
  }, [])

  const update = useCallback(async (id: number, patch: ItemPatch) => {
    const prev = items
    setItems((cur) => cur.map((it) => it.id === id ? { ...it, ...patch } as Item : it))
    try {
      const updated = await itemsApi.update(id, patch)
      setItems((cur) => cur.map((it) => it.id === id ? updated : it))
      return updated
    } catch (e) {
      setItems(prev)
      throw e
    }
  }, [items])

  const remove = useCallback(async (id: number) => {
    const prev = items
    setItems((cur) => cur.filter((it) => it.id !== id))
    try {
      await itemsApi.remove(id)
    } catch (e) {
      setItems(prev)
      throw e
    }
  }, [items])

  return { items, loading, error, reload, create, update, remove }
}