import * as RM from '@radix-ui/react-dropdown-menu'
import type { ReactNode } from 'react'

export const DropdownRoot = RM.Root
export const DropdownTrigger = RM.Trigger

export function DropdownContent({ children }: { children: ReactNode }) {
  return (
    <RM.Portal>
      <RM.Content
        className="z-50 min-w-[160px] rounded p-1 shadow-md"
        style={{ background: 'var(--paper-lift)', border: '1px solid var(--rule)', color: 'var(--ink)' }}
        sideOffset={4}
      >
        {children}
      </RM.Content>
    </RM.Portal>
  )
}

export function DropdownItem({ children, onSelect }: { children: ReactNode; onSelect?: () => void }) {
  return (
    <RM.Item
      onSelect={onSelect}
      className="cursor-pointer rounded px-3 py-2 outline-none data-[highlighted]:bg-[var(--paper-sink)]"
      style={{ fontSize: 'var(--fs-sm)' }}
    >
      {children}
    </RM.Item>
  )
}