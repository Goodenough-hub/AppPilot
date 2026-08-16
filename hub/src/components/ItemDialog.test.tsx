import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ItemDialog } from './ItemDialog'
import type { Item, ItemType, Folder } from '@/api/hub'

const foldersByType: Record<ItemType, Folder[]> = { bookmark: [], prompt: [], skill: [] }

function setup(opts: { defaultType?: ItemType; initial?: Item } = {}) {
  const onSubmit = vi.fn().mockResolvedValue(undefined)
  render(
    <ItemDialog
      open
      onOpenChange={() => {}}
      foldersByType={foldersByType}
      onSubmit={onSubmit}
      {...opts}
    />
  )
  return onSubmit
}

describe('ItemDialog 新增默认类型', () => {
  it('回归：Prompts 页新增（defaultType=prompt），不切类型直接保存应为 prompt 而非 bookmark', async () => {
    const onSubmit = setup({ defaultType: 'prompt' })
    expect(screen.getByLabelText('Prompt')).toBeChecked()
    fireEvent.change(screen.getByLabelText('标题'), { target: { value: 'Review 模板' } })
    fireEvent.click(screen.getByText('保存'))
    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ type: 'prompt', title: 'Review 模板' }))
    )
  })

  it('defaultType=skill 时新增默认选中 Skill', () => {
    setup({ defaultType: 'skill' })
    expect(screen.getByLabelText('Skill')).toBeChecked()
  })

  it('不传 defaultType 时回落 bookmark', () => {
    setup()
    expect(screen.getByLabelText('Bookmark')).toBeChecked()
  })

  it('编辑模式以 initial.type 为准，忽略 defaultType', () => {
    const editing: Item = {
      id: 1, type: 'skill', title: 'whisper', url: null, content: null,
      tags: [], favorite: false, folder: '', icon: '', position: 0, createdAt: '', updatedAt: ''
    }
    setup({ initial: editing, defaultType: 'prompt' })
    expect(screen.getByLabelText('Skill')).toBeChecked()
  })
})
