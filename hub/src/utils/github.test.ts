import { describe, it, expect, vi } from 'vitest'
import { parseRepo, fetchRepoInfo } from './github'

describe('parseRepo', () => {
  it('提取 owner/repo', () => {
    expect(parseRepo('https://github.com/ggerganov/whisper.cpp')).toEqual({ owner: 'ggerganov', repo: 'whisper.cpp' })
    expect(parseRepo('https://github.com/anthropic-ai/claude-code/tree/main')).toEqual({ owner: 'anthropic-ai', repo: 'claude-code' })
  })
  it('非 github URL 返回 null', () => {
    expect(parseRepo('https://gitlab.com/x/y')).toBeNull()
    expect(parseRepo('nope')).toBeNull()
    expect(parseRepo('')).toBeNull()
  })
})

describe('fetchRepoInfo', () => {
  it('拉 api.github.com 并转换字段', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ full_name: 'x/y', description: 'desc', language: 'Go' })
    })
    global.fetch = mockFetch as any
    const info = await fetchRepoInfo('x', 'y')
    expect(mockFetch).toHaveBeenCalledWith('https://api.github.com/repos/x/y')
    expect(info).toEqual({ title: 'x/y', content: 'desc', tags: ['Go', 'GitHub'] })
  })
  it('非 2xx 抛错', async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: false, status: 404 }) as any
    await expect(fetchRepoInfo('x', 'y')).rejects.toThrow(/failed|404/i)
  })
})