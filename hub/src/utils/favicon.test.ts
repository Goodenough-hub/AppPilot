import { describe, it, expect, beforeEach } from 'vitest'
import { faviconCandidates, saveCandidate, dropCandidate, loadCache } from './favicon'

const ORIGIN = 'https://gitlab.infini-ai.com'

describe('favicon 缓存', () => {
  beforeEach(() => localStorage.clear())

  it('无缓存时候选为默认链顺序', () => {
    expect(faviconCandidates(ORIGIN)).toEqual([
      `${ORIGIN}/favicon.ico`,
      `${ORIGIN}/favicon.svg`,
      `${ORIGIN}/favicon.png`,
      `${ORIGIN}/apple-touch-icon.png`
    ])
  })

  it('saveCandidate 写入后候选以胜者为首且不重复', () => {
    saveCandidate(ORIGIN, `${ORIGIN}/favicon.png`)
    const c = faviconCandidates(ORIGIN)
    expect(c[0]).toBe(`${ORIGIN}/favicon.png`)
    expect(c).toHaveLength(4) // 胜者不重复出现
    expect(c).toContain(`${ORIGIN}/favicon.ico`)
  })

  it('saveCandidate 重复写同一地址不产生变化', () => {
    saveCandidate(ORIGIN, `${ORIGIN}/favicon.png`)
    const before = localStorage.getItem('hub_favicon_cache')
    saveCandidate(ORIGIN, `${ORIGIN}/favicon.png`)
    expect(localStorage.getItem('hub_favicon_cache')).toBe(before)
  })

  it('dropCandidate 清除对应 origin 的缓存', () => {
    saveCandidate(ORIGIN, `${ORIGIN}/favicon.png`)
    dropCandidate(ORIGIN)
    expect(loadCache()[ORIGIN]).toBeUndefined()
    expect(faviconCandidates(ORIGIN)[0]).toBe(`${ORIGIN}/favicon.ico`)
  })

  it('dropCandidate 对不存在条目不报错', () => {
    expect(() => dropCandidate('https://no-such.example.com')).not.toThrow()
  })

  it('localStorage 内容损坏时按无缓存处理', () => {
    localStorage.setItem('hub_favicon_cache', '{bad json')
    expect(loadCache()).toEqual({})
    expect(faviconCandidates(ORIGIN)[0]).toBe(`${ORIGIN}/favicon.ico`)
  })
})
