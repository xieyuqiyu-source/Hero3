/** 世界地图真实视野的加载、移动、重试、取消与过期响应保护。 */
import type { GameApi } from '../api/gameApi'
import type { WorldMapViewResponse } from './types'

export type WorldMapPhase = 'idle' | 'loading' | 'ready' | 'error'

export interface WorldMapStateStore {
  phase: WorldMapPhase
  playerId: string | null
  data: WorldMapViewResponse | null
  receivedAt: number | null
  error: string
  overviewPhase: WorldMapPhase
  overview: WorldMapViewResponse | null
  overviewError: string
}

export interface WorldMapStateService {
  state: WorldMapStateStore
  load: (playerId: string, centerX?: number, centerY?: number, force?: boolean) => Promise<void>
  loadOverview: (playerId: string, force?: boolean) => Promise<void>
  refresh: () => Promise<void>
  navigate: (centerX: number, centerY: number) => Promise<void>
  returnHome: () => Promise<void>
  clear: () => void
}

/** 创建以 world-map/view 为唯一事实来源的地图状态服务。 */
export function createWorldMapStateService(api: GameApi, state: WorldMapStateStore): WorldMapStateService {
  let requestVersion = 0
  let controller: AbortController | null = null
  let overviewVersion = 0
  let overviewController: AbortController | null = null

  /** 取消请求并清空上一名玩家地图，避免跨存档残留。 */
  function clear() {
    requestVersion += 1
    overviewVersion += 1
    controller?.abort()
    overviewController?.abort()
    controller = null
    overviewController = null
    Object.assign(state, { phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', overviewPhase: 'idle', overview: null, overviewError: '' })
  }

  /** 独立读取半径 100 的真实全图目标，供小地图绘制色点。 */
  async function loadOverview(playerId: string, force = false) {
    if (!force && state.overviewPhase === 'loading' && state.playerId === playerId) return
    if (!force && state.overviewPhase === 'ready' && state.playerId === playerId && state.overview) return
    overviewVersion += 1
    const currentVersion = overviewVersion
    overviewController?.abort()
    overviewController = new AbortController()
    Object.assign(state, { overviewPhase: 'loading', overviewError: '' })
    try {
      const overview = await api.worldMapView(playerId, undefined, undefined, overviewController.signal, 100)
      if (currentVersion !== overviewVersion || state.playerId !== playerId) return
      Object.assign(state, { overviewPhase: 'ready', overview, overviewError: '' })
    } catch (error) {
      if (currentVersion !== overviewVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      Object.assign(state, { overviewPhase: 'error', overviewError: error instanceof Error ? error.message : '小地图加载失败' })
    }
  }

  /** 读取指定中心点附近的真实玩家城池和黄巾营地。 */
  async function load(playerId: string, centerX?: number, centerY?: number, force = false) {
    if (!force && state.phase === 'loading' && state.playerId === playerId) return
    requestVersion += 1
    const currentVersion = requestVersion
    controller?.abort()
    controller = new AbortController()
    Object.assign(state, { phase: 'loading', playerId, error: '' })
    try {
      const data = await api.worldMapView(playerId, centerX, centerY, controller.signal)
      if (currentVersion !== requestVersion) return
      Object.assign(state, { phase: 'ready', data, receivedAt: Date.now(), error: '' })
    } catch (error) {
      if (currentVersion !== requestVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      Object.assign(state, { phase: 'error', error: error instanceof Error ? error.message : '世界地图加载失败' })
    }
  }

  /** 重新读取当前中心点。 */
  async function refresh() {
    if (!state.playerId || state.phase === 'loading') return
    await load(state.playerId, state.data?.centerX, state.data?.centerY, true)
  }

  /** 将目标中心限制在后端公布的世界边界后读取。 */
  async function navigate(centerX: number, centerY: number) {
    if (!state.playerId || state.phase === 'loading') return
    const width = state.data?.width ?? 100
    const height = state.data?.height ?? 100
    const safeX = Math.max(0, Math.min(width - 1, Math.trunc(centerX)))
    const safeY = Math.max(0, Math.min(height - 1, Math.trunc(centerY)))
    await load(state.playerId, safeX, safeY, true)
  }

  /** 返回后端分配给当前存档的权威城池坐标。 */
  async function returnHome() {
    if (!state.data) return
    await navigate(state.data.self.x, state.data.self.y)
  }

  return { state, load, loadOverview, refresh, navigate, returnHome, clear }
}
