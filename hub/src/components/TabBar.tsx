import type { TabKey } from '@/utils/filter'

const TABS: { key: TabKey; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'bookmark', label: 'Bookmarks' },
  { key: 'prompt', label: 'Prompts' },
  { key: 'skill', label: 'Skills' },
  { key: 'starred', label: 'Starred' }
]

export function TabBar({ active, onChange }: { active: TabKey; onChange: (k: TabKey) => void }) {
  return (
    <nav style={{ padding: '16px 0', display: 'flex', gap: 24 }}>
      {TABS.map((t) => {
        const isActive = t.key === active
        return (
          <button
            key={t.key}
            onClick={() => onChange(t.key)}
            aria-current={isActive ? 'page' : undefined}
            style={{
              fontSize: 'var(--fs-md)',
              color: isActive ? 'var(--ink)' : 'var(--ink-mid)',
              borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
              padding: '4px 0'
            }}
          >
            {t.label}
          </button>
        )
      })}
    </nav>
  )
}
