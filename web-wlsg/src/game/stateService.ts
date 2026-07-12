/** 当前玩家完整状态的加载、重试、取消和过期结果保护服务。 */
import type { GameApi } from '../api/gameApi'
import { toOutgoingMarches } from './marchAdapter'
import type { GameStateResponse, OutgoingMarchViewModel, WorldMapMarchAction } from './types'

export type GameStatePhase = 'idle' | 'loading' | 'ready' | 'error'

export interface GameStateStore {
  phase: GameStatePhase
  playerId: string | null
  data: GameStateResponse | null
  receivedAt: number | null
  error: string
  upgradingBuildingId: string | null
  actionMessage: string
  completingBuildingId: string | null
  completingAllBuildings: boolean
  buildingInstantMessage: string
  buildingInstantSucceeded: boolean
  fillingResources: boolean
  resourceActionMessage: string
  resourceActionSucceeded: boolean
  recruitingUnitId: string | null
  completingRecruitQueueId: string | null
  militaryRefreshing: boolean
  recruitActionMessage: string
  recruitResultVersion: number
  recruitActionSucceeded: boolean
  recruitActionType: 'recruit' | 'instant' | null
  dispatchingMarch: boolean
  marchActionMessage: string
  marchActionSucceeded: boolean
  marchResultVersion: number
  outgoingMarches: OutgoingMarchViewModel[]
  outgoingMarchesLoading: boolean
  outgoingMarchesError: string
}

export interface GameStateService {
  state: GameStateStore
  load: (playerId: string, force?: boolean) => Promise<void>
  refresh: () => Promise<void>
  upgradeBuilding: (buildingId: string) => Promise<void>
  instantCompleteBuilding: (buildingId: string) => Promise<void>
  instantCompleteAllBuildings: () => Promise<void>
  fillResourcesPaid: () => Promise<void>
  refreshMilitary: () => Promise<void>
  recruit: (unitId: string, amount: number) => Promise<void>
  instantCompleteRecruit: (queueId: string) => Promise<void>
  dispatchWorldMapCommand: (action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]) => Promise<void>
  refreshOutgoingMarches: () => Promise<void>
  clear: () => void
}

