/**
 * 复制文本。优先 navigator.clipboard；在非安全上下文/权限被拒时降级
 * execCommand('copy')，保证复制总能尽力成功。两者都失败才抛错（调用方提示）。
 */
export async function copyText(s: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(s)
      return
    } catch {
      /* 降级到 execCommand */
    }
  }
  const ta = document.createElement('textarea')
  ta.value = s
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.focus()
  ta.select()
  try {
    if (!document.execCommand('copy')) throw new Error('copy command rejected')
  } finally {
    document.body.removeChild(ta)
  }
}
