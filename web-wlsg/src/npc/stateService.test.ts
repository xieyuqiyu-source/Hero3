/** NPC 状态服务测试：覆盖真实映射、扣费刷新、即时结算、重复提交和过期响应。 */
import { describe, expect, it, vi } from 'vitest'
import type { GameApi } from '../api/gameApi'
import type { GameStateStore } from '../game/stateService'
import type { GameStateResponse, NpcCityState } from '../game/types'
import { createNpcStateService, type NpcStateStore } from './stateService'

/** 创建最小真实 NPC 城池。 */
function city(id = 'npc-1'): NpcCityState {
  return { id, name: '长坂坡', faction: 'wei', tier: 'small', resources: {}, storageCapacity: {}, productionPerHour: {}, army: [], maxArmy: [], armyRecoveryRate: 0, recoveryProfile: 'normal', traits: [], resourceSettledAt: '', armySettledAt: '', generatedAt: '' }
}

/** 创建 NPC 页初始状态。 */
function npcStore(): NpcStateStore { return { phase: 'idle', playerId: null, data: null, error: '', refreshing: false, operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0 } }

/** 创建 NPC 局部回写需要的玩家状态。 */
function gameStore(): GameStateStore {
  const data: GameStateResponse = { player: { id: 'p1', nickname: '主公', faction: 'wei' }, resources: { items: {}, capacity: {} }, resourceProduction: {}, resourceSettledAt: '', cityGold: 0, buildings: [], general: null, army: [{ unitType: 'huWei', amount: 10 }, { unitType: 'zhanYingTanMa', amount: 5 }], recruitQueues: [], unreadMessageCount: 0, unreadMailCount: 0, serverTime: '' }
  return { data, receivedAt: null } as GameStateStore
}

/** 创建 NPC 动作返回的留城特性权威结算字段。 */
function npcSettlement() {
  return {
    resources: { items: { wood: 99 }, capacity: { wood: 100 } },
    resourceProduction: { wood: 12 },
    resourceSettledAt: '2026-07-14T00:59:59Z',
    generalTraitProgress: { 'caocao:weiwu_haoling:huWei': 0.5 },
  }
}

/** 创建测试可控的延迟响应。 */
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  return { promise: new Promise<T>((done, fail) => { resolve = done; reject = fail }), resolve, reject }
}

