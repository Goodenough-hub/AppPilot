import { renderHook } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useHotkey } from './useHotkey'

describe('useHotkey', () => {
  it('按 "/" 触发回调', () => {
    const cb = vi.fn()
    renderHook(() => useHotkey(cb))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '/' }))
    expect(cb).toHaveBeenCalled()
  })

  it('按 Cmd+K / Ctrl+K 触发', () => {
    const cb = vi.fn()
    renderHook(() => useHotkey(cb))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))
    expect(cb).toHaveBeenCalledTimes(2)
  })

  it('在 INPUT 焦点内不触发（避免打字冲突）', () => {
    const cb = vi.fn()
    renderHook(() => useHotkey(cb))
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()
    input.dispatchEvent(new KeyboardEvent('keydown', { key: '/', bubbles: true }))
    expect(cb).not.toHaveBeenCalled()
    input.remove()
  })
})