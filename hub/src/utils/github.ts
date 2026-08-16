export interface RepoInfo {
  title: string
  content: string
  tags: string[]
}

export function parseRepo(url: string): { owner: string; repo: string } | null {
  if (!url) return null
  const m = url.match(/github\.com\/([^\/]+)\/([^\/\?#]+)/)
  if (!m) return null
  return { owner: m[1], repo: m[2] }
}

export async function fetchRepoInfo(owner: string, repo: string): Promise<RepoInfo> {
  const res = await fetch(`https://api.github.com/repos/${owner}/${repo}`)
  if (!res.ok) throw new Error(`GitHub API failed: ${res.status}`)
  const data = await res.json()
  const tags: string[] = []
  if (data.language) tags.push(data.language)
  tags.push('GitHub')
  return {
    title: data.full_name,
    content: data.description ?? '',
    tags
  }
}