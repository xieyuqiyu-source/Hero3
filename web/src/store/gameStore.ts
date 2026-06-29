import { create } from 'zustand'
import { gameApi } from '@/api/game'
import type { CityActionResult, GameState, GeneralViewActionResult, MilitaryActionResult, ResourceActionResult } from '@/types/game'

interface GameStore {
  /** 当前活跃玩家 ID */
  activePlayerId: string | null
  /** 后端返回的权威游戏状态 */
  state: GameState | null
  /** 当前 state 到达前端的本地时间，用于资源增长预测 */
  stateReceivedAt: number | null
  /** 是否正在加载 */
  loading: boolean
  /** 错误信息 */
  error: string | null

  /** 设置完整游戏状态 */
  setState: (state: GameState) => void
  /** 部分更新（用于动作接口返回后局部刷新） */
  patchState: (patch: Partial<GameState>) => void
  /** 合并城池操作结果 */
  patchCityAction: (result: CityActionResult) => void
  /** 合并资源操作结果 */
  patchResourceAction: (result: ResourceActionResult) => void
  /** 合并军事操作结果 */
  patchMilitaryAction: (result: MilitaryActionResult) => void
  /** 合并武将操作结果 */
  patchGeneralAction: (result: GeneralViewActionResult) => void
  /** 设置 loading */
  setLoading: (loading: boolean) => void
  /** 设置错误 */
  setError: (error: string | null) => void
  /** 设置当前活跃玩家并持久化 */
  setActivePlayer: (playerId: string) => void
  /** 清除活跃玩家（退出存档） */
  clearActivePlayer: () => void
  /** 从后端加载完整游戏状态 */
  loadGameState: (playerId?: string) => Promise<void>
  /** 从后端加载玩家摘要视图 */
  loadSummaryView: (playerId?: string) => Promise<void>
  /** 从后端加载城池视图 */
  loadCityView: (playerId?: string) => Promise<void>
  /** 从后端加载资源视图 */
  loadResourceView: (playerId?: string) => Promise<void>
  /** 从后端加载军事视图 */
  loadMilitaryView: (playerId?: string) => Promise<void>
  /** 从后端加载武将视图 */
  loadGeneralsView: (playerId?: string) => Promise<void>
  /** 从后端加载背包视图 */
  loadInventoryView: (playerId?: string) => Promise<void>
  /** 升级建筑 */
  upgradeBuilding: (buildingId: string) => Promise<void>
  /** 将领四维加点 */
  allocateGeneralStat: (statKey: string, amount?: number) => Promise<void>
  /** 将领洗点 */
  resetGeneralStats: () => Promise<number | undefined>
  /** 更换将领 */
  changeGeneral: (generalId: string, itemId?: string) => Promise<number | undefined>
}

