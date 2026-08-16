import { describe, it, expect } from 'vitest'
import { groupByFolder } from './group'
import type { Folder, Item } from '@/api/hub'

function item(id: number, folder: string): Item {
  return { id, type: 'bookmark', title: `t${id}`, url: null, content: null, tags: [], favorite: false, folder, icon: '', position: 0, createdAt: '', updatedAt: '' }
}
function folder(id: number, name: string): Folder {
  return { id, type: 'bookmark', name, itemCount: 0, createdAt: '' }
}

describe('groupByFolder', () => {
  it('空输入返回空数组', () => {
    expect(groupByFolder([], [])).toEqual([])
  })

  it('已登记文件夹保持传入顺序在前，未分类固定最后', () => {
    const groups = groupByFolder(
      [item(1, 'B'), item(2, ''), item(3, 'A'), item(4, '')],
      [folder(1, 'A'), folder(2, 'B')]
    )
    expect(groups.map((g) => g.folder)).toEqual(['A', 'B', ''])
    expect(groups[0].items.map((i) => i.id)).toEqual([3])
    expect(groups[0].folderId).toBe(1)
    expect(groups[1].items.map((i) => i.id)).toEqual([1])
    expect(groups[2].items.map((i) => i.id)).toEqual([2, 4])
    expect(groups[2].folderId).toBeNull()
  })

  it('空文件夹也保留（0 条分组不丢）', () => {
    const groups = groupByFolder([], [folder(1, 'empty')])
    expect(groups).toHaveLength(1)
    expect(groups[0].folder).toBe('empty')
    expect(groups[0].items).toEqual([])
  })

  it('孤儿名按字典序排在已登记文件夹之后、未分类之前', () => {
    const groups = groupByFolder(
      [item(1, 'zeta'), item(2, 'alpha'), item(3, '')],
      [folder(1, 'reg')]
    )
    expect(groups.map((g) => g.folder)).toEqual(['reg', 'alpha', 'zeta', ''])
  })

  it('组内保持传入顺序', () => {
    const groups = groupByFolder([item(3, 'A'), item(1, 'A'), item(2, 'A')], [])
    expect(groups[0].items.map((i) => i.id)).toEqual([3, 1, 2])
  })

  it('没有未分类条目时不出现未分类分组', () => {
    const groups = groupByFolder([item(1, 'A')], [])
    expect(groups.map((g) => g.folder)).toEqual(['A'])
  })
})
