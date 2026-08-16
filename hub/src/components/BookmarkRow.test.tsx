import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BookmarkRow } from './BookmarkRow'
import type { Item } from '@/api/hub'

const base: Item = {
  id: 1, type: 'bookmark', title: 'Infini-AI GitLab', url: 'https://gitlab.infini-ai.com/',
  content: null, tags: [], favorite: false, folder: 'Infini-AI', icon: '', createdAt: '', updatedAt: ''
}

function setup(item: Item = base) {
  const handlers = {
    onToggleFav: vi.fn(),
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onTagClick: vi.fn()
  }
  render(<BookmarkRow item={item} {...handlers} />)
  return handlers
}

describe('BookmarkRow', () => {
  beforeEach(() => localStorage.clear())

  it('标题渲染为新窗口打开的跳转链接，并显示域名', () => {
    setup()
    const link = screen.getByRole('link', { name: 'Infini-AI GitLab' })
    expect(link).toHaveAttribute('href', 'https://gitlab.infini-ai.com/')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', expect.stringContaining('noopener'))
    expect(screen.getByText('gitlab.infini-ai.com')).toBeInTheDocument()
  })

  it('无 URL 时标题渲染为纯文本', () => {
    setup({ ...base, url: null })
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.getByText('Infini-AI GitLab')).toBeInTheDocument()
  })

  it('操作按钮回调正确：收藏/编辑/删除', () => {
    const h = setup()
    fireEvent.click(screen.getByLabelText('收藏'))
    expect(h.onToggleFav).toHaveBeenCalledWith(1, true)
    fireEvent.click(screen.getByLabelText('编辑 Infini-AI GitLab'))
    expect(h.onEdit).toHaveBeenCalledWith(base)
    fireEvent.click(screen.getByLabelText('删除 Infini-AI GitLab'))
    expect(h.onDelete).toHaveBeenCalledWith(1)
  })

  it('已收藏条目显示常驻收藏标记，且收藏按钮为取消语义', () => {
    setup({ ...base, favorite: true })
    expect(screen.getByLabelText('已收藏')).toBeInTheDocument()
    expect(screen.getByLabelText('取消收藏')).toBeInTheDocument()
  })

  it('未收藏条目不显示常驻收藏标记', () => {
    setup()
    expect(screen.queryByLabelText('已收藏')).not.toBeInTheDocument()
  })

  it('标签渲染为可点击 chip 并回调 onTagClick', () => {
    const h = setup({ ...base, tags: ['内网'] })
    fireEvent.click(screen.getByText('#内网'))
    expect(h.onTagClick).toHaveBeenCalledWith('内网')
  })

  it('有 URL 时渲染站点 favicon（no-referrer + lazy）', () => {
    const { container } = render(
      <BookmarkRow item={base} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    expect(img!.getAttribute('src')).toBe('https://gitlab.infini-ai.com/favicon.ico')
    expect(img!.getAttribute('referrerpolicy')).toBe('no-referrer')
    expect(img!.getAttribute('loading')).toBe('lazy')
  })

  it('favicon 加载失败按候选链依次后移，全部失败回落为默认图标', () => {
    const { container } = render(
      <BookmarkRow item={base} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    const origin = 'https://gitlab.infini-ai.com'
    // 每失败一次换下一个候选路径
    for (const path of ['/favicon.ico', '/favicon.svg', '/favicon.png']) {
      const img = container.querySelector('img')
      expect(img!.getAttribute('src')).toBe(origin + path)
      fireEvent.error(img!)
    }
    fireEvent.error(container.querySelector('img')!) // apple-touch-icon.png 也失败
    expect(container.querySelector('img')).toBeNull()
  })

  it('有自定义 icon 时只用自定义地址，失败后直接回落', () => {
    const { container } = render(
      <BookmarkRow item={{ ...base, icon: 'https://cdn.example.com/logo.png' }} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    const img = container.querySelector('img')!
    expect(img.getAttribute('src')).toBe('https://cdn.example.com/logo.png')
    fireEvent.error(img)
    expect(container.querySelector('img')).toBeNull()
  })

  it('无 URL 时不渲染 favicon', () => {
    const { container } = render(
      <BookmarkRow item={{ ...base, url: null }} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    expect(container.querySelector('img')).toBeNull()
  })

  it('有缓存时优先用缓存地址', () => {
    localStorage.setItem('hub_favicon_cache', JSON.stringify({
      'https://gitlab.infini-ai.com': 'https://gitlab.infini-ai.com/favicon.png'
    }))
    const { container } = render(
      <BookmarkRow item={base} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    expect(container.querySelector('img')!.getAttribute('src')).toBe('https://gitlab.infini-ai.com/favicon.png')
  })

  it('探测成功（onLoad）后把胜者写入缓存', () => {
    const { container } = render(
      <BookmarkRow item={base} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    fireEvent.load(container.querySelector('img')!)
    const cache = JSON.parse(localStorage.getItem('hub_favicon_cache')!)
    expect(cache['https://gitlab.infini-ai.com']).toBe('https://gitlab.infini-ai.com/favicon.ico')
  })

  it('自定义 icon 加载成功不写缓存', () => {
    const { container } = render(
      <BookmarkRow item={{ ...base, icon: 'https://cdn.example.com/logo.png' }} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    fireEvent.load(container.querySelector('img')!)
    expect(localStorage.getItem('hub_favicon_cache')).toBeNull()
  })

  it('全链失败后清掉该 origin 的过期缓存', () => {
    localStorage.setItem('hub_favicon_cache', JSON.stringify({
      'https://gitlab.infini-ai.com': 'https://gitlab.infini-ai.com/favicon.png'
    }))
    const { container } = render(
      <BookmarkRow item={base} onToggleFav={() => {}} onEdit={() => {}} onDelete={() => {}} onTagClick={() => {}} />
    )
    // 候选 = [favicon.png(缓存), favicon.ico, favicon.svg, apple-touch-icon.png]，共 4 个
    for (let i = 0; i < 4; i++) {
      fireEvent.error(container.querySelector('img')!)
    }
    expect(container.querySelector('img')).toBeNull()
    const raw = localStorage.getItem('hub_favicon_cache')
    const cache = raw ? JSON.parse(raw) : {}
    expect(cache['https://gitlab.infini-ai.com']).toBeUndefined()
  })
})
