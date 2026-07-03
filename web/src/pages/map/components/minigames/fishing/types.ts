// 本文件定义仙池垂钓前端运行时使用的鱼、鱼饵和界面状态类型。
import type { MiniGameRecord } from '@/types/game'

export type FishRarity = 'common' | 'rare' | 'epic' | 'legendary' | 'mythic'

export interface FishCatch {
  name: string
  rarity: FishRarity | string
  reward: string
  rewardAmount: number
  description: string
  emoji: string
}

export interface BaitType {
  id: string
  name: string
  tier: string
  description: string
  rarityBoost: number
  rarityWeights?: Record<string, number>
  minRarity?: string
  maxRarity?: string
  biteDelayMultiplier?: number
  cityGoldCost: number
  biteChance: number
  biteWindowMs: number
  sweetStart: number
  sweetEnd: number
}

export type GamePhase = 'idle' | 'casting' | 'waiting' | 'biting' | 'reeling' | 'caught' | 'escaped'

export interface FishingStats {
  totalCasts: number
  totalCaught: number
  combo: number
  bestCombo: number
  legendaryCount: number
  mythicCount: number
  epicCount: number
}

export interface Bubble {
  id: number
  x: number
  size: number
  delay: number
}

export interface FishShadow {
  x: number
  visible: boolean
}

export interface BiteSpot {
  x: number
  y: number
}

export type FishingRecord = MiniGameRecord
