const STORAGE_KEY = 'hub_favicon_cache'

/** 自动探测的候选路径（按常见程度排序，onError 依次后移） */
export const FAVICON_PATHS = ['/favicon.ico', '/favicon.svg', '/favicon.png', '/apple-touch-icon.png']

export function loadCache(): Record<string, string> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed: unknown = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? (parsed as Record<string, string>) : {}
  } catch {
    return {}
  }
}

function save(cache: Record<string, string>) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cache))
  } catch { /* 存储不可用时退化为不缓存 */ }
}

/** 探测成功的候选地址写入缓存（按 origin 记录） */
export function saveCandidate(origin: string, url: string) {
  const cache = loadCache()
  if (cache[origin] === url) return
  cache[origin] = url
  save(cache)
}

/** 全链失败时清掉该 origin 的过期缓存（下次重新完整探测） */
export function dropCandidate(origin: string) {
  const cache = loadCache()
  if (!(origin in cache)) return
  delete cache[origin]
  save(cache)
}

/** 候选列表：有缓存时缓存地址排第一（站点换图标时可自动自愈），其余按默认顺序跟后 */
export function faviconCandidates(origin: string): string[] {
  const base = FAVICON_PATHS.map((p) => origin + p)
  const cached = loadCache()[origin]
  if (!cached) return base
  return [cached, ...base.filter((u) => u !== cached)]
}
