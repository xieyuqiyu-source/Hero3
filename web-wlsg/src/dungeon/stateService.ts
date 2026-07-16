/** 轮回绝境状态服务：集中处理配置、实例、波次结算和玩家局部状态回写。 */
import type { GameApi } from '../api/gameApi'
import type { GameStateStore } from '../game/stateService'
import type { DungeonActionResult, DungeonConfig, DungeonRun, DungeonWave } from './types'

export type DungeonPhase = 'idle' | 'loading' | 'ready' | 'error'

export interface DungeonStateStore {
  phase: DungeonPhase
  playerId: string | null
  config: DungeonConfig | null
  run: DungeonRun | null
  error: string
  operating: boolean
  actionMessage: string
  actionSucceeded: boolean
  resultVersion: number
  lastReportId: string
  lastGeneralId: string
  lastGeneralExpGained: number
  lastGeneralLevelBefore: number | null
  lastGeneralLevelAfter: number | null
}

export interface DungeonStateService {
  state: DungeonStateStore
  load: (playerId: string, force?: boolean) => Promise<void>
  start: (level: number) => Promise<void>
  fight: (waveId: string, troops: Record<string, number>, generalIds: string[]) => Promise<void>
  resetBonus: (waveId: string) => Promise<number | null>
  settle: () => Promise<void>
  exit: () => Promise<void>
  clear: () => void
}

