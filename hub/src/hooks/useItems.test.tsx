import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useItems } from './useItems'
import { itemsApi, type Item } from '@/api/hub'

const sample: Item = {
  id: 1, type: 'bookmark', title: 'X', url: null, content: null,
  tags: [], favorite: false, folder: '', icon: '', position: 0, createdAt: '', updatedAt: ''
}

describe('useItems', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('初始加载调 itemsApi.list', async () => {
    vi.spyOn(itemsApi, 'list').mockResolvedValue([sample])
    const { result } = renderHook(() => useItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.items).toEqual([sample])
  })

  it('create 后乐观加入列表', async () => {
    vi.spyOn(itemsApi, 'list').mockResolvedValue([])
    const created = { ...sample, id: 2, title: 'new' }
    vi.spyOn(itemsApi, 'create').mockResolvedValue(created)
    const { result } = renderHook(() => useItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(async () => { await result.current.create({ type: 'bookmark', title: 'new' }) })
    expect(result.current.items).toHaveLength(1)
    expect(result.current.items[0].title).toBe('new')
  })

  it('update 后乐观替换', async () => {
    vi.spyOn(itemsApi, 'list').mockResolvedValue([sample])
    const patched = { ...sample, favorite: true }
    vi.spyOn(itemsApi, 'update').mockResolvedValue(patched)
    const { result } = renderHook(() => useItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(async () => { await result.current.update(1, { favorite: true }) })
    expect(result.current.items[0].favorite).toBe(true)
  })

  it('remove 后乐观移除', async () => {
    vi.spyOn(itemsApi, 'list').mockResolvedValue([sample])
    vi.spyOn(itemsApi, 'remove').mockResolvedValue()
    const { result } = renderHook(() => useItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(async () => { await result.current.remove(1) })
    expect(result.current.items).toEqual([])
  })

  it('remove 失败回滚', async () => {
    vi.spyOn(itemsApi, 'list').mockResolvedValue([sample])
    vi.spyOn(itemsApi, 'remove').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    await expect(
      act(async () => { await result.current.remove(1) })
    ).rejects.toThrow('boom')
    expect(result.current.items).toEqual([sample])
  })

  it('reorder 乐观重排本地顺序并调 api 持久化', async () => {
    const a = { ...sample, id: 1, title: 'A' }
    const b = { ...sample, id: 2, title: 'B' }
    const c = { ...sample, id: 3, title: 'C' }
    vi.spyOn(itemsApi, 'list').mockResolvedValue([a, b, c])
    const spy = vi.spyOn(itemsApi, 'reorder').mockResolvedValue()
    const { result } = renderHook(() => useItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(async () => { await result.current.reorder('bookmark', 'F', [3, 1, 2]) })
    expect(spy).toHaveBeenCalledWith('bookmark', 'F', [3, 1, 2])
    // 组内相对顺序已按新序排好（重排条目被移到末尾段，内部顺序 [C, A, B]）
    const reordered = result.current.items.filter((i) => [3, 1, 2].includes(i.id))
    expect(reordered.map((i) => i.title)).toEqual(['C', 'A', 'B'])
  })

  it('reorder 失败回滚原有顺序', async () => {
    const a = { ...sample, id: 1, title: 'A' }
    const b = { ...sample, id: 2, title: 'B' }
    vi.spyOn(itemsApi, 'list').mockResolvedValue([a, b])
    vi.spyOn(itemsApi, 'reorder').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    await expect(
      act(async () => { await result.current.reorder('bookmark', 'F', [2, 1]) })
    ).rejects.toThrow('boom')
    expect(result.current.items.map((i) => i.id)).toEqual([1, 2])
  })
})