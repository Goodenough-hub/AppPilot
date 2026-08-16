import { describe, it, expect } from 'vitest'
import { timeAgo, domainOf } from './format'

describe('timeAgo', () => {
  it('返回 just now 时距不足 60s', () => {
    const now = new Date().toISOString()
    expect(timeAgo(now, new Date())).toMatch(/just now|刚刚/)
  })
  it('几分钟前', () => {
    const t = new Date(Date.now() - 5 * 60_000).toISOString()
    expect(timeAgo(t)).toBe('5m')
  })
  it('几小时前', () => {
    const t = new Date(Date.now() - 3 * 3600_000).toISOString()
    expect(timeAgo(t)).toBe('3h')
  })
  it('几天前', () => {
    const t = new Date(Date.now() - 2 * 86400_000).toISOString()
    expect(timeAgo(t)).toBe('2d')
  })
  it('超过 30 天返回日期', () => {
    const t = new Date(Date.now() - 60 * 86400_000).toISOString()
    expect(timeAgo(t)).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})

describe('domainOf', () => {
  it('提取域名', () => {
    expect(domainOf('https://github.com/x/y')).toBe('github.com')
    expect(domainOf('http://www.example.co.uk/a?b=c')).toBe('www.example.co.uk')
  })
  it('null / 空值 / 非法 URL 返回空串', () => {
    expect(domainOf(null)).toBe('')
    expect(domainOf('')).toBe('')
    expect(domainOf('not a url')).toBe('')
  })
})