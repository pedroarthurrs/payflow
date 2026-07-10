// Global test setup.
//
// Node 22+ ships an experimental global `localStorage` (enabled by default in
// newer releases) that requires a backing file and otherwise shadows jsdom's
// working `window.localStorage` with a non-functional stub. To keep tests
// deterministic across Node versions, we install a simple in-memory Storage
// implementation on both `window` and `globalThis` before each file runs.
import { beforeEach } from 'vitest'

class MemoryStorage implements Storage {
  private store = new Map<string, string>()

  get length(): number {
    return this.store.size
  }

  clear(): void {
    this.store.clear()
  }

  getItem(key: string): string | null {
    return this.store.has(key) ? this.store.get(key)! : null
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null
  }

  removeItem(key: string): void {
    this.store.delete(key)
  }

  setItem(key: string, value: string): void {
    this.store.set(key, String(value))
  }
}

function installStorage(name: 'localStorage' | 'sessionStorage'): void {
  const storage = new MemoryStorage()
  const descriptor: PropertyDescriptor = {
    value: storage,
    configurable: true,
    writable: true,
  }
  Object.defineProperty(globalThis, name, descriptor)
  if (typeof window !== 'undefined') {
    Object.defineProperty(window, name, descriptor)
  }
}

beforeEach(() => {
  installStorage('localStorage')
  installStorage('sessionStorage')
})
