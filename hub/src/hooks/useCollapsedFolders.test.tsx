import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, beforeEach } from 'vitest'
import { useCollapsedFolders, collapseKey } from './useCollapsedFolders'

describe('useCollapsedFolders', () => {
  beforeEach(() => localStorage.clear())

  it('默认展开', () => {
    const { result } = renderHook(() => useCollapsedFolders())
    expect(result.current.isCollapsed('bookmark', 'Infini-AI')).toBe(false)
  })

  it('toggle 后收起并写入 localStorage，命名空间按类型隔离', () => {
    const { result } = renderHook(() => useCollapsedFolders())
    act(() => result.current.toggle('bookmark', 'Infini-AI'))
    expect(result.current.isCollapsed('bookmark', 'Infini-AI')).toBe(true)
    // 同名文件夹在另一类型下不受影响
    expect(result.current.isCollapsed('prompt', 'Infini-AI')).toBe(false)
    const stored = JSON.parse(localStorage.getItem('hub_folder_collapsed')!)
    expect(stored[collapseKey('bookmark', 'Infini-AI')]).toBe(true)
  })

  it('再次 toggle 恢复展开', () => {
    const { result } = renderHook(() => useCollapsedFolders())
    act(() => result.current.toggle('bookmark', 'A'))
    act(() => result.current.toggle('bookmark', 'A'))
    expect(result.current.isCollapsed('bookmark', 'A')).toBe(false)
  })

  it('从 localStorage 恢复既有状态', () => {
    localStorage.setItem('hub_folder_collapsed', JSON.stringify({ [collapseKey('skill', 'S')]: true }))
    const { result } = renderHook(() => useCollapsedFolders())
    expect(result.current.isCollapsed('skill', 'S')).toBe(true)
  })

  it('localStorage 内容损坏时回退为全展开', () => {
    localStorage.setItem('hub_folder_collapsed', '{bad json')
    const { result } = renderHook(() => useCollapsedFolders())
    expect(result.current.isCollapsed('bookmark', 'A')).toBe(false)
  })
})
