import * as RA from '@radix-ui/react-alert-dialog'

export function AlertDialog({
  open, onOpenChange, title, description, confirmText = '确认', cancelText = '取消', onConfirm, destructive = false
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  title: string
  description: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => void
  destructive?: boolean
}) {
  return (
    <RA.Root open={open} onOpenChange={onOpenChange}>
      <RA.Portal>
        <RA.Overlay
          className="fixed inset-0 bg-black/40 z-40"
          style={{ animation: 'overlay-in 160ms ease-out' }}
        />
        <RA.Content
          className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-full max-w-md rounded-xl p-6"
          style={{
            background: 'var(--paper-lift)',
            border: '1px solid var(--rule)',
            color: 'var(--ink)',
            boxShadow: '0 24px 64px -16px rgba(0, 0, 0, 0.5)',
            animation: 'dialog-in 200ms cubic-bezier(0.32, 0.72, 0.35, 1)'
          }}
        >
          <RA.Title className="font-serif italic" style={{ fontSize: 'var(--fs-lg)', margin: 0, marginBottom: 12 }}>{title}</RA.Title>
          <RA.Description className="text-mid" style={{ fontSize: 'var(--fs-sm)', marginBottom: 24 }}>{description}</RA.Description>
          <div className="flex justify-end gap-3">
            <RA.Cancel className="text-mid hover:text-accent" style={{ fontSize: 'var(--fs-sm)' }}>{cancelText}</RA.Cancel>
            <RA.Action
              onClick={onConfirm}
              className={destructive ? '' : 'btn-primary'}
              style={destructive ? { color: 'var(--accent)', fontWeight: 500 } : undefined}
            >
              {confirmText}
            </RA.Action>
          </div>
        </RA.Content>
      </RA.Portal>
    </RA.Root>
  )
}