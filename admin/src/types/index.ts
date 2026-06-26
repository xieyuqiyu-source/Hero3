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
    mailCode?: string
  }
  resources: {
    items: Record<string, number>
    capacity: Record<string, number>
  }
  inventory?: Record<string, ItemStack>
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
  general?: {
    id: string
    name: string
    level: number
    exp: number
    availableStatPoints?: number
    stats?: Record<string, number>
  } | null
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

export interface ItemStack {
  itemId: string
  amount: number
  obtainedAt?: string
  updatedAt?: string
}

export interface ItemEffect {
  type: string
  amount?: number
  resources?: Record<string, number>
  unitByFaction?: Record<string, string>
}

export interface ItemDefinition {
  id: string
  name: string
  description: string
  type: string
  rarity: string
  icon?: string
  usable: boolean
  stackable: boolean
  maxStack: number
  useTarget: string
  effects: ItemEffect[]
  metadata?: Record<string, unknown>
}

export interface UnitConfig {
  name: string
  description: string
  category: string
  role?: string
  icon: string
  stats: Record<string, number>
  cost: Record<string, number>
  trainSeconds: number
  unlock: Record<string, unknown>
}

export type FactionUnits = Record<string, UnitConfig>
export type UnitsConfig = Record<string, FactionUnits>

export interface FactionGeneralInfo {
  id: string
  name: string
  title: string
}

export interface FactionConfig {
  name: string
  description: string
  icon: string
  traits: Record<string, number>
  generals: FactionGeneralInfo[]
}

export type FactionsConfig = Record<string, FactionConfig>

export interface CombatRuleConfig {
  id: string
  name: string
  mode: string
  exponent: number
  equalResult: string
  lossDistribution: string
  defenseFormula: string
}

export interface CombatWallEntry {
  base: number
}

export interface CombatConfig {
  activeCombatRules: Record<string, string>
  rules: Record<string, CombatRuleConfig>
  wallConfig: Record<string, CombatWallEntry>
}

export interface ModifierConfig {
  key: string
  value: number
  mode: string
}

export interface GeneralTraitConfig {
  traitId: string
  enabled: boolean
  params: Record<string, number>
}

export interface GeneralHeroConfig {
  id: string
  name: string
  faction: string
  title: string
  rarity: string
  enabled: boolean
  buffs: Record<string, number>
  traits: GeneralTraitConfig[]
}

export interface GeneralsCommonConfig {
  expCurve: number[]
  levelBuffs: Record<string, Record<string, number>>
}

export interface GeneralsConfig {
  enabled: boolean
  common: GeneralsCommonConfig
  heroes: Record<string, GeneralHeroConfig>
}

export interface TraitParamField {
  key: string
  label: string
  description: string
  default: number
  min: number
  max: number
  step: number
}

export interface TraitMeta {
  id: string
  name: string
  description: string
  paramSchema: TraitParamField[]
}

export interface TraitRegistryResponse {
  traits: TraitMeta[]
}

export interface FishingRarityConfig {
  label: string
  color: string
  bg: string
  border: string
  weight: number
  glow: string
}

export interface FishingBaitConfig {
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

export interface FishingFishConfig {
  name: string
  rarity: string
  reward: string
  rewardAmount: number
  description: string
  emoji: string
}

export interface FishingConfig {
  rarities: Record<string, FishingRarityConfig>
  baits: FishingBaitConfig[]
  fishPool: FishingFishConfig[]
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
  mailCode?: string
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
  modifiersByLevel?: Record<string, ModifierConfig[]>
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
