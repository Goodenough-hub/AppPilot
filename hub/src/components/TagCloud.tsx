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
          aria-label={`清除标签 ${active}`}
          className="chip"
          style={{
            fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)',
            background: 'var(--accent-tint)', color: 'var(--accent)',
            border: '1px solid transparent',
            borderRadius: 999, padding: '3px 10px'
          }}
        >
          #{active} ✕
        </button>
      )}
      {!active && tags.slice(0, 12).map((t) => (
        <button
          key={t}
          onClick={() => onSelect(t)}
          aria-pressed={false}
          className="chip"
          style={{
            fontSize: 'var(--fs-xs)', fontFamily: 'var(--font-mono)',
            color: 'var(--ink-dim)', border: '1px solid var(--rule)',
            borderRadius: 999, padding: '3px 10px'
          }}
        >
          #{t}
        </button>
      ))}
    </div>
  )
}
