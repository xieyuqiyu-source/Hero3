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
function store(): GameStateStore { return { phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', upgradingBuildingId: null, actionMessage: '', completingBuildingId: null, completingAllBuildings: false, buildingInstantMessage: '', buildingInstantSucceeded: false, fillingResources: false, resourceActionMessage: '', resourceActionSucceeded: false, recruitingUnitId: null, completingRecruitQueueId: null, militaryRefreshing: false, recruitActionMessage: '', recruitResultVersion: 0, recruitActionSucceeded: false, recruitActionType: null, dispatchingMarch: false, marchActionMessage: '', marchActionSucceeded: false, marchResultVersion: 0, outgoingMarches: [], outgoingMarchesLoading: false, outgoingMarchesError: '' } }

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

  it('单条建造加速成功后合并后端建筑与城金', async () => {
    const current = stateFor('p1')
    current.buildings = [{ id: 'wood-1', type: 'wood_camp', level: 1, upgradeEndsAt: '2026-07-12T08:10:00Z' }]
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const instantCompleteBuilding = vi.fn(async () => ({ buildings: [{ id: 'wood-1', type: 'wood_camp', level: 2, upgradeEndsAt: null }], resources: current.resources, resourceProduction: { wood: 20 }, cityGold: 9, serverTime: '2026-07-12T08:00:01Z' }))
    await createGameStateService({ instantCompleteBuilding } as unknown as GameApi, state).instantCompleteBuilding('wood-1')
    expect(instantCompleteBuilding).toHaveBeenCalledWith('p1', 'wood-1')
    expect(state.data?.buildings[0]).toMatchObject({ level: 2, upgradeEndsAt: null })
    expect(state.data?.cityGold).toBe(9)
  })

  it('一键完成依次处理全部建造队列且阻止重复点击', async () => {
    const current = stateFor('p1')
    current.buildings = [{ id: 'wood-1', type: 'wood_camp', level: 1, upgradeEndsAt: '2026-07-12T08:10:00Z' }, { id: 'farm-1', type: 'farm', level: 1, upgradeEndsAt: '2026-07-12T08:20:00Z' }]
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const instantCompleteBuilding = vi.fn(async (_playerId: string, buildingId: string) => ({ buildings: current.buildings.map((building) => building.id === buildingId ? { ...building, level: building.level + 1, upgradeEndsAt: null } : building), resources: current.resources, resourceProduction: {}, cityGold: 8, serverTime: '2026-07-12T08:00:02Z' }))
    await createGameStateService({ instantCompleteBuilding } as unknown as GameApi, state).instantCompleteAllBuildings()
    expect(instantCompleteBuilding).toHaveBeenCalledTimes(2)
    expect(state.buildingInstantMessage).toBe('已一键完成 2 条建造队列')
  })

  it('攻击和掠夺使用对应 marchMode 并以后端兵力回写', async () => {
    const state = store()
    const current = stateFor('p1')
    current.army = [{ unitType: 'huWei', amount: 20 }]
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const startPvpAttack = vi.fn(async (_playerId: string, _targetId: string, troops: Record<string, number>, _generalIds: string[], marchMode: string) => ({ march: { id: 'm1', marchType: marchMode, status: 'marching', attackTroops: troops, durationSeconds: 60, startedAt: '', arrivesAt: '2026-07-12T08:01:00Z' }, army: [{ unitType: 'huWei', amount: 15 }], serverTime: '2026-07-12T08:00:00Z' }))
    const service = createGameStateService({ startPvpAttack } as unknown as GameApi, state)
    await service.dispatchWorldMapCommand('attack', 'p2', { huWei: 5 }, [])
    expect(startPvpAttack).toHaveBeenCalledWith('p1', 'p2', { huWei: 5 }, [], 'attack')
    expect(state.data?.army[0].amount).toBe(15)
    expect(state.marchActionMessage).toContain('攻击队伍已出发')
    current.army = [{ unitType: 'huWei', amount: 15 }]
    await service.dispatchWorldMapCommand('plunder', 'p2', { huWei: 2 }, [])
    expect(startPvpAttack).toHaveBeenLastCalledWith('p1', 'p2', { huWei: 2 }, [], 'plunder')
  })

  it('侦查不提交前端兵量并使用后端返回的侦察兵余额', async () => {
    const state = store()
    const current = stateFor('p1')
    current.army = [{ unitType: 'zhanYingTanMa', amount: 8 }]
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const scoutPvpTarget = vi.fn(async () => ({ march: { id: 's1', marchType: 'scout', status: 'marching', attackTroops: { zhanYingTanMa: 8 }, durationSeconds: 60, startedAt: '', arrivesAt: '2026-07-12T08:01:00Z' }, army: [], serverTime: '2026-07-12T08:00:00Z' }))
    await createGameStateService({ scoutPvpTarget } as unknown as GameApi, state).dispatchWorldMapCommand('scout', 'p2', { huWei: 999 }, [])
    expect(scoutPvpTarget).toHaveBeenCalledWith('p1', 'p2')
    expect(state.data?.army).toEqual([])
    expect(state.marchActionSucceeded).toBe(true)
  })

  it('增援以后端 patch 更新兵力武将占用并阻止重复提交', async () => {
    const pending = deferred<{ reinforcement: { reinforcementId: string; status: string; troops: Record<string, number>; marchSeconds: number; sentAt: string; arriveAt: string }; patch: { army: GameStateResponse['army']; generals: []; generalAssignments: []; serverTime: string } }>()
    const state = store()
    const current = stateFor('p1')
    current.army = [{ unitType: 'huWei', amount: 20 }]
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const sendReinforcement = vi.fn(() => pending.promise)
    const service = createGameStateService({ sendReinforcement } as unknown as GameApi, state)
    const dispatching = service.dispatchWorldMapCommand('reinforce', 'p2', { huWei: 5 }, ['g1'])
    await service.dispatchWorldMapCommand('reinforce', 'p2', { huWei: 5 }, ['g1'])
    expect(sendReinforcement).toHaveBeenCalledOnce()
    pending.resolve({ reinforcement: { reinforcementId: 'r1', status: 'marching', troops: { huWei: 5 }, marchSeconds: 60, sentAt: '', arriveAt: '2026-07-12T08:01:00Z' }, patch: { army: [{ unitType: 'huWei', amount: 15 }], generals: [], generalAssignments: [], serverTime: '2026-07-12T08:00:00Z' } })
    await dispatching
    expect(state.data?.army[0].amount).toBe(15)
    expect(state.marchActionMessage).toContain('增援队伍已出发')
  })

  it('未选择兵力或切换玩家后不会错误创建和回写行军', async () => {
    const pending = deferred<Awaited<ReturnType<GameApi['startPvpAttack']>>>()
    const state = store()
    const current = stateFor('p1')
    current.army = [{ unitType: 'huWei', amount: 10 }]
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const startPvpAttack = vi.fn(() => pending.promise)
    const service = createGameStateService({ startPvpAttack, gameState: vi.fn(async () => stateFor('p2')) } as unknown as GameApi, state)
    await service.dispatchWorldMapCommand('attack', 'p2', {}, [])
    expect(startPvpAttack).not.toHaveBeenCalled()
    expect(state.marchActionMessage).toContain('至少选择')
    const dispatching = service.dispatchWorldMapCommand('attack', 'p2', { huWei: 1 }, [])
    await service.load('p2')
    pending.resolve({ march: { id: 'm1', marchType: 'attack', status: 'marching', attackTroops: { huWei: 1 }, durationSeconds: 60, startedAt: '', arrivesAt: '' }, army: [{ unitType: 'huWei', amount: 999 }], serverTime: '' })
    await dispatching
    expect(state.data?.player.id).toBe('p2')
    expect(state.data?.army).toEqual([])
  })

  it('行军业务失败保留原兵力并展示后端可读错误', async () => {
    const state = store()
    const current = stateFor('p1')
    current.army = [{ unitType: 'huWei', amount: 10 }]
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const service = createGameStateService({ startPvpAttack: vi.fn(async () => { throw new Error('目标当前处于保护状态') }) } as unknown as GameApi, state)
    await service.dispatchWorldMapCommand('attack', 'p2', { huWei: 2 }, [])
    expect(state.data?.army[0].amount).toBe(10)
    expect(state).toMatchObject({ dispatchingMarch: false, marchActionSucceeded: false, marchActionMessage: '目标当前处于保护状态' })
  })

  it('合并 PVP 与增援列表生成真实出征状态', async () => {
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1') })
    const pvpMarches = vi.fn(async () => ({ items: [{ id: 'm1', marchType: 'attack', status: 'marching', attackTroops: { huWei: 2 }, durationSeconds: 60, startedAt: '', arrivesAt: '2026-07-13T00:02:00Z', attackerPlayerId: 'p1', attackerName: '主公', defenderPlayerId: 'p2', defenderName: '目标' }] }))
    const sentReinforcements = vi.fn(async () => ({ items: [{ reinforcementId: 'r1', status: 'marching', troops: { huWei: 3 }, marchSeconds: 60, sentAt: '', arriveAt: '2026-07-13T00:01:00Z', fromPlayerId: 'p1', toPlayerId: 'p3', toPlayerName: '盟友' }] }))
    await createGameStateService({ pvpMarches, sentReinforcements } as unknown as GameApi, state).refreshOutgoingMarches()
    expect(state.outgoingMarches.map((item) => item.id)).toEqual(['r1', 'm1'])
    expect(state.outgoingMarchesLoading).toBe(false)
  })

  it('出征列表读取失败保留已有列表并公开错误', async () => {
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1'), outgoingMarches: [{ id: 'old', kind: 'attack', label: '攻击', targetName: '旧目标', troops: {}, status: 'marching', endsAt: '' }] })
    const service = createGameStateService({ pvpMarches: vi.fn(async () => { throw new Error('网络连接失败') }), sentReinforcements: vi.fn(async () => ({ items: [] })) } as unknown as GameApi, state)
    await service.refreshOutgoingMarches()
    expect(state.outgoingMarches[0].id).toBe('old')
    expect(state).toMatchObject({ outgoingMarchesLoading: false, outgoingMarchesError: '网络连接失败' })
  })
})
