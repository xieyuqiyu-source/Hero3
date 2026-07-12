/** 验证新前端会话键与现有 Hero3 前端完全隔离。 */
import { describe, expect, it } from 'vitest'
import { createSessionStorage, storageKeys } from './storage'

/** 创建测试所需的最小内存 Storage。 */
function memoryStorage(): Storage {
  const values = new Map<string, string>()
  return {
    get length() { return values.size },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => { values.delete(key) },
    setItem: (key, value) => { values.set(key, String(value)) },
  }
}

describe('web-wlsg 会话存储', () => {
  it('只写入 hero3_wlsg_ 前缀键且不保存密码', () => {
    const raw = memoryStorage()
    const storage = createSessionStorage(raw)
    storage.writeSession({ accountId: 'a1', username: 'hero', token: 'jwt' })
    storage.writePlayerId('p1')
    expect(Object.values(storageKeys).every((key) => key.startsWith('hero3_wlsg_'))).toBe(true)
    expect([...Array(raw.length)].map((_, index) => raw.key(index))).not.toContain('password')
  })

  it('退出不删除旧前端键', () => {
    const raw = memoryStorage()
    raw.setItem('hero3_token', 'legacy-token')
    const storage = createSessionStorage(raw)
    storage.writeSession({ accountId: 'a1', username: 'hero', token: 'jwt' })
    storage.clearSession()
    storage.clearPlayerId()
    expect(raw.getItem('hero3_token')).toBe('legacy-token')
    expect(storage.readSession()).toBeNull()
  })
})
