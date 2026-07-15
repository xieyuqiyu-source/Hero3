/** 万象幻境状态服务：集中处理后端权威随机结算、库存和兑换。 */
import type { GameApi } from '../api/gameApi'
import type { GameStateStore } from '../game/stateService'
import type { GamblingRoundResult, MirageGameType, MirageSummary, SlotRoundResult } from './types'

export type MiragePhase = 'idle' | 'loading' | 'ready' | 'error'

export interface MirageStateStore {
  phase: MiragePhase
  playerId: string | null
  summary: MirageSummary | null
  error: string
  operating: boolean
  actionMessage: string
  actionSucceeded: boolean
  resultVersion: number
  gamblingResult: GamblingRoundResult | null
  slotResult: SlotRoundResult | null
}

export interface MirageStateService {
  state: MirageStateStore
  load: (playerId: string, force?: boolean) => Promise<void>
  gamble: (unitType: string, amount: number, betId: string, exactNumber: number) => Promise<void>
  spin: (unitType: string, amount: number) => Promise<void>
  redeem: (recordId: string, amount: number) => Promise<void>
  redeemAll: (gameType: MirageGameType) => Promise<void>
  clear: () => void
}

/** 创建具有切档隔离、重复提交保护和真实军队回写的万象幻境服务。 */
export function createMirageStateService(api: GameApi, state: MirageStateStore, game: GameStateStore): MirageStateService {
  let requestVersion = 0
  let controller: AbortController | null = null

  /** 退出或切档时清空小游戏记录与本局结果。 */
  function clear() {
    requestVersion += 1
    controller?.abort()
    controller = null
    Object.assign(state, { phase: 'idle', playerId: null, summary: null, error: '', operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0, gamblingResult: null, slotResult: null })
  }

  /** 读取当前存档全部小游戏记录和待兑换库存。 */
  async function load(playerId: string, force = false) {
    if (!playerId || (!force && state.playerId === playerId && (state.phase === 'loading' || state.phase === 'ready'))) return
    const changed = state.playerId !== playerId
    requestVersion += 1
    const currentVersion = requestVersion
    controller?.abort()
    controller = new AbortController()
    Object.assign(state, { phase: 'loading', playerId, summary: changed ? null : state.summary, error: '', actionMessage: '', gamblingResult: changed ? null : state.gamblingResult, slotResult: changed ? null : state.slotResult })
    try {
      const summary = await api.mirageRecords(playerId, controller.signal)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.summary = normalizeSummary(summary)
      state.phase = 'ready'
    } catch (error) {
      if (currentVersion !== requestVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      state.phase = 'error'
      state.error = error instanceof Error ? error.message : '万象幻境加载失败'
    }
  }

  /** 把小游戏扣兵或兑换后的真实军队写回公共游戏状态。 */
  function patchArmy(army: NonNullable<GamblingRoundResult['army']>, serverTime: string) {
    if (!game.data) return
    game.data.army = army
    if (serverTime) game.data.serverTime = serverTime
    game.receivedAt = Date.now()
  }

  /** 用本局记录更新列表和库存汇总。 */
  function prependRecord(record: GamblingRoundResult['record']) {
    const summary = state.summary ?? normalizeSummary({ totalRecords: 0, limit: 100, offset: 0, hasMore: false, records: [], rewardTotals: {} })
    const existed = summary.records.some((item) => item.id === record.id)
    summary.records = [record, ...summary.records.filter((item) => item.id !== record.id)]
    if (!existed) summary.totalRecords += 1
    if (record.remainingAmount > 0 && record.rewardUnit) summary.rewardTotals[record.rewardUnit] = (summary.rewardTotals[record.rewardUnit] ?? 0) + record.remainingAmount
    state.summary = summary
  }

  /** 提交六合博戏的兵种、数量和押注项。 */
  async function gamble(unitType: string, amount: number, betId: string, exactNumber: number) {
    await operate(async (playerId) => {
      const result = await api.resolveMirageGambling(playerId, unitType, amount, betId, exactNumber)
      state.gamblingResult = result
      state.slotResult = null
      if (result.army) patchArmy(result.army, result.serverTime)
      prependRecord(result.record)
      return result.won ? `骰盅揭晓：${result.betLabel}命中，赢得 ${result.winAmount.toLocaleString('zh-CN')} ${result.betUnit}` : `骰盅揭晓：${result.diceTotal} 点，本局未命中`
    }, '六合博戏结算失败')
  }

  /** 提交天机轮转的一次真实 3×3 后端结算。 */
  async function spin(unitType: string, amount: number) {
    await operate(async (playerId) => {
      const result = await api.resolveMirageSlot(playerId, unitType, amount)
      state.slotResult = result
      state.gamblingResult = null
      if (result.army) patchArmy(result.army, result.serverTime)
      prependRecord(result.record)
      return result.won ? `天机显象，赢得 ${result.winAmount.toLocaleString('zh-CN')} ${result.betUnit}` : '天机归寂，本局未形成奖励'
    }, '天机轮转结算失败')
  }

  /** 兑换单条记录的全部或指定数量库存。 */
  async function redeem(recordId: string, amount: number) {
    await operate(async (playerId) => {
      const result = await api.redeemMirageRecord(playerId, recordId, amount)
      if (result.army) patchArmy(result.army, result.serverTime)
      if (state.summary) {
        state.summary.records = state.summary.records.map((record) => record.id === result.record.id ? result.record : record)
        rebuildTotals(state.summary)
      }
      return `已兑换 ${result.redeemedAmount.toLocaleString('zh-CN')} ${result.redeemedUnit}${result.redeemedTarget === 'garrison' ? '，已进入编外驻防' : ''}`
    }, '库存兑换失败')
  }

  /** 一键兑换指定玩法全部可兑换库存。 */
  async function redeemAll(gameType: MirageGameType) {
    await operate(async (playerId) => {
      const result = await api.redeemAllMirage(playerId, gameType)
      if (result.army) patchArmy(result.army, result.serverTime)
      const summary = await api.mirageRecords(playerId)
      state.summary = normalizeSummary(summary)
      return result.redeemedAmount > 0 ? `一键兑换完成，共获得 ${result.redeemedAmount.toLocaleString('zh-CN')} 兵力` : '当前没有可兑换库存'
    }, '一键兑换失败')
  }

  /** 统一处理小游戏写操作和反馈。 */
  async function operate(action: (playerId: string) => Promise<string>, fallback: string) {
    if (!state.playerId || state.operating || !game.data) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.operating = true
    state.actionMessage = ''
    try {
      const message = await action(playerId)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.actionMessage = message
      state.actionSucceeded = true
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.actionMessage = error instanceof Error ? error.message : fallback
      state.actionSucceeded = false
    } finally {
      if (currentVersion === requestVersion && state.playerId === playerId) {
        state.operating = false
        state.resultVersion += 1
      }
    }
  }

  return { state, load, gamble, spin, redeem, redeemAll, clear }
}

/** 补齐后端可能省略的空数组和空汇总。 */
function normalizeSummary(summary: MirageSummary): MirageSummary {
  return { ...summary, records: summary.records ?? [], rewardTotals: { ...(summary.rewardTotals ?? {}) } }
}

/** 按仍可兑换的记录重新计算库存汇总。 */
function rebuildTotals(summary: MirageSummary) {
  summary.rewardTotals = summary.records.reduce<Record<string, number>>((totals, record) => {
    if (record.rewardUnit && record.remainingAmount > 0) totals[record.rewardUnit] = (totals[record.rewardUnit] ?? 0) + record.remainingAmount
    return totals
  }, {})
}
