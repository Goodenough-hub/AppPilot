import { describe, it, expect } from 'vitest'
import { filterItems } from './filter'
import type { Item } from '@/api/hub'

const items: Item[] = [
  { id: 1, type: 'bookmark', title: 'GitHub', url: 'https://github.com', content: null, tags: ['dev'], favorite: false, createdAt: '', updatedAt: '' },
  { id: 2, type: 'prompt', title: 'C++ refactor', url: null, content: 'You are a systems engineer...', tags: ['C++', 'perf'], favorite: true, createdAt: '', updatedAt: '' },
  { id: 3, type: 'skill', title: 'whisper.cpp', url: 'https://github.com/ggerganov/whisper.cpp', content: 'edge asr', tags: ['C++', 'audio'], favorite: false, createdAt: '', updatedAt: '' }
]

describe('filterItems', () => {
  it('按 type 过滤', () => {
    expect(filterItems(items, { tab: 'bookmark' }).map(i => i.id)).toEqual([1])
    expect(filterItems(items, { tab: 'prompt' }).map(i => i.id)).toEqual([2])
  })
  it('starred tab 只留 favorite', () => {
    expect(filterItems(items, { tab: 'starred' }).map(i => i.id)).toEqual([2])
  })
  it('按 tag 过滤（大小写不敏感）', () => {
    expect(filterItems(items, { tab: 'all', tag: 'c++' }).map(i => i.id)).toEqual([2, 3])
  })
  it('按 query 搜索 title/content/tags', () => {
    expect(filterItems(items, { tab: 'all', query: 'engineer' }).map(i => i.id)).toEqual([2])
    expect(filterItems(items, { tab: 'all', query: 'AUDIO' }).map(i => i.id)).toEqual([3])
    expect(filterItems(items, { tab: 'all', query: 'github' }).map(i => i.id)).toEqual([1, 3])
  })
  it('tab + tag + query 组合', () => {
    expect(filterItems(items, { tab: 'skill', tag: 'C++', query: 'whisper' }).map(i => i.id)).toEqual([3])
    expect(filterItems(items, { tab: 'bookmark', tag: 'C++' })).toEqual([])
  })
})