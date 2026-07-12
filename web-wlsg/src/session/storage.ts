/** web-wlsg 独立本地会话存储，所有键均使用 hero3_wlsg_ 前缀。 */
export interface StoredSession {
  accountId: string
  username: string
  token: string
}

export interface SessionStorage {
  readSession: () => StoredSession | null
  writeSession: (session: StoredSession) => void
  clearSession: () => void
  readPlayerId: () => string | null
  writePlayerId: (playerId: string) => void
  clearPlayerId: () => void
  getToken: () => string | null
}

export const storageKeys = {
  accountId: 'hero3_wlsg_account_id',
  username: 'hero3_wlsg_account_name',
  token: 'hero3_wlsg_token',
  playerId: 'hero3_wlsg_player_id',
} as const

/** 创建仅操作 web-wlsg 专用键的会话存储。 */
export function createSessionStorage(storage: Storage): SessionStorage {
  return {
    /** 读取完整会话，字段缺失时视为无会话。 */
    readSession() {
      const accountId = storage.getItem(storageKeys.accountId)
      const username = storage.getItem(storageKeys.username)
      const token = storage.getItem(storageKeys.token)
      return accountId && username && token ? { accountId, username, token } : null
    },
    /** 保存登录成功后的账号和 JWT。 */
    writeSession(session) {
      storage.setItem(storageKeys.accountId, session.accountId)
      storage.setItem(storageKeys.username, session.username)
      storage.setItem(storageKeys.token, session.token)
    },
    /** 清理当前新前端账号会话。 */
    clearSession() {
      storage.removeItem(storageKeys.accountId)
      storage.removeItem(storageKeys.username)
      storage.removeItem(storageKeys.token)
    },
    readPlayerId: () => storage.getItem(storageKeys.playerId),
    writePlayerId: (playerId) => storage.setItem(storageKeys.playerId, playerId),
    clearPlayerId: () => storage.removeItem(storageKeys.playerId),
    getToken: () => storage.getItem(storageKeys.token),
  }
}