describe('NPC 状态服务', () => {
  it('加载真实列表并保留后端返回数量', async () => {
    const state = npcStore()
    const npcCities = vi.fn(async () => ({ cities: [city('a'), city('b')], lastRefreshedAt: '2026-07-14T00:00:00Z' }))
    await createNpcStateService({ npcCities } as unknown as GameApi, state, gameStore()).load('p1')
    expect(npcCities).toHaveBeenCalledWith('p1', expect.any(AbortSignal))
    expect(state).toMatchObject({ phase: 'ready', playerId: 'p1' })
    expect(state.data?.cities).toHaveLength(2)
  })

  it('付费刷新防止重复提交并返回账户金币权威余额', async () => {
    const pending = deferred<{ cities: NpcCityState[]; lastRefreshedAt: string; accountGold: number; cost: number }>()
    const state = npcStore()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: { cities: [city()], lastRefreshedAt: '' } })
    const refreshNpcCities = vi.fn(() => pending.promise)
    const service = createNpcStateService({ refreshNpcCities } as unknown as GameApi, state, gameStore())
    const refreshing = service.refresh()
    void service.refresh()
    expect(refreshNpcCities).toHaveBeenCalledOnce()
    pending.resolve({ cities: [city('new')], lastRefreshedAt: '2026-07-14T01:00:00Z', accountGold: 150, cost: 100 })
    await expect(refreshing).resolves.toBe(150)
    expect(state).toMatchObject({ refreshing: false, actionSucceeded: true, resultVersion: 1 })
    expect(state.actionMessage).toContain('100 金币')
  })

  it('攻击即时结算后局部回写兵力资源城金且不创建行军', async () => {
    const state = npcStore()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: { cities: [city()], lastRefreshedAt: '' } })
    const game = gameStore()
    const settlement = npcSettlement()
    const attackNpc = vi.fn(async () => ({ battleReport: {}, ...settlement, army: [{ unitType: 'huWei', amount: 7 }], cityGold: 8, npcState: { cities: [city()], lastRefreshedAt: '' }, serverTime: '2026-07-14T01:00:00Z' }))
    await createNpcStateService({ attackNpc } as unknown as GameApi, state, game).dispatch('attack', 'npc-1', { huWei: 3 }, [])
    expect(attackNpc).toHaveBeenCalledWith('p1', 'npc-1', 'attack', { huWei: 3 }, [])
    expect(game.data).toMatchObject({ ...settlement, army: [{ unitType: 'huWei', amount: 7 }], cityGold: 8, serverTime: '2026-07-14T01:00:00Z' })
    expect(state.actionMessage).toContain('即时结算')
  })

  it('攻击失败时不回写玩家状态也不标记成功结果', async () => {
    const state = npcStore()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: { cities: [city()], lastRefreshedAt: '' } })
    const game = gameStore()
    const before = structuredClone(game.data)
    const attackNpc = vi.fn(async () => { throw new Error('只能选择一名将领') })
    await createNpcStateService({ attackNpc } as unknown as GameApi, state, game).dispatch('attack', 'npc-1', { huWei: 3 }, ['g1'])
    expect(attackNpc).toHaveBeenCalledWith('p1', 'npc-1', 'attack', { huWei: 3 }, ['g1'])
    expect(game.data).toEqual(before)
    expect(state).toMatchObject({ operating: false, actionSucceeded: false, actionMessage: '只能选择一名将领', resultVersion: 1 })
  })

  it('NPC 与 PVP 争用失败时保留较新的行军权威状态', async () => {
    const state = npcStore()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: { cities: [city()], lastRefreshedAt: '' } })
    const game = gameStore()
    const pending = deferred<Awaited<ReturnType<GameApi['attackNpc']>>>()
    const service = createNpcStateService({ attackNpc: vi.fn(() => pending.promise) } as unknown as GameApi, state, game)

    const attacking = service.dispatch('attack', 'npc-1', { huWei: 10 }, ['caocao'])
    Object.assign(game.data!, {
      army: [{ unitType: 'huWei', amount: 25 }],
      generalAssignments: [{ id: 'pvp-caocao', generalId: 'caocao', slot: 'pvp', status: 'marching' }],
      serverTime: '2026-07-14T01:00:00Z',
    })
    pending.reject(new Error('兵力或将领已被另一支队伍占用'))
    await attacking

    expect(game.data).toMatchObject({
      army: [{ unitType: 'huWei', amount: 25 }],
      generalAssignments: [{ id: 'pvp-caocao', generalId: 'caocao', slot: 'pvp', status: 'marching' }],
      serverTime: '2026-07-14T01:00:00Z',
    })
    expect(state).toMatchObject({ operating: false, actionSucceeded: false, actionMessage: '兵力或将领已被另一支队伍占用' })
  })

  it('NPC 与增援争用失败时保留较新的增援权威状态', async () => {
    const state = npcStore()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: { cities: [city()], lastRefreshedAt: '' } })
    const game = gameStore()
    const pending = deferred<Awaited<ReturnType<GameApi['attackNpc']>>>()
    const service = createNpcStateService({ attackNpc: vi.fn(() => pending.promise) } as unknown as GameApi, state, game)

    const attacking = service.dispatch('attack', 'npc-1', { huWei: 10 }, ['caocao'])
    Object.assign(game.data!, {
      army: [{ unitType: 'huWei', amount: 25 }],
      resources: { items: { wood: 1201 }, capacity: { wood: 4800 } },
      generalTraitProgress: {},
      generalAssignments: [{ id: 'reinforcement-caocao', generalId: 'caocao', slot: 'reinforcement', status: 'marching' }],
      serverTime: '2026-07-14T01:01:00Z',
    })
    pending.reject(new Error('兵力或将领已被增援队伍占用'))
    await attacking

    expect(game.data).toMatchObject({
      army: [{ unitType: 'huWei', amount: 25 }],
      resources: { items: { wood: 1201 }, capacity: { wood: 4800 } },
      generalTraitProgress: {},
      generalAssignments: [{ id: 'reinforcement-caocao', generalId: 'caocao', slot: 'reinforcement', status: 'marching' }],
      serverTime: '2026-07-14T01:01:00Z',
    })
    expect(state).toMatchObject({ operating: false, actionSucceeded: false, actionMessage: '兵力或将领已被增援队伍占用' })
  })

  it('侦查不提交前端兵量并以后端结果扣减侦察兵', async () => {
    const state = npcStore()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: { cities: [city()], lastRefreshedAt: '' } })
    const game = gameStore()
    const settlement = npcSettlement()
    const scoutNpc = vi.fn(async () => ({ success: true, battleReport: {}, npcCity: city(), ...settlement, army: [{ unitType: 'zhanYingTanMa', amount: 3 }], npcState: { cities: [city()], lastRefreshedAt: '' }, serverTime: '2026-07-14T01:00:00Z' }))
    await createNpcStateService({ scoutNpc } as unknown as GameApi, state, game).dispatch('scout', 'npc-1', { huWei: 999 }, ['g1'])
    expect(scoutNpc).toHaveBeenCalledWith('p1', 'npc-1')
    expect(game.data?.army).toEqual([{ unitType: 'zhanYingTanMa', amount: 3 }])
    expect(game.data).toMatchObject(settlement)
  })

  it('侦查失败时不回写留城特性结算状态或 NPC 状态', async () => {
    const state = npcStore()
    Object.assign(state, { phase: 'ready', playerId: 'p1', data: { cities: [city()], lastRefreshedAt: '' } })
    const game = gameStore()
    const beforeGame = structuredClone(game.data)
    const beforeNpc = structuredClone(state.data)
    const scoutNpc = vi.fn(async () => { throw new Error('兵力不足') })
    await createNpcStateService({ scoutNpc } as unknown as GameApi, state, game).dispatch('scout', 'npc-1', {}, [])
    expect(scoutNpc).toHaveBeenCalledWith('p1', 'npc-1')
    expect(game.data).toEqual(beforeGame)
    expect(state.data).toEqual(beforeNpc)
    expect(state).toMatchObject({ operating: false, actionSucceeded: false, actionMessage: '兵力不足', resultVersion: 1 })
  })

  it('切换玩家后旧列表响应不能覆盖新玩家', async () => {
    const first = deferred<{ cities: NpcCityState[]; lastRefreshedAt: string }>()
    const state = npcStore()
    const npcCities = vi.fn((playerId: string) => playerId === 'p1' ? first.promise : Promise.resolve({ cities: [city('p2-city')], lastRefreshedAt: '' }))
    const service = createNpcStateService({ npcCities } as unknown as GameApi, state, gameStore())
    const oldLoad = service.load('p1')
    await service.load('p2')
    first.resolve({ cities: [city('p1-city')], lastRefreshedAt: '' })
    await oldLoad
    expect(state.playerId).toBe('p2')
    expect(state.data?.cities[0].id).toBe('p2-city')
  })
})
