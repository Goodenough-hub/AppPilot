import * as RT from '@radix-ui/react-toast'
import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'

type ToastState = { open: boolean; msg: string }
const Ctx = createContext<{ show: (msg: string) => void }>({ show: () => {} })

export function ToastProvider({ children }: { children: ReactNode }) {
  const [t, setT] = useState<ToastState>({ open: false, msg: '' })
  // 打开 → 立即关 → 再打开，触发 Radix 重启动动效
  const trigger = useCallback((msg: string) => {
    setT({ open: false, msg })
    setTimeout(() => setT({ open: true, msg }), 10)
  }, [])
  return (
    <Ctx.Provider value={{ show: trigger }}>
      <RT.Provider swipeDirection="right" duration={2000}>
        {children}
        <RT.Root open={t.open} onOpenChange={(v) => setT((s) => ({ ...s, open: v }))}
          className="fixed bottom-6 right-6 z-[100] rounded px-4 py-3 shadow-md"
          style={{ background: 'var(--paper-lift)', border: '1px solid var(--rule)', color: 'var(--ink)', fontSize: 'var(--fs-sm)' }}
        >
          <RT.Title>{t.msg}</RT.Title>
        </RT.Root>
        <RT.Viewport />
      </RT.Provider>
    </Ctx.Provider>
  )
}

export function useToast() { return useContext(Ctx) }