/** 创建带切档隔离和重复提交保护的轮回绝境服务。 */
export function createDungeonStateService(api: GameApi, state: DungeonStateStore, game: GameStateStore): DungeonStateService {
  let requestVersion = 0
  let controller: AbortController | null = null

  /** 退出或切档时终止旧请求并清除副本状态。 */
  function clear() {
    requestVersion += 1
    controller?.abort()
    controller = null
    Object.assign(state, { phase: 'idle', playerId: null, config: null, run: null, error: '', operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0, lastReportId: '', lastGeneralId: '', lastGeneralExpGained: 0, lastGeneralLevelBefore: null, lastGeneralLevelAfter: null })
  }

  /** 读取公共配置和当前玩家活动实例。 */
  async function load(playerId: string, force = false) {
    if (!playerId || (!force && state.playerId === playerId && (state.phase === 'loading' || state.phase === 'ready'))) return
    const changed = state.playerId !== playerId
    requestVersion += 1
    const currentVersion = requestVersion
    controller?.abort()
    controller = new AbortController()
    Object.assign(state, {
      phase: 'loading',
      playerId,
      run: changed ? null : state.run,
      error: '',
      actionMessage: '',
      ...(changed ? { lastGeneralId: '', lastGeneralExpGained: 0, lastGeneralLevelBefore: null, lastGeneralLevelAfter: null } : {}),
    })
    try {
      const [config, response] = await Promise.all([api.dungeonConfig(controller.signal), api.dungeonRun(playerId, controller.signal)])
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.config = config
      state.run = response.run ?? null
      patchGame({ army: response.army, serverTime: response.serverTime })
      state.phase = 'ready'
    } catch (error) {
      if (currentVersion !== requestVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      state.phase = 'error'
      state.error = error instanceof Error ? error.message : '轮回绝境加载失败'
    }
  }

  /** 把副本接口返回的军队、武将和服务端时间写回当前玩家状态。 */
  function patchGame(result: Pick<DungeonActionResult, 'army' | 'general' | 'generals' | 'serverTime'>) {
    if (!game.data) return
    if (result.army) game.data.army = result.army
    if (result.general !== undefined) game.data.general = result.general
    if (result.generals) game.data.generals = result.generals
    if (result.serverTime) game.data.serverTime = result.serverTime
    game.receivedAt = Date.now()
  }

  /** 统一接收写操作结果并展示后端真实副本状态。 */
  function applyResult(result: DungeonActionResult, message: string) {
    state.run = result.run
    state.lastReportId = result.battleReport?.id ?? state.lastReportId
    state.actionMessage = message
    state.actionSucceeded = true
    patchGame(result)
  }

  /** 在当前存档开启指定难度实例。 */
  async function start(level: number) {
    if (!state.playerId || state.operating || !state.config?.levels.some((item) => item.level === level && item.enabled)) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.operating = true
    state.actionMessage = ''
    try {
      const result = await api.startDungeon(playerId, level)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      Object.assign(state, { lastGeneralId: '', lastGeneralExpGained: 0, lastGeneralLevelBefore: null, lastGeneralLevelAfter: null })
      applyResult(result, `已进入${result.run.levelName}，请在时限内完成十八波攻防`)
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.actionMessage = error instanceof Error ? error.message : '开启轮回绝境失败'
      state.actionSucceeded = false
    } finally {
      finishOperation(currentVersion, playerId)
    }
  }

  /** 校验并提交当前进攻或防守波次。 */
  async function fight(waveId: string, troops: Record<string, number>, generalIds: string[]) {
    const wave = activeWave(state.run)
    if (!state.playerId || !wave || wave.id !== waveId || state.operating || !game.data) return
    const available = Object.fromEntries(game.data.army.map((unit) => [unit.unitType, unit.amount]))
    const normalized = Object.fromEntries(Object.entries(troops).filter(([id, amount]) => id && Number.isInteger(amount) && amount > 0 && amount <= (available[id] ?? 0)))
    const total = Object.values(normalized).reduce((sum, amount) => sum + amount, 0)
    if (!total) {
      state.actionMessage = '请至少选择一个参战兵种'
      state.actionSucceeded = false
      state.resultVersion += 1
      return
    }
    const playerId = state.playerId
    const currentVersion = requestVersion
    const selectedGenerals = [...new Set(generalIds.map((id) => id.trim()).filter(Boolean))].slice(0, 1)
    const actionId = `${wave.id}_${crypto.randomUUID()}`
    state.operating = true
    state.actionMessage = ''
    try {
      const result = wave.waveType === 'defense'
        ? await api.defendDungeonWave(playerId, wave.id, normalized, selectedGenerals, actionId)
        : await api.attackDungeonWave(playerId, wave.id, normalized, selectedGenerals, actionId)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      const expGained = Math.max(0, result.battleReport?.generalExpGained ?? 0)
      Object.assign(state, {
        lastGeneralId: selectedGenerals[0] ?? '',
        lastGeneralExpGained: expGained,
        lastGeneralLevelBefore: result.battleReport?.generalLevelBefore ?? null,
        lastGeneralLevelAfter: result.battleReport?.generalLevelAfter ?? null,
      })
      const progressMessage = result.run.status === 'running' ? `第 ${wave.waveIndex} 波结算完成，已进入下一波` : '本次轮回已经结束，累计奖励已按后端规则结算'
      applyResult(result, selectedGenerals.length ? `${progressMessage}；随军将领获得 ${expGained.toLocaleString('zh-CN')} 经验` : progressMessage)
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.actionMessage = error instanceof Error ? error.message : '轮回波次结算失败'
      state.actionSucceeded = false
    } finally {
      finishOperation(currentVersion, playerId)
    }
  }

  /** 消耗账户金币重置当前波的双方随机加成。 */
  async function resetBonus(waveId: string): Promise<number | null> {
    const wave = activeWave(state.run)
    if (!state.playerId || !wave || wave.id !== waveId || state.operating) return null
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.operating = true
    state.actionMessage = ''
    try {
      const result = await api.resetDungeonBonus(playerId, waveId)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return null
      applyResult(result, `本波加成已重置，消耗 ${result.cost ?? state.config?.bonusResetGoldCost ?? 0} 金币`)
      return result.accountGold ?? null
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return null
      state.actionMessage = error instanceof Error ? error.message : '重置随机加成失败'
      state.actionSucceeded = false
      return null
    } finally {
      finishOperation(currentVersion, playerId)
    }
  }

  /** 主动结算已经结束或到期的副本实例。 */
  async function settle() {
    if (!state.playerId || state.operating || !state.run) return
    const playerId = state.playerId
    const currentVersion = requestVersion
    state.operating = true
    state.actionMessage = ''
    try {
      const result = await api.settleDungeon(playerId)
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      applyResult(result, '轮回绝境奖励已结算入账')
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.actionMessage = error instanceof Error ? error.message : '副本奖励结算失败'
      state.actionSucceeded = false
    } finally {
      finishOperation(currentVersion, playerId)
    }
  }

  /** 调用退出接口并回到副本层级选择，不删除历史实例和战报。 */
  async function exit() {
    if (!state.playerId || state.operating || !state.run || state.run.status === 'running') return
    const playerId = state.playerId
    const runId = state.run.id
    const currentVersion = requestVersion
    state.operating = true
    state.actionMessage = ''
    try {
      const result = await api.exitDungeon(playerId, runId)
      if (currentVersion !== requestVersion || state.playerId !== playerId || result.runId !== runId) return
      state.run = null
      state.lastReportId = ''
      state.lastGeneralId = ''
      state.lastGeneralExpGained = 0
      state.lastGeneralLevelBefore = null
      state.lastGeneralLevelAfter = null
      state.actionMessage = '已退出轮回绝境，可重新选择挑战层级'
      state.actionSucceeded = true
      if (game.data && result.serverTime) game.data.serverTime = result.serverTime
      game.receivedAt = Date.now()
    } catch (error) {
      if (currentVersion !== requestVersion || state.playerId !== playerId) return
      state.actionMessage = error instanceof Error ? error.message : '退出轮回绝境失败'
      state.actionSucceeded = false
    } finally {
      finishOperation(currentVersion, playerId)
    }
  }

  /** 只结束仍属于当前存档的写操作。 */
  function finishOperation(version: number, playerId: string) {
    if (version !== requestVersion || state.playerId !== playerId) return
    state.operating = false
    state.resultVersion += 1
  }

  return { state, load, start, fight, resetBonus, settle, exit, clear }
}

/** 返回实例当前活动波次。 */
export function activeWave(run: DungeonRun | null): DungeonWave | null {
  return run?.waves.find((wave) => wave.waveIndex === run.currentWave) ?? null
}
