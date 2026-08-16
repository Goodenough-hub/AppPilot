import type { TabKey } from '@/utils/filter'

const TABS: { key: TabKey; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'bookmark', label: 'Bookmarks' },
  { key: 'prompt', label: 'Prompts' },
  { key: 'skill', label: 'Skills' },
  { key: 'starred', label: 'Starred' }
]

export function TabBar({
  active, onChange, counts
}: {
  active: TabKey
  onChange: (k: TabKey) => void
  counts?: Partial<Record<TabKey, number>>
}) {
  return (
    <nav style={{ padding: '16px 0', display: 'flex', gap: 24, overflowX: 'auto' }}>
      {TABS.map((t) => {
        const isActive = t.key === active
        const n = counts?.[t.key]
        return (
          <button
            key={t.key}
            onClick={() => onChange(t.key)}
            aria-current={isActive ? 'page' : undefined}
            className="tab-btn"
            style={{
              fontSize: 'var(--fs-md)',
              color: isActive ? 'var(--ink)' : 'var(--ink-mid)',
              borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
              padding: '4px 0',
              whiteSpace: 'nowrap'
            }}
          >
            {t.label}
            {n != null && (
              <sup style={{ marginLeft: 6, fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: isActive ? 'var(--accent)' : 'var(--ink-dim)' }}>
                {n}
              </sup>
            )}
          </button>
        )
      })}
    </nav>
  )
}
