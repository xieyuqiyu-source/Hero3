/** 验证轮回绝境加载、独立波次结算、金币回写和切档隔离。 */
import { describe, expect, it, vi } from 'vitest'
import type { GameApi } from '../api/gameApi'
import type { GameStateStore } from '../game/stateService'
import type { DungeonConfig, DungeonRun } from './types'
import { createDungeonStateService, type DungeonStateStore } from './stateService'

const config: DungeonConfig = { levels: [{ level: 1, name: '入门轮回', wavePowerBase: 100, playerTroopCap: 100, enemyTroopBase: 100, durationSeconds: 7200, rewardExpCap: 1000, enabled: true }], waves: [], enemyFactions: ['wei'], bonusValues: [.25], defenseCountdownSeconds: 3, bonusResetGoldCost: 10 }

/** 创建只有一个活动进攻波的副本实例。 */
function run(id = 'run-1'): DungeonRun {
  return { id, playerId: 'p1', level: 1, levelName: '入门轮回', status: 'running', currentWave: 1, startedAt: '', expiresAt: '2030-01-01T00:00:00Z', pendingRewards: [], waves: [{ id: `${id}-w1`, runId: id, waveIndex: 1, waveType: 'attack', enemyFaction: 'wei', enemyTroops: { huWei: 10 }, enemyRemaining: { huWei: 10 }, allyBonus: { side: 'ally', unitType: 'huWei', stat: 'attack', value: .25, label: '虎卫攻击 +25%' }, enemyBonus: { side: 'enemy', unitType: 'huWei', stat: 'defense', value: .25, label: '虎卫防御 +25%' }, rewardPreview: [], troopCap: 100, status: 'active', startedAt: '' }], createdAt: '', updatedAt: '' }
}

/** 创建测试所需的最小公共游戏状态。 */
function gameStore() { return { data: { army: [{ unitType: 'huWei', amount: 50 }], serverTime: '', general: null, generals: [] }, receivedAt: null } as unknown as GameStateStore }

/** 创建独立副本状态。 */
function store(): DungeonStateStore { return { phase: 'idle', playerId: null, config: null, run: null, error: '', operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0, lastReportId: '' } }

describe('轮回绝境状态服务', () => {
  it('并行读取配置和当前实例并回写真实军队', async () => {
    const state = store()
    const game = gameStore()
    const api = { dungeonConfig: vi.fn(async () => config), dungeonRun: vi.fn(async () => ({ run: run(), army: [{ unitType: 'huWei', amount: 42 }], serverTime: '2026-07-14T00:00:00Z' })) } as unknown as GameApi
    await createDungeonStateService(api, state, game).load('p1')
    expect(state.phase).toBe('ready')
    expect(state.run?.id).toBe('run-1')
    expect(game.data?.army[0].amount).toBe(42)
  })

  it('每次波次按独立实例提交并以后端结果推进', async () => {
    const state = { ...store(), phase: 'ready' as const, playerId: 'p1', config, run: run() }
    const game = gameStore()
    const next = { ...run(), currentWave: 2 }
    const attackDungeonWave = vi.fn(async () => ({ run: next, army: [{ unitType: 'huWei', amount: 47 }], serverTime: '2026-07-14T00:00:01Z', battleReport: { id: 'report-1' } }))
    await createDungeonStateService({ attackDungeonWave } as unknown as GameApi, state, game).fight('run-1-w1', { huWei: 20 }, [])
    expect(attackDungeonWave).toHaveBeenCalledWith('p1', 'run-1-w1', { huWei: 20 }, [], expect.stringContaining('run-1-w1_'))
    expect(state.run?.currentWave).toBe(2)
    expect(state.lastReportId).toBe('report-1')
    expect(game.data?.army[0].amount).toBe(47)
  })

  it('拒绝超过本城真实可用兵力的前端提交', async () => {
    const state = { ...store(), phase: 'ready' as const, playerId: 'p1', config, run: run() }
    const attackDungeonWave = vi.fn()
    await createDungeonStateService({ attackDungeonWave } as unknown as GameApi, state, gameStore()).fight('run-1-w1', { huWei: 999 }, [])
    expect(attackDungeonWave).not.toHaveBeenCalled()
    expect(state.actionSucceeded).toBe(false)
  })

  it('允许投入超过旧波次上限的真实可用兵力', async () => {
    const unlimitedRun = run()
    unlimitedRun.waves[0].troopCap = 10
    const state = { ...store(), phase: 'ready' as const, playerId: 'p1', config, run: unlimitedRun }
    const attackDungeonWave = vi.fn(async () => ({ run: unlimitedRun, army: [{ unitType: 'huWei', amount: 30 }], serverTime: '', battleReport: { id: 'report-unlimited' } }))
    await createDungeonStateService({ attackDungeonWave } as unknown as GameApi, state, gameStore()).fight('run-1-w1', { huWei: 20 }, [])
    expect(attackDungeonWave).toHaveBeenCalledOnce()
  })

  it('重置加成返回后端账户金币余额', async () => {
    const state = { ...store(), phase: 'ready' as const, playerId: 'p1', config, run: run() }
    const resetDungeonBonus = vi.fn(async () => ({ run: run(), army: [], accountGold: 88, cost: 10, serverTime: '' }))
    const gold = await createDungeonStateService({ resetDungeonBonus } as unknown as GameApi, state, gameStore()).resetBonus('run-1-w1')
    expect(gold).toBe(88)
    expect(state.actionMessage).toContain('10 金币')
  })
})
