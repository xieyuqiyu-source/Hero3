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
function store(): GameStateStore { return { phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', upgradingBuildingId: null, actionMessage: '', completingBuildingId: null, completingAllBuildings: false, buildingInstantMessage: '', buildingInstantSucceeded: false, fillingResources: false, resourceActionMessage: '', resourceActionSucceeded: false, capacityBoostPrices: {}, capacityBoostPricesLoading: false, capacityBoosting: false, capacityBoostMessage: '', capacityBoostSucceeded: false, recruitingUnitId: null, completingRecruitQueueId: null, militaryRefreshing: false, recruitActionMessage: '', recruitResultVersion: 0, recruitActionSucceeded: false, recruitActionType: null, dispatchingMarch: false, marchActionMessage: '', marchActionSucceeded: false, marchResultVersion: 0, outgoingMarches: [], sentReinforcements: [], receivedReinforcements: [], outgoingMarchesLoading: false, outgoingMarchesError: '', operatingMarchId: null, operatingMarchAction: null, marchOperationMessage: '', marchOperationSucceeded: false } }

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

  it('军事局部刷新同步真实守军、武将占用和服务端时间', async () => {
    const state = store()
    const current = stateFor('p1')
    current.general = { id: 'g1', name: '曹操', level: 56 }
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const militaryView = vi.fn(async () => ({ army: [{ unitType: 'huWei', amount: 7 }], recruitQueues: [], resources: current.resources, cityGold: 9, general: current.general, generals: [current.general!], generalAssignments: [{ id: 'reinforcement-r1-g1', generalId: 'g1', slot: 'reinforcement', status: 'marching' }], serverTime: '2026-07-14T08:00:00Z' }))
    await createGameStateService({ militaryView } as unknown as GameApi, state).refreshMilitary()
    expect(militaryView).toHaveBeenCalledWith('p1', expect.any(AbortSignal))
    expect(state.data).toMatchObject({ army: [{ unitType: 'huWei', amount: 7 }], cityGold: 9, serverTime: '2026-07-14T08:00:00Z' })
    expect(state.data?.generalAssignments?.[0]).toMatchObject({ generalId: 'g1', slot: 'reinforcement' })
  })

  it('军事接口明确返回空武将时清除旧的驻城武将', async () => {
    const state = store()
    const current = stateFor('p1')
    current.general = { id: 'g1', name: '旧武将', level: 1 }
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current })
    const militaryView = vi.fn(async () => ({ army: [], recruitQueues: [], resources: current.resources, cityGold: 0, general: null, generals: [], generalAssignments: [], serverTime: '2026-07-14T08:00:00Z' }))
    await createGameStateService({ militaryView } as unknown as GameApi, state).refreshMilitary()
    expect(state.data?.general).toBeNull()
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

  it('读取一次后复用后端四倍率与四时长价格表', async () => {
    const state = store()
    const capacityBoostPrices = vi.fn(async () => ({ '2x_1h': 30, '16x_24h': 9600 }))
    const service = createGameStateService({ capacityBoostPrices } as unknown as GameApi, state)
    await service.loadCapacityBoostPrices()
    await service.loadCapacityBoostPrices()
    expect(capacityBoostPrices).toHaveBeenCalledOnce()
    expect(state.capacityBoostPrices['16x_24h']).toBe(9600)
  })

  it('购买容量爆仓后回写真实容量、倍率、到期时间与城金', async () => {
    const state = store()
    const current = stateFor('p1')
    current.cityGold = 1000
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current, capacityBoostPrices: { '4x_6h': 450 } })
    const purchaseCapacityBoost = vi.fn(async () => ({ resources: { items: { wood: 10 }, capacity: { wood: 400 } }, resourceProduction: {}, resourceSettledAt: '2026-07-14T04:00:00Z', capacityBoost: 4, capacityBoostEnd: '2026-07-14T10:00:00Z', cityGold: 50, cost: 450, serverTime: '2026-07-14T04:00:00Z' }))
    await createGameStateService({ purchaseCapacityBoost } as unknown as GameApi, state).purchaseCapacityBoost(4, 6)
    expect(purchaseCapacityBoost).toHaveBeenCalledWith('p1', 4, 6)
    expect(state.data?.resources.capacity.wood).toBe(400)
    expect(state.data).toMatchObject({ capacityBoost: 4, capacityBoostEnd: '2026-07-14T10:00:00Z', cityGold: 50 })
    expect(state).toMatchObject({ capacityBoosting: false, capacityBoostSucceeded: true })
    expect(state.capacityBoostMessage).toContain('容量 ×4')
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
    const receivedReinforcements = vi.fn(async () => ({ items: [{ reinforcementId: 'rr1', status: 'marching', troops: { huWei: 4 }, marchSeconds: 60, sentAt: '', arriveAt: '2026-07-13T00:00:30Z', fromPlayerId: 'p4', fromPlayerName: '援军', toPlayerId: 'p1', toPlayerName: '主公' }] }))
    await createGameStateService({ pvpMarches, sentReinforcements, receivedReinforcements } as unknown as GameApi, state).refreshOutgoingMarches()
    expect(state.outgoingMarches.map((item) => item.id)).toEqual(['rr1', 'r1', 'm1'])
    expect(state.outgoingMarches[0]).toMatchObject({ label: '被增援', reinforcementRole: 'received' })
    expect(state.sentReinforcements.map((item) => item.reinforcementId)).toEqual(['r1'])
    expect(state.receivedReinforcements.map((item) => item.reinforcementId)).toEqual(['rr1'])
    expect(state.outgoingMarchesLoading).toBe(false)
  })

  it('出征列表读取失败保留已有列表并公开错误', async () => {
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1'), outgoingMarches: [{ id: 'old', kind: 'attack', label: '攻击', targetName: '旧目标', troops: {}, status: 'marching', endsAt: '' }] })
    const service = createGameStateService({ pvpMarches: vi.fn(async () => { throw new Error('网络连接失败') }), sentReinforcements: vi.fn(async () => ({ items: [] })), receivedReinforcements: vi.fn(async () => ({ items: [] })) } as unknown as GameApi, state)
    await service.refreshOutgoingMarches()
    expect(state.outgoingMarches[0].id).toBe('old')
    expect(state).toMatchObject({ outgoingMarchesLoading: false, outgoingMarchesError: '网络连接失败' })
  })

  it('PVP 行军加速调用真实接口、防止重复提交并回写城金后刷新队列', async () => {
    const pending = deferred<{ march: { id: string; marchType: string; status: string; attackTroops: Record<string, number>; durationSeconds: number; startedAt: string; arrivesAt: string; acceleratedTimes: number }; cityGold: number; cost: number; serverTime: string }>()
    const state = store()
    const current = stateFor('p1')
    current.cityGold = 100
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: current, outgoingMarches: [{ id: 'm1', kind: 'attack', label: '攻击', targetName: '目标', troops: { huWei: 2 }, status: 'marching', endsAt: '2026-07-13T00:02:00Z', acceleratedTimes: 0 }] })
    const acceleratePvpMarch = vi.fn(() => pending.promise)
    const pvpMarches = vi.fn(async () => ({ items: [{ id: 'm1', marchType: 'attack', status: 'marching', attackTroops: { huWei: 2 }, durationSeconds: 60, startedAt: '', arrivesAt: '2026-07-13T00:01:00Z', attackerPlayerId: 'p1', attackerName: '主公', defenderPlayerId: 'p2', defenderName: '目标', acceleratedTimes: 1 }] }))
    const service = createGameStateService({ acceleratePvpMarch, pvpMarches, sentReinforcements: vi.fn(async () => ({ items: [] })), receivedReinforcements: vi.fn(async () => ({ items: [] })) } as unknown as GameApi, state)
    const operating = service.accelerateOutgoingMarch('m1')
    void service.accelerateOutgoingMarch('m1')
    expect(acceleratePvpMarch).toHaveBeenCalledTimes(1)
    pending.resolve({ march: { id: 'm1', marchType: 'attack', status: 'marching', attackTroops: { huWei: 2 }, durationSeconds: 60, startedAt: '', arrivesAt: '2026-07-13T00:01:00Z', acceleratedTimes: 1 }, cityGold: 90, cost: 10, serverTime: '2026-07-13T00:00:10Z' })
    await operating
    expect(acceleratePvpMarch).toHaveBeenCalledWith('p1', 'm1')
    expect(state.data).toMatchObject({ cityGold: 90, serverTime: '2026-07-13T00:00:10Z' })
    expect(state.outgoingMarches[0]).toMatchObject({ id: 'm1', acceleratedTimes: 1 })
    expect(state).toMatchObject({ operatingMarchId: null, marchOperationSucceeded: true, marchOperationMessage: '行军已加速，消耗 10 城金' })
  })

  it('本人派出的增援召回后使用后端返程状态刷新右侧队列', async () => {
    const state = store()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: stateFor('p1'), outgoingMarches: [{ id: 'r1', kind: 'reinforce', label: '增援', targetName: '盟友', troops: { huWei: 3 }, status: 'marching', endsAt: '2026-07-13T00:02:00Z', reinforcementRole: 'sent', acceleratedTimes: 0 }] })
    const recallReinforcement = vi.fn(async () => ({ reinforcement: { reinforcementId: 'r1', status: 'returning', troops: { huWei: 3 }, marchSeconds: 60, sentAt: '', arriveAt: '2026-07-13T00:02:00Z', expectedReturnedAt: '2026-07-13T00:03:00Z' }, patch: { serverTime: '2026-07-13T00:00:20Z' } }))
    const sentReinforcements = vi.fn(async () => ({ items: [{ reinforcementId: 'r1', status: 'returning', troops: { huWei: 3 }, marchSeconds: 60, sentAt: '', arriveAt: '2026-07-13T00:02:00Z', expectedReturnedAt: '2026-07-13T00:03:00Z', fromPlayerId: 'p1', toPlayerId: 'p3', toPlayerName: '盟友' }] }))
    const service = createGameStateService({ recallReinforcement, pvpMarches: vi.fn(async () => ({ items: [] })), sentReinforcements, receivedReinforcements: vi.fn(async () => ({ items: [] })) } as unknown as GameApi, state)
    await service.recallOutgoingMarch('r1')
    expect(recallReinforcement).toHaveBeenCalledWith('p1', 'r1')
    expect(state.data?.serverTime).toBe('2026-07-13T00:00:20Z')
    expect(state.outgoingMarches[0]).toMatchObject({ id: 'r1', status: 'returning', endsAt: '2026-07-13T00:03:00Z' })
    expect(state.marchOperationMessage).toBe('增援已召回，正在返程')
  })
})
