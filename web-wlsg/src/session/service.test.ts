/** 验证会话恢复、失效玩家清理、登录和存档选择边界。 */
import { describe, expect, it, vi } from 'vitest'
import type { GameApi } from '../api/gameApi'
import type { PlayerSummary } from '../api/types'
import { createSessionService, type SessionState } from './service'
import type { SessionStorage, StoredSession } from './storage'

const player: PlayerSummary = { id: 'p1', nickname: '主公', faction: 'wei', totalArmy: 10, buildingLevel: 2, updatedAt: '2026-07-12T00:00:00Z' }

/** 创建可观察的内存会话存储。 */
function createMemorySession(saved: StoredSession | null = null, playerId: string | null = null) {
  let session = saved
  let currentPlayerId = playerId
  const storage: SessionStorage = {
    readSession: () => session,
    writeSession: (value) => { session = value },
    clearSession: () => { session = null },
    readPlayerId: () => currentPlayerId,
    writePlayerId: (value) => { currentPlayerId = value },
    clearPlayerId: () => { currentPlayerId = null },
    getToken: () => session?.token ?? null,
  }
  return { storage, session: () => session, playerId: () => currentPlayerId }
}

/** 创建每个测试独立的初始状态。 */
function initialState(): SessionState {
  return { phase: 'loading', account: null, players: [], currentPlayer: null, submitting: false, error: '', bootstrap: null }
}

/** 创建带默认成功响应的 API 替身。 */
function mockApi(players: PlayerSummary[] = [player]): GameApi {
  return {
    bootstrap: vi.fn(async () => ({ gameName: 'Hero3', modules: [], balance: { buildings: {} }, units: {}, message: 'ok' })),
    login: vi.fn(async () => ({ accountId: 'a1', username: 'hero', gold: 0, token: 'jwt' })),
    accountInfo: vi.fn(async () => ({ accountId: 'a1', username: 'hero', gold: 0 })),
    players: vi.fn(async () => ({ players })),
    gameState: vi.fn(async () => { throw new Error('当前测试不读取游戏状态') }),
    upgradeBuilding: vi.fn(async () => { throw new Error('当前测试不执行建造') }),
    instantCompleteBuilding: vi.fn(async () => { throw new Error('当前测试不执行建造加速') }),
    fillResourcesPaid: vi.fn(async () => { throw new Error('当前测试不执行一键爆仓') }),
    capacityBoostPrices: vi.fn(async () => ({})),
    purchaseCapacityBoost: vi.fn(async () => { throw new Error('当前测试不执行仓库扩容') }),
    militaryView: vi.fn(async () => { throw new Error('当前测试不读取军事状态') }),
    recruit: vi.fn(async () => { throw new Error('当前测试不执行征兵') }),
    instantCompleteRecruit: vi.fn(async () => { throw new Error('当前测试不执行征兵加速') }),
    worldMapView: vi.fn(async () => { throw new Error('当前测试不读取世界地图') }),
    npcCities: vi.fn(async () => ({ cities: [], lastRefreshedAt: '' })),
    refreshNpcCities: vi.fn(async () => { throw new Error('当前测试不刷新 NPC') }),
    attackNpc: vi.fn(async () => { throw new Error('当前测试不进攻 NPC') }),
    scoutNpc: vi.fn(async () => { throw new Error('当前测试不侦查 NPC') }),
    dungeonConfig: vi.fn(async () => ({ levels: [], waves: [], enemyFactions: [], bonusValues: [], defenseCountdownSeconds: 3, bonusResetGoldCost: 10 })),
    dungeonRun: vi.fn(async () => ({ serverTime: '' })),
    startDungeon: vi.fn(async () => { throw new Error('当前测试不开启副本') }),
    attackDungeonWave: vi.fn(async () => { throw new Error('当前测试不进攻副本') }),
    defendDungeonWave: vi.fn(async () => { throw new Error('当前测试不防守副本') }),
    resetDungeonBonus: vi.fn(async () => { throw new Error('当前测试不重置副本加成') }),
    settleDungeon: vi.fn(async () => { throw new Error('当前测试不结算副本') }),
    exitDungeon: vi.fn(async () => { throw new Error('当前测试不退出副本') }),
    mirageRecords: vi.fn(async () => ({ totalRecords: 0, limit: 100, offset: 0, hasMore: false, records: [], rewardTotals: {} })),
    resolveMirageGambling: vi.fn(async () => { throw new Error('当前测试不结算六合博戏') }),
    resolveMirageSlot: vi.fn(async () => { throw new Error('当前测试不结算天机轮转') }),
    redeemMirageRecord: vi.fn(async () => { throw new Error('当前测试不兑换幻境库存') }),
    redeemAllMirage: vi.fn(async () => { throw new Error('当前测试不一键兑换幻境库存') }),
    scoutPvpTarget: vi.fn(async () => { throw new Error('当前测试不执行侦查') }),
    startPvpAttack: vi.fn(async () => { throw new Error('当前测试不执行 PVP 行军') }),
    sendReinforcement: vi.fn(async () => { throw new Error('当前测试不执行增援') }),
    pvpMarches: vi.fn(async () => ({ items: [] })),
    acceleratePvpMarch: vi.fn(async () => { throw new Error('当前测试不执行 PVP 行军加速') }),
    recallPvpMarch: vi.fn(async () => { throw new Error('当前测试不执行 PVP 行军召回') }),
    sentReinforcements: vi.fn(async () => ({ items: [] })),
    receivedReinforcements: vi.fn(async () => ({ items: [] })),
    accelerateReinforcement: vi.fn(async () => { throw new Error('当前测试不执行增援加速') }),
    recallReinforcement: vi.fn(async () => { throw new Error('当前测试不执行增援召回') }),
    listReports: vi.fn(async () => ({ reports: [], page: 1, pageSize: 8, total: 0 })),
    report: vi.fn(async () => { throw new Error('当前测试不读取战报详情') }),
    markReportRead: vi.fn(async () => { throw new Error('当前测试不标记军情') }),
    deleteReport: vi.fn(async () => { throw new Error('当前测试不删除军情') }),
  }
}

