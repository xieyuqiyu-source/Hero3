/** 本文件验证军情真实请求、过期隔离、详情已读和删除防重。 */
import { describe, expect, it, vi } from 'vitest'
import type { GameApi } from '../api/gameApi'
import type { BattleReportPageResponse, BattleReportState } from '../game/types'
import { createIntelligenceService, type IntelligenceStore } from './stateService'

/** 创建字段完整且允许覆盖的后端战报。 */
function report(overrides: Partial<BattleReportState> = {}): BattleReportState {
  return { id: 'r1', playerId: 'p1', viewType: 'attack', battleType: 'pvp', title: '主公 攻击 敌城', type: 'attack', result: 'attacker_victory', read: false, createdAt: '2026-07-13T08:00:00Z', ...overrides }
}

/** 创建每个测试独立的军情状态。 */
function state(): IntelligenceStore {
  return { phase: 'idle', playerId: null, activeTab: 'all', page: 1, pageSize: 8, reports: [], total: 0, error: '', detail: null, detailLoading: false, detailError: '', deleting: false, actionMessage: '' }
}

/** 创建可由测试控制完成时机的 Promise。 */
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}

describe('军情状态服务', () => {
  it('分类标签向后端传递类型筛选并只显示已读军情', async () => {
    const listReports = vi.fn(async () => ({ reports: [report({ id: 'read', read: true }), report({ id: 'unread' })], page: 1, pageSize: 50, total: 2 }))
    const store = state()
    const service = createIntelligenceService({ listReports } as unknown as GameApi, store)
    await service.load('p1', 'defense', 1)
    expect(listReports).toHaveBeenCalledWith('p1', 1, 50, { viewType: 'defense' }, expect.any(AbortSignal))
    expect(store.phase).toBe('ready')
    expect(store.reports.map((item) => item.id)).toEqual(['read'])
  })

  it('最新标签分批读取全部可见战报并只分页显示未读项', async () => {
    const listReports = vi.fn()
      .mockResolvedValueOnce({ reports: [report({ id: 'read', read: true }), report({ id: 'unread-1' })], page: 1, pageSize: 2, total: 3 })
      .mockResolvedValueOnce({ reports: [report({ id: 'unread-2' })], page: 2, pageSize: 2, total: 3 })
    const store = state()
    store.pageSize = 1
    const service = createIntelligenceService({ listReports } as unknown as GameApi, store)
    await service.load('p1')
    expect(listReports).toHaveBeenNthCalledWith(1, 'p1', 1, 50, undefined, expect.any(AbortSignal))
    expect(listReports).toHaveBeenNthCalledWith(2, 'p1', 2, 50, undefined, expect.any(AbortSignal))
    expect(store).toMatchObject({ phase: 'ready', page: 1, total: 2 })
    expect(store.reports.map((item) => item.id)).toEqual(['unread-1'])
    await service.selectPage(2)
    expect(listReports).toHaveBeenCalledTimes(2)
    expect(store.reports.map((item) => item.id)).toEqual(['unread-2'])
  })

  it('并发结算异常返回同一掠夺战报时列表只保留一份权威记录', async () => {
    const listReports = vi.fn(async () => ({
      reports: [report({ id: 'plunder-r1', title: '首次权威掠夺战报' }), report({ id: 'plunder-r1', title: '重复掠夺战报' })],
      page: 1, pageSize: 50, total: 2,
    }))
    const store = state()
    const service = createIntelligenceService({ listReports } as unknown as GameApi, store)
    await service.load('p1')
    expect(store).toMatchObject({ phase: 'ready', total: 1 })
    expect(store.reports).toHaveLength(1)
    expect(store.reports[0]).toMatchObject({ id: 'plunder-r1', title: '首次权威掠夺战报' })
  })

  it('空响应和网络失败分别进入空态与错误态', async () => {
    const store = state()
    const listReports = vi.fn<() => Promise<BattleReportPageResponse>>().mockResolvedValueOnce({ reports: [], page: 1, pageSize: 8, total: 0 }).mockRejectedValueOnce(new Error('网络失败'))
    const service = createIntelligenceService({ listReports } as unknown as GameApi, store)
    await service.load('p1')
    expect(store).toMatchObject({ phase: 'ready', reports: [], total: 0 })
    await service.refresh()
    expect(store).toMatchObject({ phase: 'error', reports: [], error: '网络失败' })
  })

  it('快速切换存档时忽略旧请求结果', async () => {
    const first = deferred<BattleReportPageResponse>()
    const listReports = vi.fn().mockImplementationOnce(() => first.promise).mockResolvedValueOnce({ reports: [report({ id: 'new', playerId: 'p2', title: '新存档军情' })], page: 1, pageSize: 8, total: 1 })
    const store = state()
    const service = createIntelligenceService({ listReports } as unknown as GameApi, store)
    const oldLoad = service.load('p1')
    await service.load('p2')
    first.resolve({ reports: [report({ id: 'old', title: '旧存档军情' })], page: 1, pageSize: 8, total: 1 })
    await oldLoad
    expect(store.playerId).toBe('p2')
    expect(store.reports.map((item) => item.id)).toEqual(['new'])
  })

  it('打开最新未读详情后标记已读、移出最新并回写未读数量', async () => {
    const patch = vi.fn()
    const listReports = vi.fn(async () => ({ reports: [report()], page: 1, pageSize: 8, total: 1 }))
    const detail = vi.fn(async () => report({ summary: '完整战报' }))
    const markReportRead = vi.fn(async () => ({ unreadMessageCount: 2, serverTime: '2026-07-13T08:01:00Z' }))
    const store = state()
    const service = createIntelligenceService({ listReports, report: detail, markReportRead } as unknown as GameApi, store, patch)
    await service.load('p1')
    await service.openReport('r1')
    expect(detail).toHaveBeenCalledWith('p1', 'r1', expect.any(AbortSignal))
    expect(markReportRead).toHaveBeenCalledWith('p1', 'r1')
    expect(store.detail?.summary).toBe('完整战报')
    expect(store.reports).toEqual([])
    expect(store.total).toBe(0)
    expect(patch).toHaveBeenCalledWith({ unreadMessageCount: 2, serverTime: '2026-07-13T08:01:00Z' }, 'p1')
  })

  it('串行删除去重后的已选军情并阻止重复提交', async () => {
    const pending = deferred<{ unreadMessageCount: number; serverTime: string }>()
    const listReports = vi.fn().mockResolvedValueOnce({ reports: [report(), report({ id: 'r2' })], page: 1, pageSize: 8, total: 2 }).mockResolvedValueOnce({ reports: [], page: 1, pageSize: 8, total: 0 })
    const deleteReport = vi.fn().mockImplementationOnce(() => pending.promise).mockResolvedValue({ unreadMessageCount: 0, serverTime: '2026-07-13T08:02:00Z' })
    const store = state()
    const service = createIntelligenceService({ listReports, deleteReport } as unknown as GameApi, store)
    await service.load('p1')
    const deleting = service.deleteReports(['r1', 'r1', 'r2'])
    void service.deleteReports(['r2'])
    expect(deleteReport).toHaveBeenCalledTimes(1)
    pending.resolve({ unreadMessageCount: 1, serverTime: '2026-07-13T08:01:00Z' })
    await deleting
    expect(deleteReport).toHaveBeenCalledTimes(2)
    expect(store).toMatchObject({ deleting: false, total: 0, actionMessage: '已删除 2 条军情' })
  })
})
