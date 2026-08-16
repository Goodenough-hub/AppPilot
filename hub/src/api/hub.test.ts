import { describe, it, expect, beforeEach, vi } from 'vitest'
import { itemsApi, foldersApi } from './hub'
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

  it('reorder 请求 POST /hub/items/order', async () => {
    const spy = vi.spyOn(apiClient, 'post').mockResolvedValue({ data: null })
    await itemsApi.reorder('bookmark', 'Infini-AI', [3, 1, 2])
    expect(spy).toHaveBeenCalledWith('/hub/items/order', { type: 'bookmark', folder: 'Infini-AI', ids: [3, 1, 2] })
  })
})

describe('foldersApi', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
    localStorage.setItem('hub_token', 'test-token')
  })

  it('list 请求 GET /hub/folders 带 type 参数', async () => {
    const spy = vi.spyOn(apiClient, 'get').mockResolvedValue({ data: [] })
    await foldersApi.list('bookmark')
    expect(spy).toHaveBeenCalledWith('/hub/folders', { params: { type: 'bookmark' } })
  })

  it('create 请求 POST /hub/folders', async () => {
    const spy = vi.spyOn(apiClient, 'post').mockResolvedValue({ data: { id: 1 } })
    await foldersApi.create('bookmark', 'Infini-AI')
    expect(spy).toHaveBeenCalledWith('/hub/folders', { type: 'bookmark', name: 'Infini-AI' })
  })

  it('rename 请求 PATCH /hub/folders/:id', async () => {
    const spy = vi.spyOn(apiClient, 'patch').mockResolvedValue({ data: { id: 1 } })
    await foldersApi.rename(1, '芯穹')
    expect(spy).toHaveBeenCalledWith('/hub/folders/1', { name: '芯穹' })
  })

  it('remove 请求 DELETE /hub/folders/:id', async () => {
    const spy = vi.spyOn(apiClient, 'delete').mockResolvedValue({ data: null })
    await foldersApi.remove(2)
    expect(spy).toHaveBeenCalledWith('/hub/folders/2')
  })
})