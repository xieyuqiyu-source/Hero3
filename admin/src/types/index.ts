export interface HealthState {
  status: string
  service: string
  version: string
  environment: string
  time: string
}

export interface GameState {
  player: {
    id: string
    nickname: string
    faction: string
  }
  resources: {
    items: Record<string, number>
    capacity: Record<string, number>
  }
  resourceProduction: Record<string, number>
  resourceSettledAt: string
  buildings: Array<{
    id: string
    type: string
    level: number
    upgradeEndsAt: string | null
  }>
  army: Array<{
    unitType: string
    amount: number
  }>
  recruitQueues: Array<{
    id: string
    unitType: string
    amount: number
    endsAt: string
  }>
  npcState?: NpcState | null
  mapTargets: Array<{
    id: string
    type: string
    level: number
    power: number
    rewards?: Record<string, number>
  }>
  recentBattleReports: Array<{
    id: string
    targetId?: string
    result: string
    playerPower?: number
    enemyPower?: number
    lostUnits?: Record<string, number>
    rewards?: Record<string, number>
    createdAt?: string
  }>
  unreadMessageCount: number
  unreadMailCount: number
  buffs?: Array<{
    id: string
    source: string
    key: string
    value: number
    mode: string
    expiresAt?: string
    createdAt: string
    note?: string
  }>
  serverTime: string
}

export interface MailAttachment {
  type: string
  itemId: string
  amount: number
}

export interface Mail {
  id: string
  playerId: string
  mailType: string
  senderType: string
  senderName: string
  title: string
  content: string
  attachments?: MailAttachment[]
  sourceType?: string
  sourceId?: string
  isRead: boolean
  isClaimed: boolean
  expiresAt?: string
  createdAt: string
  readAt?: string
}

export interface MailPage {
  mails: Mail[]
  page: number
  pageSize: number
  total: number
  unread: number
}

export interface PlayerSummary {
  id: string
  nickname: string
  faction: string
  updatedAt: string
}

export interface AccountSummary {
  id: string
  username: string
  createdAt: string
  players: PlayerSummary[]
}

export interface NpcTrait {
  id: string
  name: string
  buffs: Record<string, number>
}

export interface NpcCity {
  id: string
  name: string
  faction: string
  tier: 'small' | 'medium' | 'large' | 'golden'
  resources: Record<string, number>
  storageCapacity: Record<string, number>
  productionPerHour: Record<string, number>
  army: Array<{
    unitType: string
    amount: number
  }>
  maxArmy: Array<{
    unitType: string
    amount: number
  }>
  armyRecoveryRate: number
  recoveryProfile: string
  traits: NpcTrait[]
  resourceSettledAt: string
  armySettledAt: string
  generatedAt: string
}

export interface NpcState {
  cities: NpcCity[]
  lastRefreshedAt: string
}

export interface IntRange {
  min: number
  max: number
}

export interface NpcCountRule {
  guaranteed: number
  weight: number
}

export interface NpcTierConfig {
  multiplier: number
  armyRange: IntRange
  armyTypes: IntRange
  traitCount: IntRange
  count: NpcCountRule
}

export interface NpcRecoveryProfile {
  id: string
  name: string
  armyMultiplier: number
  resourceMultiplier: number
  weight: number
}

export interface NpcTraitConfig {
  id: string
  name: string
  buffs: Record<string, number>
  weight: number
}

export interface NpcConfig {
  baseProduction: number
  baseStorage: number
  refreshIntervalHours: number
  manualRefreshCostGold: number
  goldenAppearRate: number
  totalCities: number
  tiers: Record<string, NpcTierConfig>
  recoveryProfiles: NpcRecoveryProfile[]
  traitPool: NpcTraitConfig[]
  cityNames: string[]
  scoutCost: Record<string, number>
}

export interface BuildingConfig {
  type: string
  name: string
  resourceType?: string
  productionByLevel?: number[]
  capacityByLevel?: number[]
  upgradeCostByLevel?: Record<string, Record<string, number>>
  upgradeSecondsByLevel?: Record<string, number>
}

export interface BalanceConfig {
  baseProduction: Record<string, number>
  buildings: Record<string, BuildingConfig>
  overflowToCityGold: number
  exchangeRate: number
  reverseExchangeRate: number
  exchangeCooldownSecs: number
  cityGoldPerSecond: number
  boostBaseCost: number
  boostMultiplierFactor: Record<string, number>
  boostDurationFactor: Record<string, number>
}

export interface GoldLedgerEntry {
  id: number
  accountId?: string
  playerId?: string
  currency: 'gold' | 'cityGold'
  direction: 'credit' | 'debit'
  amount: number
  balanceAfter: number
  refType?: string
  refId?: string
  reason?: string
  createdAt: string
}
