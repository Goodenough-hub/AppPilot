/**
 * 把 arr[from] 移动到 arr[to]（to 为移除 from 之后的目标下标）。
 * 用于拖拽落点换算：moveItem([a,b,c,d], 0, 2) → [b,c,a,d]。
 * 不修改入参，返回新数组。
 */
export function moveItem<T>(arr: readonly T[], from: number, to: number): T[] {
  const copy = [...arr]
  const [x] = copy.splice(from, 1)
  copy.splice(to, 0, x)
  return copy
}
