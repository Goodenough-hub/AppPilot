import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, beforeEach } from 'vitest'
import { useTheme } from './useTheme'

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('默认无存储时设置 data-theme 属性', () => {
    renderHook(() => useTheme())
    expect(document.documentElement.getAttribute('data-theme')).toBeTruthy()
  })

  it('toggle 在 dark 与 light 之间切换并持久化', () => {
    const { result } = renderHook(() => useTheme())
    const initial = result.current.theme
    act(() => result.current.toggle())
    expect(result.current.theme).not.toBe(initial)
    expect(localStorage.getItem('hub_theme')).toBe(result.current.theme)
    expect(document.documentElement.getAttribute('data-theme')).toBe(result.current.theme)
  })

  it('setTheme 直接设置', () => {
    const { result } = renderHook(() => useTheme())
    act(() => result.current.setTheme('light'))
    expect(result.current.theme).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    expect(localStorage.getItem('hub_theme')).toBe('light')
  })
})