/** 验证地图状态服务导航、过期响应保护与清理行为。 */
import { describe, expect, it, vi } from 'vitest'
import type { GameApi } from '../api/gameApi'
import { createWorldMapStateService, type WorldMapStateStore } from './stateService'
import type { WorldMapViewResponse } from './types'

/** 创建指定中心点的地图响应。 */
function response(centerX: number, centerY: number): WorldMapViewResponse {
  return { worldId: 'world_1', width: 100, height: 100, self: { playerId: 'p1', worldId: 'world_1', x: 10, y: 12, assignedBy: 'test' }, centerX, centerY, radius: 10, targets: [], serverTime: '2026-07-12T00:00:00Z' }
}

/** 创建独立的地图初始状态。 */
function store(): WorldMapStateStore { return { phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', overviewPhase: 'idle', overview: null, overviewError: '' } }

describe('世界地图状态服务', () => {
  it('读取后端真实视野并保留中心点', async () => {
    const worldMapView = vi.fn(async () => response(50, 50))
    const state = store()
    await createWorldMapStateService({ worldMapView } as unknown as GameApi, state).load('p1')
    expect(worldMapView).toHaveBeenCalledWith('p1', undefined, undefined, expect.any(AbortSignal))
    expect(state).toMatchObject({ phase: 'ready', playerId: 'p1', data: { centerX: 50, centerY: 50 } })
  })

  it('小地图使用半径一百独立读取全世界目标且不重复加载', async () => {
    const worldMapView = vi.fn(async () => response(50, 50))
    const state = store()
    const service = createWorldMapStateService({ worldMapView } as unknown as GameApi, state)
    await service.load('p1')
    await service.loadOverview('p1')
    await service.loadOverview('p1')
    expect(worldMapView).toHaveBeenLastCalledWith('p1', undefined, undefined, expect.any(AbortSignal), 100)
    expect(worldMapView).toHaveBeenCalledTimes(2)
    expect(state.overviewPhase).toBe('ready')
  })

  it('导航会限制坐标边界并返回自己的权威坐标', async () => {
    const worldMapView = vi.fn(async (_id: string, x?: number, y?: number) => response(x ?? 50, y ?? 50))
    const state = store()
    const service = createWorldMapStateService({ worldMapView } as unknown as GameApi, state)
    await service.load('p1')
    await service.navigate(-10, 999)
    expect(worldMapView).toHaveBeenLastCalledWith('p1', 0, 99, expect.any(AbortSignal))
    await service.returnHome()
    expect(worldMapView).toHaveBeenLastCalledWith('p1', 10, 12, expect.any(AbortSignal))
  })

  it('清除后在途结果不能恢复上一存档地图', async () => {
    let resolve!: (value: WorldMapViewResponse) => void
    const pending = new Promise<WorldMapViewResponse>((done) => { resolve = done })
    const state = store()
    const service = createWorldMapStateService({ worldMapView: vi.fn(() => pending) } as unknown as GameApi, state)
    const loading = service.load('p1')
    service.clear()
    resolve(response(50, 50))
    await loading
    expect(state).toMatchObject({ phase: 'idle', playerId: null, data: null })
  })
})
