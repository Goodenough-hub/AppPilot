export function timeAgo(iso: string, now: Date = new Date()): string {
  const t = new Date(iso).getTime()
  const diffSec = Math.floor((now.getTime() - t) / 1000)
  if (diffSec < 60) return 'just now'
  const min = Math.floor(diffSec / 60)
  if (min < 60) return `${min}m`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h`
  const day = Math.floor(hr / 24)
  if (day < 30) return `${day}d`
  // 超过 30 天返回 YYYY-MM-DD
  return iso.slice(0, 10)
}

export function domainOf(url: string | null | undefined): string {
  if (!url) return ''
  try {
    return new URL(url).host
  } catch {
    return ''
  }
}