import { create } from 'zustand'
import { gameApi } from '@/api/game'
import type { AccountSession, PlayerSummary } from '@/types/game'

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
      set({ account: session, loading: false })
    } catch {
      set({ loading: false })
      throw new Error('注册失败')
    }
  },

  logout: () => {
    localStorage.removeItem('hero3_account_id')
    localStorage.removeItem('hero3_account_name')
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
    if (accountId && username) {
      set({ account: { accountId, username } })
      // 延迟加载存档列表，避免循环依赖导致 gameApi 未初始化
      setTimeout(() => get().loadPlayers(), 0)
    }
  },
}))

// Auto-restore on load
useAccountStore.getState().restore()
