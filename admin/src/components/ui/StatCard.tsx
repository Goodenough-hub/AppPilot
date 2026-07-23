import type { ReactNode } from 'react'

interface StatCardProps {
  label: string
  value: ReactNode
  /** CSS gradient for the accent dot + (optional) gradient number */
  gradient?: string
  /** glow color for the dot */
  glow?: string
  /** render the value with the gradient clipped into the text */
  gradientValue?: boolean
  className?: string
}

/**
 * A single stat tile used across Dashboard and Users.
 * Replaces the repeated inline-styled `glass-panel` stat blocks.
 */
export default function StatCard({
  label, value, gradient, glow, gradientValue, className = ''
}: StatCardProps) {
  return (
    <div className={`glass-panel stat-card ${className}`}>
      <div className="stat-card-head">
        {gradient && (
          <span
            className="stat-card-dot"
            style={{ background: gradient, boxShadow: glow ? `0 0 10px ${glow}` : undefined }}
          />
        )}
        <span className="stat-card-label">{label}</span>
      </div>
      <div
        className={`stat-card-value${gradientValue && gradient ? ' gradient' : ''}`}
        style={gradientValue && gradient ? { backgroundImage: gradient } : undefined}
      >
        {value}
      </div>
    </div>
  )
}
