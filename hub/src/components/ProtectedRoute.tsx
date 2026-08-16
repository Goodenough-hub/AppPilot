import { Navigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth, canAccessHub } from '@/contexts/AuthContext'

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { token, role, appScope } = useAuth()
  if (!token) return <Navigate to="/login" replace />
  if (!canAccessHub(appScope, role)) return <Navigate to="/login" replace />
  return <>{children}</>
}
