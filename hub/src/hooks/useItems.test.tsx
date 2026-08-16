import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useItems } from './useItems'
import { itemsApi, type Item } from '@/api/hub'

const sample: Item = {
  id: 1, type: 'bookmark', title: 'X', url: null, content: null,
  tags: [], favorite: false, folder: '', icon: '', createdAt: '', updatedAt: ''
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
})