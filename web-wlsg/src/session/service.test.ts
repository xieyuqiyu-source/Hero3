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
    fillResourcesPaid: vi.fn(async () => { throw new Error('当前测试不执行一键爆仓') }),
    militaryView: vi.fn(async () => { throw new Error('当前测试不读取军事状态') }),
    recruit: vi.fn(async () => { throw new Error('当前测试不执行征兵') }),
    instantCompleteRecruit: vi.fn(async () => { throw new Error('当前测试不执行征兵加速') }),
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
})
