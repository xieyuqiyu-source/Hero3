/** 验证玩家状态服务的过期响应保护、清除和重复刷新保护。 */
import { describe, expect, it, vi } from 'vitest'
import type { GameApi } from '../api/gameApi'
import type { GameStateResponse } from './types'
import { createGameStateService, type GameStateStore } from './stateService'

/** 创建指定玩家的最小完整状态。 */
function stateFor(playerId: string): GameStateResponse {
  return { player: { id: playerId, nickname: playerId, faction: 'wei' }, resources: { items: {}, capacity: {} }, resourceProduction: {}, resourceSettledAt: '', cityGold: 0, buildings: [], resourceSlots: [], general: null, army: [], recruitQueues: [], unreadMessageCount: 0, unreadMailCount: 0, serverTime: '2026-07-12T08:00:00Z' }
}

/** 创建测试可控的延迟 Promise。 */
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}

/** 创建玩家状态服务初始状态。 */
function store(): GameStateStore { return { phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', upgradingBuildingId: null, actionMessage: '', fillingResources: false, resourceActionMessage: '', resourceActionSucceeded: false, recruitingUnitId: null, completingRecruitQueueId: null, militaryRefreshing: false, recruitActionMessage: '', recruitResultVersion: 0, recruitActionSucceeded: false, recruitActionType: null } }

describe('玩家真实状态服务', () => {
  it('后发请求覆盖前一请求且旧结果不能回写', async () => {
    const first = deferred<GameStateResponse>()
    const second = deferred<GameStateResponse>()
    const api = { gameState: vi.fn((id: string) => id === 'p1' ? first.promise : second.promise) } as unknown as GameApi
    const state = store()
    const service = createGameStateService(api, state)
    const loadingFirst = service.load('p1')
    const loadingSecond = service.load('p2')
    second.resolve(stateFor('p2'))
    await loadingSecond
    first.resolve(stateFor('p1'))
    await loadingFirst
    expect(state.data?.player.id).toBe('p2')
  })

  it('退出清除后在途结果不能恢复旧玩家数据', async () => {
    const pending = deferred<GameStateResponse>()
    const api = { gameState: vi.fn(() => pending.promise) } as unknown as GameApi
    const state = store()
    const service = createGameStateService(api, state)
    const loading = service.load('p1')
    service.clear()
    pending.resolve(stateFor('p1'))
    await loading
    expect(state).toMatchObject({ phase: 'idle', playerId: null, data: null })
  })

  it('加载期间手动刷新不会重复并发提交', async () => {
    const pending = deferred<GameStateResponse>()
    const gameState = vi.fn(() => pending.promise)
    const state = store()
    const service = createGameStateService({ gameState } as unknown as GameApi, state)
    const loading = service.load('p1')
    await service.refresh()
    expect(gameState).toHaveBeenCalledOnce()
    pending.resolve(stateFor('p1'))
    await loading
  })

  it('网络失败进入错误状态且不会保留上一份玩家数据', async () => {
    const state = store()
    state.data = stateFor('old-player')
    state.phase = 'ready'
    const service = createGameStateService({ gameState: vi.fn(async () => { throw new Error('网络连接失败') }) } as unknown as GameApi, state)
    await service.load('p1')
    expect(state).toMatchObject({ phase: 'error', playerId: 'p1', data: null, error: '网络连接失败' })
  })

  it('建造成功后合并后台返回的建筑和资源状态', async () => {
    const current = stateFor('p1')
    current.buildings = [{ id: 'wood-1', type: 'wood_camp', level: 1, upgradeEndsAt: null }]
    const upgraded = { ...current.buildings[0], upgradeEndsAt: '2026-07-12T08:01:00Z', status: 'upgrading' }
    const api = { upgradeBuilding: vi.fn(async () => ({ buildings: [upgraded], resources: { items: { wood: 10 }, capacity: { wood: 100 } }, resourceProduction: { wood: 10 }, cityGold: 0, serverTime: '2026-07-12T08:00:01Z' })) } as unknown as GameApi
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    await createGameStateService(api, state).upgradeBuilding('wood-1')
    expect(api.upgradeBuilding).toHaveBeenCalledWith('p1', 'wood-1')
    expect(state.data?.buildings[0].upgradeEndsAt).toBe('2026-07-12T08:01:00Z')
    expect(state.data?.resources.items.wood).toBe(10)
    expect(state.actionMessage).toBe('已开始建造')
  })

  it('同一时间只提交一次建造请求', async () => {
    const pending = deferred<never>()
    const upgradeBuilding = vi.fn(() => pending.promise)
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1') })
    const service = createGameStateService({ upgradeBuilding } as unknown as GameApi, state)
    void service.upgradeBuilding('wood-1')
    await service.upgradeBuilding('wood-2')
    expect(upgradeBuilding).toHaveBeenCalledOnce()
  })

  it('征兵成功后以后端结果更新资源和队列', async () => {
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1') })
    const recruit = vi.fn(async () => ({ army: [], recruitQueues: [{ id: 'q1', unitType: 'huWei', amount: 20, endsAt: '2026-07-12T08:10:00Z' }], resources: { items: { wood: 80 }, capacity: { wood: 100 } }, cityGold: 7, serverTime: '2026-07-12T08:00:01Z' }))
    await createGameStateService({ recruit } as unknown as GameApi, state).recruit('huWei', 20)
    expect(recruit).toHaveBeenCalledWith('p1', 'huWei', 20)
    expect(state.data?.recruitQueues[0].id).toBe('q1')
    expect(state.data?.resources.items.wood).toBe(80)
    expect(state).toMatchObject({ recruitActionSucceeded: true, recruitActionType: 'recruit', recruitResultVersion: 1 })
  })

  it('非法数量与重复点击都不会产生重复征兵请求', async () => {
    const pending = deferred<never>()
    const recruit = vi.fn(() => pending.promise)
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1') })
    const service = createGameStateService({ recruit } as unknown as GameApi, state)
    await service.recruit('huWei', 0)
    expect(recruit).not.toHaveBeenCalled()
    expect(state.recruitActionMessage).toContain('1 至 100000')
    void service.recruit('huWei', 10)
    await service.recruit('jinWeiSoldier', 10)
    expect(recruit).toHaveBeenCalledOnce()
  })

  it('征兵失败保留当前数据并公开可读错误', async () => {
    const current = stateFor('p1')
    current.resources.items.wood = 99
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const service = createGameStateService({ recruit: vi.fn(async () => { throw new Error('资源不足，无法执行操作') }) } as unknown as GameApi, state)
    await service.recruit('huWei', 10)
    expect(state.data?.resources.items.wood).toBe(99)
    expect(state).toMatchObject({ recruitActionSucceeded: false, recruitActionMessage: '资源不足，无法执行操作' })
  })

  it('立即完成成功后合并兵力、队列和城金', async () => {
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1') })
    const instantCompleteRecruit = vi.fn(async () => ({ army: [{ unitType: 'huWei', amount: 20 }], recruitQueues: [], resources: { items: {}, capacity: {} }, cityGold: 5, serverTime: '2026-07-12T08:00:02Z' }))
    await createGameStateService({ instantCompleteRecruit } as unknown as GameApi, state).instantCompleteRecruit('q1')
    expect(instantCompleteRecruit).toHaveBeenCalledWith('p1', 'q1')
    expect(state.data?.army[0].amount).toBe(20)
    expect(state.data?.cityGold).toBe(5)
    expect(state.recruitActionType).toBe('instant')
  })

  it('切换玩家后旧征兵响应不能污染新玩家', async () => {
    const pending = deferred<Awaited<ReturnType<GameApi['recruit']>>>()
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1') })
    const service = createGameStateService({ recruit: vi.fn(() => pending.promise), gameState: vi.fn(async () => stateFor('p2')) } as unknown as GameApi, state)
    const action = service.recruit('huWei', 1)
    await service.load('p2')
    pending.resolve({ army: [{ unitType: 'huWei', amount: 999 }], recruitQueues: [], resources: { items: {}, capacity: {} }, cityGold: 0, serverTime: '2026-07-12T08:00:03Z' })
    await action
    expect(state.data?.player.id).toBe('p2')
    expect(state.data?.army).toEqual([])
  })

  it('一键爆仓成功后以后端结果更新资源、城金和服务端时间', async () => {
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1') })
    const fillResourcesPaid = vi.fn(async () => ({ resources: { items: { wood: 100 }, capacity: { wood: 100 } }, resourceProduction: { wood: 10 }, resourceSettledAt: '2026-07-12T08:00:04Z', cityGold: 8, cost: 2, serverTime: '2026-07-12T08:00:04Z' }))
    await createGameStateService({ fillResourcesPaid } as unknown as GameApi, state).fillResourcesPaid()
    expect(fillResourcesPaid).toHaveBeenCalledWith('p1')
    expect(state.data?.resources.items.wood).toBe(100)
    expect(state.data?.cityGold).toBe(8)
    expect(state).toMatchObject({ fillingResources: false, resourceActionSucceeded: true, resourceActionMessage: '爆仓成功，消耗 2 城金' })
  })

  it('一键爆仓阻止重复提交并保留失败前资源', async () => {
    const pending = deferred<never>()
    const fillResourcesPaid = vi.fn(() => pending.promise)
    const state = store()
    const current = stateFor('p1')
    current.resources.items.wood = 12
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const service = createGameStateService({ fillResourcesPaid } as unknown as GameApi, state)
    void service.fillResourcesPaid()
    await service.fillResourcesPaid()
    expect(fillResourcesPaid).toHaveBeenCalledOnce()
    expect(state.data?.resources.items.wood).toBe(12)
  })
})
