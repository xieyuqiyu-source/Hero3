/** 万象幻境小游戏的记录、结算与兑换接口类型。 */
import type { ArmyUnitState, ReinforcementListItem } from '../game/types'

export type MirageGameType = 'gambling' | 'slot'

export interface MirageRecord {
  id: string
  playerId: string
  gameType: string
  resultName: string
  rarity: string
  rewardUnit: string
  rewardAmount: number
  remainingAmount: number
  betUnit?: string
  betAmount?: number
  createdAt: string
}

export interface MirageSummary {
  totalRecords: number
  limit: number
  offset: number
  hasMore: boolean
  records: MirageRecord[]
  rewardTotals: Record<string, number>
}

export interface GamblingRoundResult {
  record: MirageRecord
  army?: ArmyUnitState[]
  serverTime: string
  won: boolean
  multiplier: number
  betUnitId: string
  betUnit: string
  betAmount: number
  winAmount: number
  diceTotal: number
  diceValues: number[]
  betLabel: string
  rewardRarity: string
}

export interface SlotWinningLine { lineId: string; symbol: string; symbolName: string; multiplier: number; amount: number; positions: number[][] }
export interface SlotBonusReward { multiplier: number; amount: number; positions: number[][] }
export interface SlotAllPayReward { symbol: string; symbolName: string; count: number; multiplier: number; amount: number; positions: number[][] }
export interface SlotFreeSpinResult { spinIndex: number; grid: string[][]; winningLines: SlotWinningLine[]; bonusRewards: SlotBonusReward[]; allPayRewards: SlotAllPayReward[]; scatterCount: number; retriggeredFreeSpins: number; winAmount: number }

export interface SlotRoundResult {
  record: MirageRecord
  army?: ArmyUnitState[]
  serverTime: string
  won: boolean
  grid: string[][]
  lineBet: number
  lineCount: number
  totalBet: number
  winningLines: SlotWinningLine[]
  freeSpins: SlotFreeSpinResult[]
  bonusRewards: SlotBonusReward[]
  allPayRewards: SlotAllPayReward[]
  betUnitId: string
  betUnit: string
  betAmount: number
  winAmount: number
  rewardRarity: string
}

export interface MirageRedeemResult {
  record: MirageRecord
  army?: ArmyUnitState[]
  serverTime: string
  redeemedUnitId: string
  redeemedUnit: string
  redeemedAmount: number
  redeemedTarget: 'army' | 'garrison'
  garrison?: ReinforcementListItem
}

export interface MirageRedeemAllResult {
  army?: ArmyUnitState[]
  serverTime: string
  redeemedUnits: Record<string, number>
  redeemedAmount: number
  redeemedRecords: number
  garrisonedUnits?: Record<string, number>
  garrisonRecords?: number
  skippedUnits: Record<string, number>
  skippedRecords: number
}
