/** NPC 页面状态服务：负责真实列表、付费刷新和即时战斗结果的过期保护与局部回写。 */
import type { GameApi } from '../api/gameApi'
import type { GameStateStore } from '../game/stateService'
import type { NpcCommandAction, NpcStateResponse } from '../game/types'

export type NpcPhase = 'idle' | 'loading' | 'ready' | 'error'

export interface NpcStateStore {
  phase: NpcPhase
  playerId: string | null
  data: NpcStateResponse | null
  error: string
  refreshing: boolean
  operating: boolean
  actionMessage: string
  actionSucceeded: boolean
  resultVersion: number
}

export interface NpcStateService {
  state: NpcStateStore
  load: (playerId: string, force?: boolean) => Promise<void>
  refresh: () => Promise<number | null>
  dispatch: (action: NpcCommandAction, npcId: string, troops: Record<string, number>, generalIds: string[]) => Promise<void>
  clear: () => void
}

/** 创建可取消旧查询并拒绝重复写操作的 NPC 页面服务。 */
export function createNpcStateService(api: GameApi, state: NpcStateStore, game: GameStateStore): NpcStateService {
  let requestVersion = 0
  let controller: AbortController | null = null

  /** 退出或切换存档时取消旧请求并清空 NPC 数据。 */
  function clear() {
    requestVersion += 1
    controller?.abort()
    controller = null
    Object.assign(state, { phase: 'idle', playerId: null, data: null, error: '', refreshing: false, operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0 })
  }

  /** 读取当前账号已验证存档的 NPC 状态。 */
  async function load(playerId: string, force = false) {
    if (!playerId || (!force && state.playerId === playerId && (state.phase === 'loading' || state.phase === 'ready'))) return
    const playerChanged = state.playerId !== playerId
    requestVersion += 1
    const currentVersion = requestVersion
    controller?.abort()
    controller = new AbortController()
    Object.assign(state, { phase: 'loading', playerId, data: playerChanged ? null : state.data, error: '', actionMessage: '' })
    try {
      const data = await api.npcCities(playerId, controller.signal)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      Object.assign(state, { phase: 'ready', data: { cities: data.cities ?? [], lastRefreshedAt: data.lastRefreshedAt ?? '', refreshCost: data.refreshCost }, error: '' })
    } catch (error) {
      if (currentVersion !== requestVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      Object.assign(state, { phase: 'error', data: null, error: error instanceof Error ? error.message : 'NPC 城池加载失败' })
    }
  }

  /** 调用付费刷新接口并返回后端权威账户金币余额。 */
  async function refresh(): Promise<number | null> {
    if (!state.playerId || state.refreshing || state.operating) return null
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.refreshing = true
    state.actionMessage = ''
    try {
      const result = await api.refreshNpcCities(playerId)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return null
      state.data = { cities: result.cities ?? [], lastRefreshedAt: result.lastRefreshedAt ?? '', refreshCost: result.refreshCost ?? result.cost }
      state.phase = 'ready'
      state.actionMessage = `NPC 城池已刷新，消耗 ${result.cost} 金币`
      state.actionSucceeded = true
      return result.accountGold
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return null
      state.actionMessage = error instanceof Error ? error.message : '刷新 NPC 城池失败'
      state.actionSucceeded = false
      return null
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) {
        state.refreshing = false
        state.resultVersion += 1
      }
    }
  }

  /** 合并 NPC 战斗接口返回的真实玩家局部状态。 */
  function patchGame(result: { army: NonNullable<GameStateStore['data']>['army']; npcState?: NpcStateResponse; serverTime: string; resources: NonNullable<GameStateStore['data']>['resources']; resourceProduction: NonNullable<GameStateStore['data']>['resourceProduction']; resourceSettledAt: string; generalTraitProgress: Record<string, number>; cityGold?: number; general?: NonNullable<GameStateStore['data']>['general']; generals?: NonNullable<GameStateStore['data']>['generals'] }) {
    if (!game.data) return
    Object.assign(game.data, {
      army: result.army ?? game.data.army,
      resources: result.resources,
      resourceProduction: result.resourceProduction,
      resourceSettledAt: result.resourceSettledAt,
      generalTraitProgress: result.generalTraitProgress,
      cityGold: result.cityGold ?? game.data.cityGold,
      general: result.general === undefined ? game.data.general : result.general,
      generals: result.generals ?? game.data.generals,
      serverTime: result.serverTime || game.data.serverTime,
    })
    if (result.npcState) state.data = result.npcState
    game.receivedAt = Date.now()
  }

  /** 即时结算 NPC 进攻、掠夺或侦查，不创建行军队列。 */
  async function dispatch(action: NpcCommandAction, npcId: string, troops: Record<string, number>, generalIds: string[]) {
    if (!state.playerId || !game.data || state.operating || state.refreshing || !state.data?.cities.some((city) => city.id === npcId)) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    const available = game.data.army.reduce<Record<string, number>>((result, item) => {
      result[item.unitType] = (result[item.unitType] ?? 0) + item.amount
      return result
    }, {})
    const normalizedTroops = Object.fromEntries(Object.entries(troops).filter(([unitType, amount]) => unitType && Number.isInteger(amount) && amount > 0 && amount <= (available[unitType] ?? 0)))
    if (action !== 'scout' && !Object.keys(normalizedTroops).length) {
      state.actionMessage = '请至少选择一个出征兵种'
      state.actionSucceeded = false
      state.resultVersion += 1
      return
    }
    const selectedGenerals = [...new Set(generalIds.map((id) => id.trim()).filter(Boolean))].slice(0, 1)
    state.operating = true
    state.actionMessage = ''
    try {
      if (action === 'scout') {
        const result = await api.scoutNpc(playerId, npcId)
        if (currentVersion !== requestVersion || state.playerId !== playerId) return
        patchGame(result)
        state.actionMessage = result.success ? '侦查已即时结算并成功获取情报，可前往军情查看战报' : '侦查已即时结算，但未能获取守军情报'
      } else {
        const result = await api.attackNpc(playerId, npcId, action, normalizedTroops, selectedGenerals)
        if (currentVersion !== requestVersion || state.playerId !== playerId) return
        patchGame(result)
        state.actionMessage = `${action === 'attack' ? '进攻' : '掠夺'}已即时结算，可前往军情查看战报`
      }
      state.actionSucceeded = true
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.actionMessage = error instanceof Error ? error.message : 'NPC 命令提交失败'
      state.actionSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) {
        state.operating = false
        state.resultVersion += 1
      }
    }
  }

  return { state, load, refresh, dispatch, clear }
}
