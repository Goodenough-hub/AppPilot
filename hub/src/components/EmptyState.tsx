import { Inbox } from 'lucide-react'

export function EmptyState({ message = '暂无条目' }: { message?: string }) {
  return (
    <div style={{
      padding: '80px 0', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12,
      color: 'var(--ink-dim)'
    }}>
      <Inbox size={28} strokeWidth={1.5} />
      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--fs-sm)' }}>{message}</span>
    </div>
  )
}