/** 创建能够取消旧请求并忽略过期响应的玩家状态服务。 */
export function createGameStateService(api: GameApi, state: GameStateStore): GameStateService {
  let requestVersion = 0
  let controller: AbortController | null = null
  let militaryRequestVersion = 0
  let militaryController: AbortController | null = null
  let marchesRequestVersion = 0
  let marchesController: AbortController | null = null

  /** 将后端军事局部响应合并到完整玩家状态。 */
  function patchMilitary(result: { army: GameStateResponse['army']; recruitQueues: GameStateResponse['recruitQueues']; resources: GameStateResponse['resources']; cityGold: number; serverTime: string }) {
    if (!state.data) return
    Object.assign(state.data, { army: result.army ?? [], recruitQueues: result.recruitQueues ?? [], resources: result.resources, cityGold: result.cityGold, serverTime: result.serverTime })
    state.receivedAt = Date.now()
  }

  /** 立即清除当前玩家数据并取消在途请求。 */
  function clear() {
    requestVersion += 1
    militaryRequestVersion += 1
    marchesRequestVersion += 1
    controller?.abort()
    militaryController?.abort()
    marchesController?.abort()
    controller = null
    militaryController = null
    marchesController = null
    Object.assign(state, { phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', upgradingBuildingId: null, actionMessage: '', completingBuildingId: null, completingAllBuildings: false, buildingInstantMessage: '', buildingInstantSucceeded: false, fillingResources: false, resourceActionMessage: '', resourceActionSucceeded: false, recruitingUnitId: null, completingRecruitQueueId: null, militaryRefreshing: false, recruitActionMessage: '', recruitResultVersion: 0, recruitActionSucceeded: false, recruitActionType: null, dispatchingMarch: false, marchActionMessage: '', marchActionSucceeded: false, marchResultVersion: 0, outgoingMarches: [], outgoingMarchesLoading: false, outgoingMarchesError: '' })
  }

  /** 加载指定且已由会话层校验的玩家状态。 */
  async function load(playerId: string, force = false) {
    if (!force && state.phase === 'loading' && state.playerId === playerId) return
    requestVersion += 1
    const currentVersion = requestVersion
    controller?.abort()
    controller = new AbortController()
    Object.assign(state, { phase: 'loading', playerId, data: null, receivedAt: null, error: '', actionMessage: '', completingBuildingId: null, completingAllBuildings: false, buildingInstantMessage: '', buildingInstantSucceeded: false, fillingResources: false, resourceActionMessage: '', resourceActionSucceeded: false, recruitingUnitId: null, completingRecruitQueueId: null, militaryRefreshing: false, recruitActionMessage: '', recruitActionSucceeded: false, recruitActionType: null, dispatchingMarch: false, marchActionMessage: '', marchActionSucceeded: false, outgoingMarches: [], outgoingMarchesLoading: false, outgoingMarchesError: '' })
    try {
      const data = await api.gameState(playerId, controller.signal)
      if (currentVersion !== requestVersion) return
      Object.assign(state, { phase: 'ready', data, receivedAt: Date.now(), error: '' })
      void refreshOutgoingMarches()
    } catch (error) {
      if (currentVersion !== requestVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      Object.assign(state, { phase: 'error', data: null, receivedAt: null, error: error instanceof Error ? error.message : '玩家状态加载失败' })
    }
  }

  /** 手动刷新当前玩家状态，加载中不重复提交。 */
  async function refresh() {
    if (!state.playerId || state.phase === 'loading') return
    await load(state.playerId, true)
  }

  /** 读取军事局部视图，用于队列到期结算和页内重试。 */
  async function refreshMilitary() {
    if (!state.playerId || !state.data || state.militaryRefreshing) return
    militaryRequestVersion += 1
    const currentVersion = militaryRequestVersion
    const playerId = state.playerId
    militaryController?.abort()
    militaryController = new AbortController()
    state.militaryRefreshing = true
    state.recruitActionMessage = ''
    try {
      const result = await api.militaryView(playerId, militaryController.signal)
      if (currentVersion !== militaryRequestVersion || state.playerId !== playerId) return
      patchMilitary(result)
    } catch (error) {
      if (currentVersion !== militaryRequestVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      state.recruitActionMessage = error instanceof Error ? error.message : '军事数据刷新失败'
    } finally {
      if (currentVersion === militaryRequestVersion) state.militaryRefreshing = false
    }
  }

  /** 并行读取 PVP 与增援活动行军，统一映射到右侧出征状态。 */
  async function refreshOutgoingMarches() {
    if (!state.playerId || !state.data || state.outgoingMarchesLoading) return
    marchesRequestVersion += 1
    const currentVersion = marchesRequestVersion
    const playerId = state.playerId
    marchesController?.abort()
    marchesController = new AbortController()
    state.outgoingMarchesLoading = true
    state.outgoingMarchesError = ''
    try {
      const [pvp, reinforcements] = await Promise.all([api.pvpMarches(playerId, marchesController.signal), api.sentReinforcements(playerId, marchesController.signal)])
      if (currentVersion !== marchesRequestVersion || state.playerId !== playerId) return
      state.outgoingMarches = toOutgoingMarches(playerId, pvp.items ?? [], reinforcements.items ?? [])
    } catch (error) {
      if (currentVersion !== marchesRequestVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      state.outgoingMarchesError = error instanceof Error ? error.message : '出征状态加载失败'
    } finally {
      if (currentVersion === marchesRequestVersion) state.outgoingMarchesLoading = false
    }
  }

  /** 提交真实征兵操作并合并后端返回的队列和资源。 */
  async function recruit(unitId: string, amount: number) {
    if (!state.playerId || !state.data || state.phase !== 'ready' || state.recruitingUnitId || state.completingRecruitQueueId) return
    if (!Number.isInteger(amount) || amount < 1 || amount > 100000) {
      Object.assign(state, { recruitActionMessage: '征兵数量必须是 1 至 100000 的整数', recruitActionSucceeded: false, recruitActionType: 'recruit', recruitResultVersion: state.recruitResultVersion + 1 })
      return
    }
    const playerId = state.playerId
    const currentVersion = requestVersion
    militaryRequestVersion += 1
    militaryController?.abort()
    state.militaryRefreshing = false
    state.recruitingUnitId = unitId
    state.recruitActionMessage = ''
    try {
      const result = await api.recruit(playerId, unitId, amount)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      patchMilitary(result)
      state.recruitActionMessage = '已加入征兵队列'
      state.recruitActionSucceeded = true
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.recruitActionMessage = error instanceof Error ? error.message : '征兵失败，请稍后重试'
      state.recruitActionSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) {
        state.recruitingUnitId = null
        state.recruitActionType = 'recruit'
        state.recruitResultVersion += 1
      }
    }
  }

  /** 调用现有城金接口立即完成征兵队列。 */
  async function instantCompleteRecruit(queueId: string) {
    if (!state.playerId || !state.data || state.phase !== 'ready' || state.completingRecruitQueueId || state.recruitingUnitId) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    militaryRequestVersion += 1
    militaryController?.abort()
    state.militaryRefreshing = false
    state.completingRecruitQueueId = queueId
    state.recruitActionMessage = ''
    try {
      const result = await api.instantCompleteRecruit(playerId, queueId)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      patchMilitary(result)
      state.recruitActionMessage = '征兵队列已立即完成'
      state.recruitActionSucceeded = true
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.recruitActionMessage = error instanceof Error ? error.message : '立即完成失败'
      state.recruitActionSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) {
        state.completingRecruitQueueId = null
        state.recruitActionType = 'instant'
        state.recruitResultVersion += 1
      }
    }
  }

  /** 调用现有 PVP 或增援接口派兵，并以后端返回兵力与武将占用为准回写。 */
  async function dispatchWorldMapCommand(action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]) {
    if (!state.playerId || !state.data || state.phase !== 'ready' || state.dispatchingMarch) return
    const target = targetPlayerId.trim()
    if (!target) return
    const selectedGenerals = [...new Set(generalIds.map((id) => id.trim()).filter(Boolean))].slice(0, 1)
    const available = state.data.army.reduce<Record<string, number>>((result, item) => {
      result[item.unitType] = (result[item.unitType] ?? 0) + item.amount
      return result
    }, {})
    const normalizedTroops = Object.fromEntries(Object.entries(troops).filter(([unitType, amount]) => unitType && Number.isInteger(amount) && amount > 0 && amount <= (available[unitType] ?? 0)))
    if (action !== 'scout' && Object.keys(normalizedTroops).length === 0) {
      Object.assign(state, { marchActionMessage: '请至少选择一个出征兵种', marchActionSucceeded: false, marchResultVersion: state.marchResultVersion + 1 })
      return
    }
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.dispatchingMarch = true
    state.marchActionMessage = ''
    try {
      const result = action === 'scout'
        ? await api.scoutPvpTarget(playerId, target)
        : action === 'reinforce'
          ? await api.sendReinforcement(playerId, target, normalizedTroops, selectedGenerals)
          : await api.startPvpAttack(playerId, target, normalizedTroops, selectedGenerals, action)
      if (currentVersion !== requestVersion || state.playerId !== playerId || !state.data) return
      if ('march' in result) {
        Object.assign(state.data, { army: result.army, generals: result.generals ?? state.data.generals, generalAssignments: result.generalAssignments ?? state.data.generalAssignments, serverTime: result.serverTime })
        state.marchActionMessage = `${{ attack: '攻击', plunder: '掠夺', scout: '侦查', reinforce: '增援' }[action]}队伍已出发，预计 ${result.march.arrivesAt} 到达`
      } else {
        const patch = result.patch
        Object.assign(state.data, { army: patch?.army ?? state.data.army, generals: patch?.generals ?? state.data.generals, generalAssignments: patch?.generalAssignments ?? state.data.generalAssignments, serverTime: patch?.serverTime ?? state.data.serverTime })
        state.marchActionMessage = `增援队伍已出发${result.reinforcement.arriveAt ? `，预计 ${result.reinforcement.arriveAt} 到达` : ''}`
      }
      state.receivedAt = Date.now()
      state.marchActionSucceeded = true
      void refreshOutgoingMarches()
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.marchActionMessage = error instanceof Error ? error.message : '行军命令提交失败'
      state.marchActionSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) {
        state.dispatchingMarch = false
        state.marchResultVersion += 1
      }
    }
  }

  /** 调用建造接口并将返回的城池局部状态合并到当前页。 */
  async function upgradeBuilding(buildingId: string) {
    if (!state.playerId || !state.data || state.phase !== 'ready' || state.upgradingBuildingId) return
    state.upgradingBuildingId = buildingId
    state.actionMessage = ''
    try {
      const result = await api.upgradeBuilding(state.playerId, buildingId)
      if (!state.data) return
      Object.assign(state.data, {
        buildings: result.buildings,
        resourceSlots: result.resourceSlots ?? state.data.resourceSlots,
        resources: result.resources,
        resourceProduction: result.resourceProduction,
        cityGold: result.cityGold,
        serverTime: result.serverTime,
      })
      state.receivedAt = Date.now()
      state.actionMessage = '已开始建造'
    } catch (error) {
      state.actionMessage = error instanceof Error ? error.message : '建造失败，请稍后重试'
    } finally {
      state.upgradingBuildingId = null
    }
  }

  /** 调用现有付费爆仓接口并以后端返回资源、城金为准更新页面。 */
  async function fillResourcesPaid() {
    if (!state.playerId || !state.data || state.phase !== 'ready' || state.fillingResources) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.fillingResources = true
    state.resourceActionMessage = ''
    try {
      const result = await api.fillResourcesPaid(playerId)
      if (currentVersion !== requestVersion || state.playerId !== playerId || !state.data) return
      Object.assign(state.data, { resources: result.resources, resourceProduction: result.resourceProduction, resourceSettledAt: result.resourceSettledAt, cityGold: result.cityGold, serverTime: result.serverTime })
      state.receivedAt = Date.now()
      state.resourceActionMessage = result.cost ? `爆仓成功，消耗 ${result.cost} 城金` : '资源已经满仓'
      state.resourceActionSucceeded = true
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.resourceActionMessage = error instanceof Error ? error.message : '一键爆仓失败'
      state.resourceActionSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) state.fillingResources = false
    }
  }

  /** 将建筑加速接口返回的城池事实合并到完整玩家状态。 */
  function patchCityAction(result: Awaited<ReturnType<GameApi['instantCompleteBuilding']>>) {
    if (!state.data) return
    Object.assign(state.data, { buildings: result.buildings, resourceSlots: result.resourceSlots ?? state.data.resourceSlots, resources: result.resources, resourceProduction: result.resourceProduction, cityGold: result.cityGold, serverTime: result.serverTime })
    state.receivedAt = Date.now()
  }

  /** 直接使用城金完成指定建造队列。 */
  async function instantCompleteBuilding(buildingId: string) {
    if (!state.playerId || !state.data || state.phase !== 'ready' || state.completingBuildingId || state.completingAllBuildings) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.completingBuildingId = buildingId
    state.buildingInstantMessage = ''
    try {
      const result = await api.instantCompleteBuilding(playerId, buildingId)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      patchCityAction(result)
      state.buildingInstantMessage = '建造队列已立即完成'
      state.buildingInstantSucceeded = true
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.buildingInstantMessage = error instanceof Error ? error.message : '立即完成建造失败'
      state.buildingInstantSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) state.completingBuildingId = null
    }
  }

  /** 依次调用现有单队列接口完成当前全部建造队列。 */
  async function instantCompleteAllBuildings() {
    if (!state.playerId || !state.data || state.phase !== 'ready' || state.completingBuildingId || state.completingAllBuildings) return
    const ids = state.data.buildings.filter((building) => building.upgradeEndsAt).map((building) => building.id)
    if (!ids.length) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    let completed = 0
    state.completingAllBuildings = true
    state.buildingInstantMessage = ''
    try {
      for (const buildingId of ids) {
        const result = await api.instantCompleteBuilding(playerId, buildingId)
        if (currentVersion !== requestVersion || state.playerId !== playerId) return
        patchCityAction(result)
        completed += 1
      }
      state.buildingInstantMessage = `已一键完成 ${completed} 条建造队列`
      state.buildingInstantSucceeded = true
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      const message = error instanceof Error ? error.message : '一键完成建造失败'
      state.buildingInstantMessage = completed ? `已完成 ${completed} 条，随后失败：${message}` : message
      state.buildingInstantSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) state.completingAllBuildings = false
    }
  }

  return { state, load, refresh, refreshMilitary, refreshOutgoingMarches, recruit, instantCompleteRecruit, dispatchWorldMapCommand, upgradeBuilding, fillResourcesPaid, instantCompleteBuilding, instantCompleteAllBuildings, clear }
}
