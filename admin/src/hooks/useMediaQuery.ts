import { useEffect, useState } from 'react'

/**
 * 订阅一个 CSS 媒体查询，返回当前是否匹配。
 * 用于需要按视口宽度切换 DOM 结构的场景（如仪表盘手机端单列只读）。
 * matchMedia 不可用时（如 SSR / 测试环境未 mock）返回 false。
 */
export function useMediaQuery(query: string): boolean {
  const getMatch = () =>
    typeof window !== 'undefined' && typeof window.matchMedia === 'function'
      ? window.matchMedia(query).matches
      : false

  const [matches, setMatches] = useState(getMatch)

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return
    const mql = window.matchMedia(query)
    const handler = () => setMatches(mql.matches)
    handler()
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [query])

  return matches
}
