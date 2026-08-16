import { describe, it, expect, vi } from 'vitest'
import { copyText } from './copy'

describe('copyText', () => {
  it('调用 navigator.clipboard.writeText', async () => {
    const spy = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText: spy } })
    await copyText('hello')
    expect(spy).toHaveBeenCalledWith('hello')
  })
})