export const useGameStore = create<GameStore>((set, get) => ({
  activePlayerId: localStorage.getItem('hero3_active_player_id'),
  state: null,
  stateReceivedAt: null,
  loading: false,
  error: null,

  setState: (state) => set({ state, stateReceivedAt: Date.now(), error: null }),
  patchState: (patch) =>
    set((prev) => {
      const cleanPatch = Object.fromEntries(Object.entries(patch).filter(([, value]) => value !== undefined)) as Partial<GameState>
      return {
        state: prev.state ? { ...prev.state, ...cleanPatch } : null,
        stateReceivedAt: Date.now(),
      }
    }),
  patchCityAction: (result) =>
    get().patchState({
      buildings: result.buildings ?? get().state?.buildings,
      resourceSlots: result.resourceSlots ?? get().state?.resourceSlots,
      resources: result.resources,
      resourceProduction: result.resourceProduction,
      cityGold: result.cityGold,
      activeModifiers: result.activeModifiers ?? get().state?.activeModifiers,
      serverTime: result.serverTime,
    }),
  patchResourceAction: (result) =>
    get().patchState({
      resources: result.resources,
      resourceProduction: result.resourceProduction,
      resourceSettledAt: result.resourceSettledAt,
      productionBoost: result.productionBoost,
      productionBoostEnd: result.productionBoostEnd,
      capacityBoost: result.capacityBoost,
      capacityBoostEnd: result.capacityBoostEnd,
      activeModifiers: result.activeModifiers ?? get().state?.activeModifiers,
      cityGold: result.cityGold,
      serverTime: result.serverTime,
    }),
  patchMilitaryAction: (result) =>
    get().patchState({
      army: result.army,
      recruitQueues: result.recruitQueues,
      resources: result.resources,
      cityGold: result.cityGold,
      serverTime: result.serverTime,
    }),
  patchGeneralAction: (result) =>
    get().patchState({
      general: result.general,
      generals: result.generals ?? get().state?.generals,
      generalAssignments: result.generalAssignments ?? get().state?.generalAssignments,
      generalChangeUntil: result.generalChangeUntil ?? get().state?.generalChangeUntil,
      activeModifiers: result.activeModifiers ?? get().state?.activeModifiers,
      serverTime: result.serverTime,
    }),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error, loading: false }),
  setActivePlayer: (playerId) => {
    localStorage.setItem('hero3_active_player_id', playerId)
    set({ activePlayerId: playerId })
  },
  clearActivePlayer: () => {
    localStorage.removeItem('hero3_active_player_id')
    set({ activePlayerId: null, state: null, stateReceivedAt: null })
  },
  loadGameState: async (playerId?: string) => {
    const id = playerId ?? get().activePlayerId
    if (!id) return
    set({ loading: true, error: null })
    try {
      const state = await gameApi.getState(id)
      set({ state, stateReceivedAt: Date.now(), loading: false, error: null })
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载游戏状态失败'
      set({ error: message, loading: false })
    }
  },
  loadSummaryView: async (playerId?: string) => {
    const id = playerId ?? get().activePlayerId
    if (!id) return
    try {
      const view = await gameApi.getSummaryView(id)
      get().patchState(view)
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载玩家摘要失败'
      set({ error: message })
    }
  },
  loadCityView: async (playerId?: string) => {
    const id = playerId ?? get().activePlayerId
    if (!id) return
    try {
      const view = await gameApi.getCityView(id)
      get().patchState(view)
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载城池视图失败'
      set({ error: message })
    }
  },
  loadResourceView: async (playerId?: string) => {
    const id = playerId ?? get().activePlayerId
    if (!id) return
    try {
      const view = await gameApi.getResourceView(id)
      get().patchState(view)
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载资源视图失败'
      set({ error: message })
    }
  },
  loadMilitaryView: async (playerId?: string) => {
    const id = playerId ?? get().activePlayerId
    if (!id) return
    try {
      const view = await gameApi.getMilitaryView(id)
      get().patchState(view)
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载军事视图失败'
      set({ error: message })
    }
  },
  loadGeneralsView: async (playerId?: string) => {
    const id = playerId ?? get().activePlayerId
    if (!id) return
    try {
      const view = await gameApi.getGeneralsView(id)
      get().patchState(view)
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载武将视图失败'
      set({ error: message })
    }
  },
  loadInventoryView: async (playerId?: string) => {
    const id = playerId ?? get().activePlayerId
    if (!id) return
    try {
      const view = await gameApi.getInventoryView(id)
      get().patchState(view)
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载背包视图失败'
      set({ error: message })
    }
  },
  upgradeBuilding: async (buildingId: string) => {
    const playerId = get().activePlayerId
    if (!playerId) return
    const result = await gameApi.upgradeBuilding(playerId, buildingId)
    get().patchCityAction(result)
    set({ error: null })
  },
  allocateGeneralStat: async (statKey: string, amount = 1) => {
    const playerId = get().activePlayerId
    if (!playerId) return
    const result = await gameApi.allocateGeneralStat(playerId, statKey, amount)
    get().patchGeneralAction(result)
    set({ error: null })
  },
  resetGeneralStats: async () => {
    const playerId = get().activePlayerId
    if (!playerId) return undefined
    const result = await gameApi.resetGeneralStats(playerId)
    get().patchGeneralAction(result)
    set({ error: null })
    return result.accountGold
  },
  changeGeneral: async (generalId: string, itemId?: string) => {
    const playerId = get().activePlayerId
    if (!playerId) return undefined
    const result = await gameApi.changeGeneral(playerId, generalId, itemId)
    get().patchGeneralAction(result)
    set({ error: null })
    return result.accountGold
  },
}))

window.addEventListener('hero3:clear-active-player', () => {
  useGameStore.getState().clearActivePlayer()
})
