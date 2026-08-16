import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useFolders } from './useFolders'
import { foldersApi, type Folder } from '@/api/hub'

const bm: Folder = { id: 1, type: 'bookmark', name: 'Infini-AI', itemCount: 2, createdAt: '' }

describe('useFolders', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('初始并行拉取三个类型的目录', async () => {
    const spy = vi.spyOn(foldersApi, 'list').mockImplementation(async (t) => (t === 'bookmark' ? [bm] : []))
    const { result } = renderHook(() => useFolders())
    await waitFor(() => expect(result.current.folders.bookmark).toEqual([bm]))
    expect(spy).toHaveBeenCalledTimes(3)
    expect(spy).toHaveBeenCalledWith('bookmark')
    expect(spy).toHaveBeenCalledWith('prompt')
    expect(spy).toHaveBeenCalledWith('skill')
    expect(result.current.folders.prompt).toEqual([])
  })

  it('create 调 api 后刷新目录', async () => {
    const listSpy = vi.spyOn(foldersApi, 'list').mockResolvedValue([])
    const created = { ...bm, id: 2, name: 'new' }
    const createSpy = vi.spyOn(foldersApi, 'create').mockResolvedValue(created)
    const { result } = renderHook(() => useFolders())
    await waitFor(() => expect(listSpy).toHaveBeenCalledTimes(3))
    await act(async () => { await result.current.create('bookmark', 'new') })
    expect(createSpy).toHaveBeenCalledWith('bookmark', 'new')
    expect(listSpy).toHaveBeenCalledTimes(6) // 初始 3 + reload 3
  })

  it('rename/remove 调对应 api 后刷新目录', async () => {
    const listSpy = vi.spyOn(foldersApi, 'list').mockResolvedValue([])
    const renamed = { ...bm, name: '芯穹' }
    const renameSpy = vi.spyOn(foldersApi, 'rename').mockResolvedValue(renamed)
    const removeSpy = vi.spyOn(foldersApi, 'remove').mockResolvedValue()
    const { result } = renderHook(() => useFolders())
    await waitFor(() => expect(listSpy).toHaveBeenCalledTimes(3))
    await act(async () => { await result.current.rename(1, '芯穹') })
    expect(renameSpy).toHaveBeenCalledWith(1, '芯穹')
    await act(async () => { await result.current.remove(1) })
    expect(removeSpy).toHaveBeenCalledWith(1)
    expect(listSpy).toHaveBeenCalledTimes(9) // 初始 3 + rename reload 3 + remove reload 3
  })

  it('list 失败时保持空目录、不抛错', async () => {
    vi.spyOn(foldersApi, 'list').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useFolders())
    await act(async () => { await Promise.resolve() })
    expect(result.current.folders.bookmark).toEqual([])
    expect(result.current.folders.prompt).toEqual([])
  })
})
