/** 登录、会话恢复、存档选择和退出的状态服务。 */
import { ApiError } from '../api/client'
import type { GameApi } from '../api/gameApi'
import type { AccountInfo, BootstrapResponse, PlayerSummary } from '../api/types'
import type { SessionStorage } from './storage'

export type SessionPhase = 'loading' | 'login' | 'players' | 'game' | 'error'

export interface SessionState {
  phase: SessionPhase
  account: AccountInfo | null
  players: PlayerSummary[]
  currentPlayer: PlayerSummary | null
  submitting: boolean
  error: string
  bootstrap: BootstrapResponse | null
}

export interface SessionService {
  state: SessionState
  initialize: () => Promise<void>
  login: (username: string, password: string) => Promise<void>
  selectPlayer: (playerId: string) => void
  updateAccountGold: (gold: number) => void
  logout: () => void
  handleUnauthorized: () => void
}

/** 创建可注入依赖、便于自动测试的会话服务。 */
export function createSessionService(api: GameApi, storage: SessionStorage, state: SessionState): SessionService {
  /** 清空内存和本地状态并返回登录页。 */
  function logout() {
    storage.clearSession()
    storage.clearPlayerId()
    Object.assign(state, { phase: 'login', account: null, players: [], currentPlayer: null, submitting: false, error: '', bootstrap: null })
  }

  /** 加载账号存档并校验已保存的当前玩家。 */
  async function loadPlayers(account: AccountInfo, allowSavedPlayer: boolean) {
    const result = await api.players(account.accountId)
    state.players = result.players ?? []
    const savedPlayerId = allowSavedPlayer ? storage.readPlayerId() : null
    const currentPlayer = savedPlayerId ? state.players.find((player) => player.id === savedPlayerId) ?? null : null
    if (savedPlayerId && !currentPlayer) storage.clearPlayerId()
    state.currentPlayer = currentPlayer
    state.phase = currentPlayer ? 'game' : 'players'
  }

  /** 启动时验证 bootstrap 和本地会话。 */
  async function initialize() {
    Object.assign(state, { phase: 'loading', error: '' })
    try {
      state.bootstrap = await api.bootstrap()
      const saved = storage.readSession()
      if (!saved) {
        storage.clearSession()
        storage.clearPlayerId()
        state.phase = 'login'
        return
      }
      const account = await api.accountInfo(saved.accountId)
      state.account = account
      await loadPlayers(account, true)
    } catch (error) {
      if (state.phase === 'login') return
      if (error instanceof ApiError && error.status === 404) {
        logout()
        state.error = error.message
        return
      }
      state.phase = 'error'
      state.error = error instanceof Error ? error.message : '启动失败，请稍后重试'
    }
  }

  /** 使用账号密码登录并加载该账号存档。 */
  async function login(username: string, password: string) {
    if (state.submitting) return
    state.submitting = true
    state.error = ''
    try {
      const session = await api.login(username.trim(), password)
      storage.clearPlayerId()
      storage.writeSession({ accountId: session.accountId, username: session.username, token: session.token })
      state.account = { accountId: session.accountId, username: session.username, gold: session.gold }
      await loadPlayers(state.account, false)
    } catch (error) {
      state.error = error instanceof Error ? error.message : '登录失败，请稍后重试'
      state.phase = 'login'
      throw error
    } finally {
      state.submitting = false
    }
  }

  /** 只允许选择当前账号存档列表中的有效玩家。 */
  function selectPlayer(playerId: string) {
    const player = state.players.find((item) => item.id === playerId && !item.deleteScheduledAt)
    if (!player) {
      state.error = '该存档不属于当前账号或当前不可用'
      return
    }
    storage.writePlayerId(player.id)
    state.currentPlayer = player
    state.error = ''
    state.phase = 'game'
  }

  /** 使用后端写操作返回的权威余额更新当前账号金币。 */
  function updateAccountGold(gold: number) {
    if (!state.account || !Number.isFinite(gold) || gold < 0) return
    state.account.gold = Math.trunc(gold)
  }

  return { state, initialize, login, selectPlayer, updateAccountGold, logout, handleUnauthorized: logout }
}
