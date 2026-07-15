/** 验证万象幻境记录加载、两种结算、军队回写和库存兑换。 */
import { describe, expect, it, vi } from 'vitest'
import type { GameApi } from '../api/gameApi'
import type { GameStateStore } from '../game/stateService'
import type { MirageRecord } from './types'
import { createMirageStateService, type MirageStateStore } from './stateService'

const record: MirageRecord = { id: 'm1', playerId: 'p1', gameType: 'slot', resultName: '赤金符三连', rarity: 'epic', rewardUnit: '虎卫', rewardAmount: 3000, remainingAmount: 3000, betUnit: '虎卫', betAmount: 1000, createdAt: '' }

/** 创建测试所需的最小游戏状态。 */
function store(): MirageStateStore { return { phase: 'idle', playerId: null, summary: null, error: '', operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0, gamblingResult: null, slotResult: null } }

/** 创建带真实兵力字段的最小公共状态。 */
function gameStore() { return { data: { army: [{ unitType: 'huWei', amount: 5000 }], serverTime: '' }, receivedAt: null } as unknown as GameStateStore }

describe('万象幻境状态服务', () => {
  it('读取全部记录和库存汇总', async () => {
    const state = store()
    const mirageRecords = vi.fn(async () => ({ totalRecords: 1, limit: 100, offset: 0, hasMore: false, records: [record], rewardTotals: { 虎卫: 3000 } }))
    await createMirageStateService({ mirageRecords } as unknown as GameApi, state, gameStore()).load('p1')
    expect(state.phase).toBe('ready')
    expect(state.summary?.records).toHaveLength(1)
  })

  it('天机轮转结果回写军队并进入当前库存', async () => {
    const state: MirageStateStore = { ...store(), phase: 'ready', playerId: 'p1', summary: { totalRecords: 0, limit: 100, offset: 0, hasMore: false, records: [], rewardTotals: {} } }
    const game = gameStore()
    const resolveMirageSlot = vi.fn(async () => ({ record, army: [{ unitType: 'huWei', amount: 4000 }], serverTime: '2026-07-14T00:00:00Z', won: true, grid: [['a']], lineBet: 1000, lineCount: 5, totalBet: 1000, winningLines: [], freeSpins: [], bonusRewards: [], allPayRewards: [], betUnitId: 'huWei', betUnit: '虎卫', betAmount: 1000, winAmount: 3000, rewardRarity: 'epic' }))
    await createMirageStateService({ resolveMirageSlot } as unknown as GameApi, state, game).spin('huWei', 1000)
    expect(resolveMirageSlot).toHaveBeenCalledWith('p1', 'huWei', 1000)
    expect(game.data?.army[0].amount).toBe(4000)
    expect(state.summary?.rewardTotals.虎卫).toBe(3000)
  })

  it('单条兑换使用后端返回记录更新剩余库存', async () => {
    const state: MirageStateStore = { ...store(), phase: 'ready', playerId: 'p1', summary: { totalRecords: 1, limit: 100, offset: 0, hasMore: false, records: [record], rewardTotals: { 虎卫: 3000 } } }
    const redeemed = { ...record, remainingAmount: 0 }
    const redeemMirageRecord = vi.fn(async () => ({ record: redeemed, army: [{ unitType: 'huWei', amount: 8000 }], serverTime: '', redeemedUnitId: 'huWei', redeemedUnit: '虎卫', redeemedAmount: 3000, redeemedTarget: 'army' as const }))
    await createMirageStateService({ redeemMirageRecord } as unknown as GameApi, state, gameStore()).redeem('m1', 3000)
    expect(state.summary?.records[0].remainingAmount).toBe(0)
    expect(state.summary?.rewardTotals).toEqual({})
  })
})
