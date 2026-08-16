import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ItemCard } from './ItemCard'
import { FolderSection } from './FolderSection'
import { ToastProvider } from '@/components/ui/Toast'
import type { Item } from '@/api/hub'

const base: Item = {
  id: 1, type: 'prompt', title: 'Code review', url: null,
  content: '第一行\n第二行\n第三行', tags: [], favorite: false,
  folder: '', icon: '', position: 0, createdAt: '', updatedAt: ''
}

const writeText = vi.fn()

function setup(item: Item = base) {
  const handlers = {
    onToggleFav: vi.fn(),
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onTagClick: vi.fn()
  }
  render(
    <ToastProvider>
      <ItemCard item={item} {...handlers} />
    </ToastProvider>
  )
  return handlers
}

describe('ItemCard', () => {
  beforeEach(() => {
    writeText.mockReset()
    writeText.mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
  })

  it('内容块以 pre-wrap 等宽字体渲染，整块可点击复制', async () => {
    setup()
    const block = screen.getByTitle('点击复制内容')
    expect(block.style.whiteSpace).toBe('pre-wrap')
    expect(block.textContent).toContain('第一行\n第二行')
    fireEvent.click(block)
    await waitFor(() => expect(writeText).toHaveBeenCalledWith('第一行\n第二行\n第三行'))
  })

  it('右上角常驻复制按钮：点击写入剪贴板并反馈已复制', async () => {
    setup()
    const btn = screen.getByLabelText('复制内容')
    fireEvent.click(btn)
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(base.content))
    expect(await screen.findByLabelText('已复制')).toBeInTheDocument()
  })

  it('复制成功后弹出 toast 通知「已复制」', async () => {
    setup()
    fireEvent.click(screen.getByLabelText('复制内容'))
    await waitFor(() => expect(writeText).toHaveBeenCalled())
    // Radix Toast 标题渲染「已复制」文本
    expect(await screen.findByText('已复制', { selector: '[class], div, span' })).toBeInTheDocument()
  })

  it('复制按钮点击不冒泡触发内容块二次复制', async () => {
    setup()
    fireEvent.click(screen.getByLabelText('复制内容'))
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1))
  })

  it('复制失败时弹「复制失败」提示而非成功', async () => {
    writeText.mockRejectedValue(new Error('denied'))
    document.execCommand = vi.fn().mockReturnValue(false)
    setup()
    fireEvent.click(screen.getByLabelText('复制内容'))
    expect(await screen.findByText('复制失败', { selector: '[class], div, span' })).toBeInTheDocument()
  })

  it('键盘 Enter/Space 在内容块上也触发复制', async () => {
    setup()
    const block = screen.getByTitle('点击复制内容')
    fireEvent.keyDown(block, { key: 'Enter' })
    fireEvent.keyDown(block, { key: ' ' })
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(2))
  })

  it('无内容时不渲染内容块与复制按钮', () => {
    setup({ ...base, content: null })
    expect(screen.queryByTitle('点击复制内容')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('复制内容')).not.toBeInTheDocument()
  })

  it('edit/delete 回调不变（动作行不再重复 copy 按钮）', () => {
    const h = setup()
    fireEvent.click(screen.getByText('edit'))
    expect(h.onEdit).toHaveBeenCalledWith(base)
    fireEvent.click(screen.getByText('delete'))
    expect(h.onDelete).toHaveBeenCalledWith(1)
    expect(screen.queryByText('copy')).not.toBeInTheDocument()
  })

  it('收藏按钮回调与标签 chip 回调', () => {
    const h = setup({ ...base, tags: ['review'] })
    fireEvent.click(screen.getByLabelText('收藏'))
    expect(h.onToggleFav).toHaveBeenCalledWith(1, true)
    fireEvent.click(screen.getByText('#review'))
    expect(h.onTagClick).toHaveBeenCalledWith('review')
  })
})

describe('FolderSection 卡片网格', () => {
  it('非 dense（prompt/skill）用 card-grid 容器，一行四列样式挂全局 CSS', () => {
    const { container } = render(
      <FolderSection name="F" count={1} collapsed={false} onToggle={() => {}}>
        <div>card</div>
      </FolderSection>
    )
    expect(container.querySelector('.card-grid')).not.toBeNull()
  })

  it('dense（书签）保持纵向紧凑列表，不用网格', () => {
    const { container } = render(
      <FolderSection name="F" count={1} collapsed={false} onToggle={() => {}} dense>
        <div>row</div>
      </FolderSection>
    )
    expect(container.querySelector('.card-grid')).toBeNull()
  })
})