describe('会话服务', () => {
  it('刷新后恢复有效账号和当前玩家', async () => {
    const memory = createMemorySession({ accountId: 'a1', username: 'hero', token: 'jwt' }, 'p1')
    const state = initialState()
    await createSessionService(mockApi(), memory.storage, state).initialize()
    expect(state.phase).toBe('game')
    expect(state.currentPlayer?.id).toBe('p1')
  })

  it('保存的玩家不在账号列表时清除选择并返回选档', async () => {
    const memory = createMemorySession({ accountId: 'a1', username: 'hero', token: 'jwt' }, 'other-player')
    const state = initialState()
    await createSessionService(mockApi(), memory.storage, state).initialize()
    expect(state.phase).toBe('players')
    expect(memory.playerId()).toBeNull()
  })

  it('只允许选择当前账号列表中的有效存档', () => {
    const memory = createMemorySession()
    const state = initialState()
    state.players = [player]
    const service = createSessionService(mockApi(), memory.storage, state)
    service.selectPlayer('foreign-player')
    expect(state.phase).not.toBe('game')
    expect(memory.playerId()).toBeNull()
    service.selectPlayer('p1')
    expect(state.phase).toBe('game')
    expect(memory.playerId()).toBe('p1')
  })

  it('登录后保存会话但不保留旧玩家选择', async () => {
    const memory = createMemorySession(null, 'old-player')
    const state = initialState()
    await createSessionService(mockApi(), memory.storage, state).login('hero', 'password-only-for-request')
    expect(memory.session()).toEqual({ accountId: 'a1', username: 'hero', token: 'jwt' })
    expect(memory.playerId()).toBeNull()
    expect(state.phase).toBe('players')
  })

  it('统一失效处理会清除账号和当前玩家', () => {
    const memory = createMemorySession({ accountId: 'a1', username: 'hero', token: 'jwt' }, 'p1')
    const state = initialState()
    state.account = { accountId: 'a1', username: 'hero', gold: 0 }
    const service = createSessionService(mockApi(), memory.storage, state)
    service.handleUnauthorized()
    expect(state.phase).toBe('login')
    expect(memory.session()).toBeNull()
    expect(memory.playerId()).toBeNull()
  })

  it('付费操作后使用后端权威账户金币余额更新页头', () => {
    const state = initialState()
    state.account = { accountId: 'a1', username: 'hero', gold: 250 }
    const service = createSessionService(mockApi(), createMemorySession().storage, state)
    service.updateAccountGold(150)
    expect(state.account.gold).toBe(150)
    service.updateAccountGold(-1)
    expect(state.account.gold).toBe(150)
  })
})
