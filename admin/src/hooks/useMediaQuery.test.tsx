import { describe, it, expect, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useMediaQuery } from './useMediaQuery'

type Listener = () => void

function installMatchMedia(initial: boolean) {
  let listeners: Listener[] = []
  const mql = {
    matches: initial,
    media: '',
    onchange: null,
    addEventListener: (_: string, cb: Listener) => { listeners.push(cb) },
    removeEventListener: (_: string, cb: Listener) => { listeners = listeners.filter(l => l !== cb) },
    dispatchEvent: () => true,
    // 测试辅助：模拟视口变化
    _emit(next: boolean) { this.matches = next; listeners.forEach(l => l()) },
  }
  ;(window as unknown as { matchMedia: unknown }).matchMedia = () => mql
  return mql
}

describe('useMediaQuery', () => {
  const original = window.matchMedia

  afterEach(() => {
    ;(window as unknown as { matchMedia: unknown }).matchMedia = original
  })

  it('初始返回媒体查询的当前匹配值', () => {
    installMatchMedia(true)
    const { result } = renderHook(() => useMediaQuery('(max-width: 640px)'))
    expect(result.current).toBe(true)
  })

  it('视口变化时更新返回值', () => {
    const mql = installMatchMedia(false)
    const { result } = renderHook(() => useMediaQuery('(max-width: 640px)'))
    expect(result.current).toBe(false)
    act(() => { mql._emit(true) })
    expect(result.current).toBe(true)
  })

  it('matchMedia 不可用时返回 false', () => {
    ;(window as unknown as { matchMedia: unknown }).matchMedia = undefined
    const { result } = renderHook(() => useMediaQuery('(max-width: 640px)'))
    expect(result.current).toBe(false)
  })
})
