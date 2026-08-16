import '@testing-library/jest-dom'

// Node.js 25 提供实验性全局 localStorage（无 Storage API 方法），会覆盖 jsdom 的 localStorage。
// 用内存实现替换，确保测试环境有完整的 Storage API。
const store = new Map<string, string>()
const storage: Storage = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, value: string) => { store.set(key, value) },
  removeItem: (key: string) => { store.delete(key) },
  clear: () => { store.clear() },
  get length() { return store.size },
  key: (index: number) => [...store.keys()][index] ?? null
}
Object.defineProperty(globalThis, 'localStorage', { value: storage, configurable: true })

// jsdom 默认不实现 matchMedia，加一个 stub 供 useTheme 的 prefers-color-scheme 监听使用。
Object.defineProperty(window, 'matchMedia', {
  configurable: true,
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false
  })
})