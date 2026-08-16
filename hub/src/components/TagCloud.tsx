export function TagCloud({
  tags, active, onSelect
}: { tags: string[]; active: string | null; onSelect: (t: string | null) => void }) {
  if (tags.length === 0) return null
  return (
    <div style={{ padding: '12px 0', display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
      <span style={{ fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)', color: 'var(--ink-dim)', marginRight: 4 }}>filter:</span>
      {active && (
        <button
          onClick={() => onSelect(null)}
          style={{
            fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)',
            background: 'var(--accent-tint)', color: 'var(--accent)',
            borderRadius: 4, padding: '2px 8px'
          }}
        >
          #{active} ✕
        </button>
      )}
      {!active && tags.slice(0, 12).map((t) => (
        <button
          key={t}
          onClick={() => onSelect(t)}
          style={{
            fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)',
            color: 'var(--ink-dim)', border: '1px solid var(--rule)',
            borderRadius: 4, padding: '2px 8px'
          }}
        >
          #{t}
        </button>
      ))}
    </div>
  )
}
