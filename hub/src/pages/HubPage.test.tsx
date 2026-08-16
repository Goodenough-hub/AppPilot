import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import HubPage from './HubPage'
import { ToastProvider } from '@/components/ui/Toast'
import { itemsApi, foldersApi, faviconApi } from '@/api/hub'
import type { Item } from '@/api/hub'

vi.mock('@/api/hub', () => ({
  itemsApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    exportJson: vi.fn(),
    importJson: vi.fn(),
    reorder: vi.fn()
  },
  foldersApi: { list: vi.fn(), create: vi.fn(), rename: vi.fn(), remove: vi.fn() },
  faviconApi: { discover: vi.fn() }
}))

const mockItems = vi.mocked(itemsApi)
const mockFolders = vi.mocked(foldersApi)

/** 两个书签分属不同文件夹：A1/A2 在「工作」，B1 在「个人」 */
const bookmarkItems: Item[] = [
  { id: 1, type: 'bookmark', title: 'A1', url: 'https://a1.example.com/', content: null, tags: [], favorite: false, folder: '工作', icon: '', position: 1, createdAt: '', updatedAt: '' },
  { id: 2, type: 'bookmark', title: 'A2', url: 'https://a2.example.com/', content: null, tags: [], favorite: false, folder: '工作', icon: '', position: 2, createdAt: '', updatedAt: '' },
  { id: 3, type: 'bookmark', title: 'B1', url: 'https://b1.example.com/', content: null, tags: [], favorite: false, folder: '个人', icon: '', position: 1, createdAt: '', updatedAt: '' }
]

/** folderNames：已登记的 bookmark 文件夹（可含无条目的空文件夹） */
function setupPage(folderNames: string[] = ['工作', '个人']) {
  mockItems.list.mockResolvedValue(bookmarkItems)
  mockItems.update.mockImplementation(async (id, patch) => ({ ...bookmarkItems.find((i) => i.id === id)!, ...patch }))
  mockItems.reorder.mockResolvedValue(undefined)
  mockItems.exportJson.mockResolvedValue([])
  mockItems.importJson.mockResolvedValue({ affected: 0, mode: 'merge' })
  mockFolders.list.mockImplementation(async (type) =>
    type === 'bookmark'
      ? folderNames.map((name, i) => ({ id: i + 1, type: 'bookmark' as const, name, itemCount: 0, createdAt: '' }))
      : [])
  mockFolders.create.mockResolvedValue({ id: 9, type: 'bookmark', name: 'x', itemCount: 0, createdAt: '' })
  vi.mocked(faviconApi.discover).mockResolvedValue([])
  render(
    <ToastProvider>
      <HubPage />
    </ToastProvider>
  )
}

/** dataTransfer 桩：jsdom 拖拽事件不带它，handler 需要 setData/effectAllowed */
const dt = () => ({ setData: vi.fn(), effectAllowed: '' })

const rowOf = (el: HTMLElement) => el.closest('[draggable]')!

describe('HubPage 拖拽', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('回归：跨文件夹拖拽书签会改 folder 并按落点持久化目标组 position（旧行为直接忽略）', async () => {
    setupPage()
    const a1 = await screen.findByText('A1')
    const b1 = await screen.findByText('B1')
    // A1 拖到 B1 前半区（clientY=-100 在 rect 中点之上 → after=false）= 插到目标组第 1 位
    fireEvent.dragStart(rowOf(a1), { dataTransfer: dt() })
    fireEvent.dragOver(rowOf(b1), { dataTransfer: dt(), clientY: -100 })
    fireEvent.drop(rowOf(b1), { dataTransfer: dt(), clientY: -100 })

    await waitFor(() => expect(mockItems.update).toHaveBeenCalledWith(1, { folder: '个人' }))
    await waitFor(() => expect(mockItems.reorder).toHaveBeenCalledWith('bookmark', '个人', [1, 3]))
  })

  it('同文件夹拖拽仍是组内重排，不改 folder（jsdom 中 after 恒 false，拖后者到前者上半区）', async () => {
    setupPage()
    const a2 = await screen.findByText('A2')
    const a1 = await screen.findByText('A1')
    // A2 拖到 A1 上半区（jsdom 中 after=false）= 插到 A1 之前 → 顺序 [A2, A1]
    fireEvent.dragStart(rowOf(a2), { dataTransfer: dt() })
    fireEvent.dragOver(rowOf(a1), { dataTransfer: dt(), clientY: -100 })
    fireEvent.drop(rowOf(a1), { dataTransfer: dt(), clientY: -100 })

    await waitFor(() => expect(mockItems.reorder).toHaveBeenCalledWith('bookmark', '工作', [2, 1]))
    expect(mockItems.update).not.toHaveBeenCalled()
  })

  it('拖到空文件夹的空态区：改 folder 追加，无需 reorder', async () => {
    setupPage(['工作', '空组', '个人'])
    const a1 = await screen.findByText('A1')
    const emptyZone = await screen.findByText('空文件夹')
    fireEvent.dragStart(rowOf(a1), { dataTransfer: dt() })
    fireEvent.dragOver(emptyZone, { dataTransfer: dt() })
    fireEvent.drop(emptyZone, { dataTransfer: dt() })

    await waitFor(() => expect(mockItems.update).toHaveBeenCalledWith(1, { folder: '空组' }))
    expect(mockItems.reorder).not.toHaveBeenCalled()
  })

  it('prompt/skill 卡片也可拖拽（落点按左右半区判定）', async () => {
    // 两张 prompt 卡：P1 在「工作」，P2 在「个人」
    const prompts: Item[] = [
      { id: 11, type: 'prompt', title: 'P1', url: null, content: 'p1', tags: [], favorite: false, folder: '工作', icon: '', position: 1, createdAt: '', updatedAt: '' },
      { id: 12, type: 'prompt', title: 'P2', url: null, content: 'p2', tags: [], favorite: false, folder: '个人', icon: '', position: 1, createdAt: '', updatedAt: '' }
    ]
    mockItems.list.mockResolvedValue(prompts)
    mockItems.update.mockImplementation(async (id, patch) => ({ ...prompts.find((i) => i.id === id)!, ...patch }))
    mockFolders.list.mockImplementation(async () => [
      { id: 1, type: 'prompt', name: '工作', itemCount: 0, createdAt: '' },
      { id: 2, type: 'prompt', name: '个人', itemCount: 0, createdAt: '' }
    ])
    render(
      <ToastProvider>
        <HubPage />
      </ToastProvider>
    )
    // 切到 Prompts tab
    fireEvent.click(await screen.findByRole('button', { name: /prompt/i }))
    const p1 = await screen.findByText('P1')
    const p2 = await screen.findByText('P2')
    // P1 拖到 P2 左半区（jsdom 中 after=false）= 插到 P2 之前 → [P1, P2]
    fireEvent.dragStart(rowOf(p1), { dataTransfer: dt() })
    fireEvent.dragOver(rowOf(p2), { dataTransfer: dt(), clientX: -100 })
    fireEvent.drop(rowOf(p2), { dataTransfer: dt(), clientX: -100 })

    await waitFor(() => expect(mockItems.update).toHaveBeenCalledWith(11, { folder: '个人' }))
    await waitFor(() => expect(mockItems.reorder).toHaveBeenCalledWith('prompt', '个人', [11, 12]))
  })
})
