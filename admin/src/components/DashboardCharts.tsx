import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'
import type { User } from '../api/admin'

const AXIS = '#71717A'
const GRID = 'rgba(255,255,255,0.06)'
const INDIGO = '#818CF8'
const PURPLE = '#A855F7'

const tooltipStyle = {
  background: 'rgba(9, 9, 11, 0.95)',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: 12,
  fontSize: 13,
  color: '#fff',
}

/** Cumulative user registrations over time. */
export function UserGrowthChart({ users }: { users: User[] }) {
  const sorted = [...users].sort(
    (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
  )
  let running = 0
  const byDay = new Map<string, number>()
  for (const u of sorted) {
    const day = new Date(u.createdAt).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
    running += 1
    byDay.set(day, running)
  }
  const data = Array.from(byDay, ([date, total]) => ({ date, total }))

  return (
    <div className="glass-panel chart-card">
      <h2>用户增长</h2>
      <div className="subtitle">累计注册用户</div>
      <ResponsiveContainer width="100%" height={220}>
        <AreaChart data={data} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
          <defs>
            <linearGradient id="userGrowthFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={INDIGO} stopOpacity={0.35} />
              <stop offset="100%" stopColor={INDIGO} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis dataKey="date" stroke={AXIS} fontSize={12} tickLine={false} axisLine={{ stroke: GRID }} />
          <YAxis stroke={AXIS} fontSize={12} tickLine={false} axisLine={false} allowDecimals={false} width={32} />
          <Tooltip contentStyle={tooltipStyle} cursor={{ stroke: GRID }} />
          <Area
            type="monotone"
            dataKey="total"
            name="累计用户"
            stroke={INDIGO}
            strokeWidth={2}
            fill="url(#userGrowthFill)"
            animationDuration={280}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}

/** Admin vs regular-user split. */
export function RoleDonut({ admins, regular }: { admins: number; regular: number }) {
  const data = [
    { name: '管理员', value: admins, color: PURPLE },
    { name: '普通用户', value: regular, color: INDIGO },
  ]
  return (
    <div className="glass-panel chart-card">
      <h2>角色分布</h2>
      <div className="subtitle">管理员 / 普通用户</div>
      <ResponsiveContainer width="100%" height={220}>
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            innerRadius={54}
            outerRadius={80}
            paddingAngle={3}
            stroke="none"
            animationDuration={280}
          >
            {data.map(d => <Cell key={d.name} fill={d.color} />)}
          </Pie>
          <Tooltip contentStyle={tooltipStyle} />
        </PieChart>
      </ResponsiveContainer>
    </div>
  )
}
