import { useEffect, useRef } from 'react'
import { Search } from 'lucide-react'
import { useHotkey } from '@/hooks/useHotkey'

export function SearchBar({
  value, onChange
}: { value: string; onChange: (v: string) => void }) {
  const ref = useRef<HTMLInputElement>(null)

  useHotkey(() => {
    ref.current?.focus()
    ref.current?.select()
  })

  useEffect(() => {
    // Esc clears value and blurs
    const h = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && document.activeElement === ref.current) {
        onChange('')
        ref.current?.blur()
      }
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [onChange])

  return (
    <div style={{
      position: 'fixed', bottom: 24, left: 0, right: 0,
      display: 'grid', placeItems: 'center', pointerEvents: 'none', zIndex: 20
    }}>
      <div className="search-pill" style={{
        pointerEvents: 'auto',
        display: 'flex', alignItems: 'center', gap: 8,
        background: 'var(--paper-lift)', border: '1px solid var(--rule)',
        borderRadius: 999, padding: '10px 16px', width: 'min(440px, calc(100vw - 48px))',
        boxShadow: '0 8px 24px rgba(0,0,0,0.25)'
      }}>
        <Search size={14} color="var(--ink-mid)" />
        <input
          ref={ref}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="搜索  ·  按 / 或 ⌘K 唤起"
          aria-label="搜索"
          style={{ background: 'transparent', border: 'none', padding: 0, flex: 1, fontSize: 'var(--fs-sm)', minWidth: 0 }}
        />
        {value && (
          <button aria-label="清空" onClick={() => onChange('')} style={{ color: 'var(--ink-dim)', fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)' }}>✕</button>
        )}
      </div>
    </div>
  )
}
