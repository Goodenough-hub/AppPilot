import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { copyText } from './copy'

describe('copyText', () => {
  const original = Object.getOwnPropertyDescriptor(navigator, 'clipboard')

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    if (original) Object.defineProperty(navigator, 'clipboard', original)
  })

  it('优先调用 navigator.clipboard.writeText', async () => {
    const spy = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: spy }, configurable: true })
    await copyText('hello')
    expect(spy).toHaveBeenCalledWith('hello')
  })

  it('clipboard 不可用时降级 execCommand', async () => {
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true })
    const exec = vi.fn().mockReturnValue(true)
    document.execCommand = exec
    await copyText('hello')
    expect(exec).toHaveBeenCalledWith('copy')
  })

  it('clipboard 拒绝时降级 execCommand', async () => {
    const spy = vi.fn().mockRejectedValue(new Error('denied'))
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: spy }, configurable: true })
    const exec = vi.fn().mockReturnValue(true)
    document.execCommand = exec
    await copyText('hello')
    expect(exec).toHaveBeenCalledWith('copy')
  })

  it('clipboard 与 execCommand 都失败时抛错（调用方弹失败提示）', async () => {
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true })
    document.execCommand = vi.fn().mockReturnValue(false)
    await expect(copyText('hello')).rejects.toThrow()
  })
})
