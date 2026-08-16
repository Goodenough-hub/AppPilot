import { useEffect } from 'react'

/**
 * 全局搜索快捷键：按 "/" 或 Cmd+K / Ctrl+K 触发。
 * 在输入框焦点内不触发（避免打字冲突）。
 */
export function useHotkey(onTrigger: () => void) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (document.activeElement?.tagName ?? '').toLowerCase()
      const inInput = tag === 'input' || tag === 'textarea' || (document.activeElement as HTMLElement)?.isContentEditable
      const isSlash = e.key === '/' && !e.metaKey && !e.ctrlKey && !e.altKey && !inInput
      const isCmdK = e.key.toLowerCase() === 'k' && (e.metaKey || e.ctrlKey)
      if (isSlash || isCmdK) {
        e.preventDefault()
        onTrigger()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onTrigger])
}