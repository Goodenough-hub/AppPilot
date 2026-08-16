import { describe, it, expect } from 'vitest'
import { filterItems } from './filter'
import type { Item } from '@/api/hub'

const items: Item[] = [
  { id: 1, type: 'bookmark', title: 'GitHub', url: 'https://github.com', content: null, tags: ['dev'], favorite: false, folder: '', icon: '', position: 0, createdAt: '', updatedAt: '' },
  { id: 2, type: 'prompt', title: 'C++ refactor', url: null, content: 'You are a systems engineer...', tags: ['C++', 'perf'], favorite: true, folder: '', icon: '', position: 0, createdAt: '', updatedAt: '' },
  { id: 3, type: 'skill', title: 'whisper.cpp', url: 'https://github.com/ggerganov/whisper.cpp', content: 'edge asr', tags: ['C++', 'audio'], favorite: false, folder: '', icon: '', position: 0, createdAt: '', updatedAt: '' }
]

describe('filterItems', () => {
  it('按 type 过滤', () => {
    expect(filterItems(items, { tab: 'bookmark' }).map(i => i.id)).toEqual([1])
    expect(filterItems(items, { tab: 'prompt' }).map(i => i.id)).toEqual([2])
    expect(filterItems(items, { tab: 'skill' }).map(i => i.id)).toEqual([3])
  })
  it('按 tag 过滤（大小写不敏感）', () => {
    expect(filterItems(items, { tab: 'prompt', tag: 'c++' }).map(i => i.id)).toEqual([2])
    expect(filterItems(items, { tab: 'skill', tag: 'c++' }).map(i => i.id)).toEqual([3])
    expect(filterItems(items, { tab: 'bookmark', tag: 'c++' })).toEqual([])
  })
  it('按 query 搜索 title/content/tags', () => {
    expect(filterItems(items, { tab: 'prompt', query: 'engineer' }).map(i => i.id)).toEqual([2])
    expect(filterItems(items, { tab: 'skill', query: 'AUDIO' }).map(i => i.id)).toEqual([3])
    expect(filterItems(items, { tab: 'bookmark', query: 'github' }).map(i => i.id)).toEqual([1])
  })
  it('tab + tag + query 组合', () => {
    expect(filterItems(items, { tab: 'skill', tag: 'C++', query: 'whisper' }).map(i => i.id)).toEqual([3])
    expect(filterItems(items, { tab: 'bookmark', tag: 'C++' })).toEqual([])
  })
})
