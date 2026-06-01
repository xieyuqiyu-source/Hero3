import { create } from 'zustand'
import { gameApi } from '@/api/game'
import type { AccountSession, PlayerSummary } from '@/types/game'

function clearActivePlayerSession() {
  localStorage.removeItem('hero3_active_player_id')
  window.dispatchEvent(new Event('hero3:clear-active-player'))
}

interface AccountStore {
  /** 当前登录的账号 */
  account: AccountSession | null
  /** 该账号下的存档列表 */
  players: PlayerSummary[]
  /** 加载中 */
  loading: boolean

  /** 登录 */
  login: (username: string, password: string) => Promise<void>
  /** 注册 */
  register: (username: string, password: string) => Promise<void>
  /** 退出登录 */
  logout: () => void
  /** 加载存档列表 */
  loadPlayers: () => Promise<void>
  /** 删除存档 */
  deletePlayer: (playerId: string) => Promise<void>
  /** 从 localStorage 恢复会话 */
  restore: () => void
}

export const useAccountStore = create<AccountStore>((set, get) => ({
  account: null,
  players: [],
  loading: false,

  login: async (username, password) => {
    set({ loading: true })
    try {
      const session = await gameApi.loginAccount(username, password)
      localStorage.setItem('hero3_account_id', session.accountId)
      localStorage.setItem('hero3_account_name', session.username)
      if (session.token) localStorage.setItem('hero3_token', session.token)
      set({ account: session, loading: false })
      // Auto-load players after login
      await get().loadPlayers()
    } catch {
      set({ loading: false })
      throw new Error('登录失败')
    }
  },

  register: async (username, password) => {
    set({ loading: true })
    try {
      const session = await gameApi.registerAccount(username, password)
      localStorage.setItem('hero3_account_id', session.accountId)
      localStorage.setItem('hero3_account_name', session.username)
      if (session.token) localStorage.setItem('hero3_token', session.token)
      set({ account: session, loading: false })
    } catch {
      set({ loading: false })
      throw new Error('注册失败')
    }
  },

  logout: () => {
    localStorage.removeItem('hero3_account_id')
    localStorage.removeItem('hero3_account_name')
    localStorage.removeItem('hero3_token')
    clearActivePlayerSession()
    set({ account: null, players: [] })
  },

  loadPlayers: async () => {
    const account = get().account
    if (!account) return
    try {
      const result = await gameApi.listAccountPlayers(account.accountId)
      set({ players: result.players ?? [] })
    } catch (e) {
      console.warn('[accountStore] loadPlayers failed:', e)
    }
  },

  deletePlayer: async (playerId: string) => {
    await gameApi.deletePlayer(playerId)
    // 删除后刷新列表
    const players = get().players.filter((p) => p.id !== playerId)
    set({ players })
  },

  restore: () => {
    const accountId = localStorage.getItem('hero3_account_id')
    const username = localStorage.getItem('hero3_account_name')
    const token = localStorage.getItem('hero3_token')

    // 没有账号信息直接结束
    if (!accountId || !username) return

    // 升级到 JWT 版本后老 session 没有 token，引导用户重新登录
    // 直接清掉本地 session，让 RequirePlayer 把用户带去登录页
    if (!token) {
      localStorage.removeItem('hero3_account_id')
      localStorage.removeItem('hero3_account_name')
      clearActivePlayerSession()
      return
    }

    // 先用本地标识恢复会话（gold 暂时为 0）
    set({ account: { accountId, username, gold: 0 } })
    // 异步从服务端拉取最新金币
    setTimeout(async () => {
      try {
        const info = await gameApi.getAccountInfo(accountId)
        set({ account: { accountId, username, gold: info.gold ?? 0 } })
      } catch { /* 网络失败时保持 0，下次操作会刷新 */ }
      get().loadPlayers()
    }, 0)
  },
}))

// Auto-restore on load
useAccountStore.getState().restore()
