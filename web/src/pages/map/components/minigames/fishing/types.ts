import type { MiniGameRecord } from '@/types/game'

export interface FishCatch {
  name: string
  rarity: 'common' | 'rare' | 'epic' | 'legendary'
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
