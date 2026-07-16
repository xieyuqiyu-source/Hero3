/** 轮回绝境副本的真实接口类型。 */
import type { ArmyUnitState, BattleReportState, GeneralState } from '../game/types'

export interface DungeonItemStack { slotId?: string; itemId: string; amount: number; obtainedAt?: string; updatedAt?: string }

export interface DungeonReward {
  type: string
  id: string
  amount: number
  metadata?: Record<string, unknown>
}

export interface DungeonLevelConfig {
  level: number
  name: string
  wavePowerBase: number
  playerTroopCap: number
  enemyTroopBase: number
  durationSeconds: number
  rewardExpCap: number
  enabled: boolean
}

export interface DungeonConfig {
  levels: DungeonLevelConfig[]
  waves: Array<{ waveIndex: number; rewardPreview?: DungeonReward[]; expBudgetRate: number; expRandomMin: number; expRandomMax: number; fixedRewards: DungeonReward[]; dropPoolId?: string }>
  enemyFactions: string[]
  bonusValues: number[]
  defenseCountdownSeconds: number
  bonusResetGoldCost: number
}

export interface DungeonBonus {
  side: string
  unitType: string
  stat: string
  value: number
  label: string
  unitName?: string
  faction?: string
}

export interface DungeonWave {
  id: string
  runId: string
  waveIndex: number
  waveType: 'attack' | 'defense' | string
  enemyFaction: string
  enemyTroops: Record<string, number>
  enemyRemaining: Record<string, number>
  allyBonus: DungeonBonus
  enemyBonus: DungeonBonus
  rewardPreview: DungeonReward[]
  rewardResult?: DungeonReward[]
  troopCap: number
  status: string
  startedAt: string
  clearedAt?: string
}

export interface DungeonRun {
  id: string
  playerId: string
  level: number
  levelName: string
  status: string
  currentWave: number
  startedAt: string
  expiresAt: string
  completedAt?: string
  failedAt?: string
  endedReason?: string
  pendingRewards: DungeonReward[]
  rewardGrantedAt?: string
  exitedAt?: string
  waves: DungeonWave[]
  createdAt: string
  updatedAt: string
}

export interface DungeonRunResponse {
  run?: DungeonRun
  army?: ArmyUnitState[]
  serverTime: string
}

export interface DungeonActionResult {
  run: DungeonRun
  battleReport?: BattleReportState
  army?: ArmyUnitState[]
  inventory?: Record<string, DungeonItemStack>
  inventorySlots?: DungeonItemStack[]
  general?: GeneralState
  generals?: GeneralState[]
  accountGold?: number
  cost?: number
  serverTime: string
}

export interface DungeonExitResult {
  runId: string
  exitedAt: string
  serverTime: string
}
