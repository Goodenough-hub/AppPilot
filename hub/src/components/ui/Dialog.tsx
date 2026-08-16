import * as RD from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'

export function Dialog({ open, onOpenChange, children }: { open: boolean; onOpenChange: (v: boolean) => void; children: ReactNode }) {
  return (
    <RD.Root open={open} onOpenChange={onOpenChange}>
      <RD.Portal>
        <RD.Overlay className="fixed inset-0 bg-black/40 z-40" />
        <RD.Content
          className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-full max-w-lg rounded-lg p-6 shadow-lg"
          style={{ background: 'var(--paper-lift)', border: '1px solid var(--rule)', color: 'var(--ink)' }}
        >
          {children}
        </RD.Content>
      </RD.Portal>
    </RD.Root>
  )
}

export function DialogTitle({ children }: { children: ReactNode }) {
  return (
    <RD.Title asChild>
      <h2 className="font-serif italic" style={{ fontSize: 'var(--fs-lg)', margin: 0, marginBottom: 16 }}>
        {children}
      </h2>
    </RD.Title>
  )
}

export function DialogClose() {
  return (
    <RD.Close asChild>
      <button aria-label="关闭" className="absolute top-4 right-4 text-mid hover:text-accent">
        <X size={18} />
      </button>
    </RD.Close>
  )
}