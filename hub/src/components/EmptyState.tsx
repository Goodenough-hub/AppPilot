export function EmptyState({ message = '暂无条目' }: { message?: string }) {
  return (
    <div style={{
      padding: '80px 0', textAlign: 'center',
      color: 'var(--ink-dim)', fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-sm)'
    }}>
      {message}
    </div>
  )
}
