import { describe, it, expect, beforeEach, vi } from 'vitest'
import { itemsApi } from './hub'
import { apiClient } from './client'

describe('itemsApi', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
    localStorage.setItem('hub_token', 'test-token')
  })

  it('list 请求 GET /hub/items', async () => {
    const spy = vi.spyOn(apiClient, 'get').mockResolvedValue({ data: [] })
    await itemsApi.list()
    expect(spy).toHaveBeenCalledWith('/hub/items')
  })

  it('create 请求 POST /hub/items', async () => {
    const spy = vi.spyOn(apiClient, 'post').mockResolvedValue({ data: { id: 1 } })
    await itemsApi.create({ type: 'bookmark', title: 'x', tags: [] })
    expect(spy).toHaveBeenCalledWith('/hub/items', expect.objectContaining({ type: 'bookmark' }))
  })

  it('update 请求 PATCH /hub/items/:id', async () => {
    const spy = vi.spyOn(apiClient, 'patch').mockResolvedValue({ data: { id: 1 } })
    await itemsApi.update(1, { favorite: true })
    expect(spy).toHaveBeenCalledWith('/hub/items/1', { favorite: true })
  })

  it('remove 请求 DELETE /hub/items/:id', async () => {
    const spy = vi.spyOn(apiClient, 'delete').mockResolvedValue({ data: null })
    await itemsApi.remove(2)
    expect(spy).toHaveBeenCalledWith('/hub/items/2')
  })
})