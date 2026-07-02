/* 本文件定义 GM 后台使用的接口数据类型。 */
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
  inventorySlots?: ItemStack[]
  resourceProduction: Record<string, number>
  resourceSettledAt: string
  buildings: Array<{
    id: string
    type: string
    level: number
    status?: string
    upgradeEndsAt: string | null
    statusEndsAt?: string | null
  }>
  resourceSlots?: Array<{
    id: string
    resourceType: string
    buildingId?: string
    unlockedBy?: string
    unlockedAt?: string
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
  generals?: Array<{
    id: string
    name: string
    level: number
    exp: number
    availableStatPoints?: number
    stats?: Record<string, number>
  }>
  generalAssignments?: Array<{
    id: string
    generalId: string
    slot: string
    moduleId?: string
    status?: string
    assignedAt?: string
    endsAt?: string
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

export interface ItemStack {
  slotId?: string
  itemId: string
  amount: number
  obtainedAt?: string
  updatedAt?: string
}

export interface ItemEffect {
  type: string
  amount?: number
  category?: string
  pool?: string
  resources?: Record<string, number>
  unitByFaction?: Record<string, string>
  protectionType?: string
  durationSeconds?: number
}

export interface ItemDefinition {
  id: string
  name: string
  description: string
  category?: string
  quality?: string
  type?: string
  rarity?: string
  icon?: string
  usable: boolean
  stackable: boolean
  maxStack: number
  bindType?: string
  useTarget: string
  confirmOnUse?: string
  effects: ItemEffect[]
  metadata?: Record<string, unknown>
}

export interface InventoryView {
  inventory: Record<string, ItemStack>
  inventorySlots?: ItemStack[]
  serverTime: string
}

export interface ItemLedgerEntry {
  id: string
  playerId: string
  itemId: string
  changeAmount: number
  beforeAmount: number
  afterAmount: number
  reason: string
  refType?: string
  refId?: string
  createdAt: string
}

export interface ItemLedgerPage {
  entries: ItemLedgerEntry[]
  total: number
  limit: number
  offset: number
}

export interface Reward {
  type: string
  id: string
  amount: number
  metadata?: Record<string, unknown>
}

export interface ReincarnationConfig {
  levels: Array<{
    level: number
    name: string
    wavePowerBase: number
    playerTroopCap: number
    enemyTroopBase: number
    durationSeconds: number
    rewardExpCap: number
    enabled: boolean
  }>
  waves: Array<{
    waveIndex: number
    rewardPreview?: Reward[]
    expBudgetRate: number
    expRandomMin: number
    expRandomMax: number
    fixedRewards: Reward[]
    dropPoolId?: string
  }>
  enemyFactions: string[]
  bonusValues: number[]
  defenseCountdownSeconds: number
}

export interface ReincarnationRun {
  id: string
  playerId: string
  level: number
  levelName: string
  status: string
  currentWave: number
  startedAt: string
  expiresAt: string
  endedReason?: string
  pendingRewards: Reward[]
  rewardGrantedAt?: string
}

export interface ReincarnationRunPage {
  items: ReincarnationRun[]
  total: number
  limit: number
  offset: number
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
  noLossPowerRatioThreshold: number
  equalResult: string
  lossDistribution: string
  defenseFormula: string
}

export interface CombatWallEntry {
  base: number
  hardness?: number
  minDamagedLevelFrom20?: number
  maxDamagedLevelFrom20?: number
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
  traitType?: 'special' | 'bonus'
  enabled: boolean
  scope?: string
  targetUnitType?: string
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
  traits?: GeneralTraitConfig[]
  specialTrait: GeneralTraitConfig
  bonusTrait: GeneralTraitConfig
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
  traitType: 'special' | 'bonus'
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

export interface SlotBonusMultiplierConfig {
  multiplier: number
  weight: number
}

export interface SlotSymbolConfig {
  id: string
  name: string
  rarity: string
  type: 'normal' | 'wild' | 'scatter' | 'bonus' | string
  weight: number
  multiplier?: number
  freeSpins?: number
  retriggerFreeSpins?: number
  bonusMultipliers?: SlotBonusMultiplierConfig[]
}

export interface SlotConfig {
  minLineBet: number
  lineCount: number
  maxFreeSpinsPerRound: number
  symbols: SlotSymbolConfig[]
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

export type AnnouncementType = 'system' | 'maintenance' | 'update' | 'activity' | 'compensation' | 'emergency'

export type AnnouncementStatus = 'draft' | 'scheduled' | 'published' | 'withdrawn' | 'archived'

export type AnnouncementDisplayMode = 'center_only' | 'popup' | 'banner'

export type AnnouncementTargetType = 'all' | 'player_ids' | 'account_ids' | 'factions' | 'level_range' | 'created_at_range'

export interface AnnouncementTarget {
  type: AnnouncementTargetType | string
  value?: unknown
}

export interface Announcement {
  id: string
  title: string
  summary: string
  content?: string
  type: AnnouncementType | string
  status: AnnouncementStatus | string
  displayMode: AnnouncementDisplayMode | string
  pinned: boolean
  priority: number
  forcePopup: boolean
  startsAt?: string
  endsAt?: string
  publishedAt?: string
  withdrawnAt?: string
  archivedAt?: string
  createdAt: string
  updatedAt: string
  targets?: AnnouncementTarget[]
}

export interface SaveAnnouncementPayload {
  title: string
  summary: string
  content: string
  type: string
  status?: string
  displayMode: string
  pinned: boolean
  priority: number
  forcePopup: boolean
  startsAt?: string
  endsAt?: string
  targets: AnnouncementTarget[]
}

export interface AdminAnnouncementPage {
  items: Announcement[]
  page: number
  pageSize: number
  total: number
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

export interface WorldPosition {
  playerId: string
  worldId: string
  x: number
  y: number
  assignedBy: string
  createdAt?: string
  updatedAt?: string
}

export interface WorldMapOccupancyStats {
  worldId: string
  width: number
  height: number
  totalCells: number
  occupiedCells: number
  availableCells: number
  occupancyRate: number
}

export interface WorldMapCoordinateCheck {
  worldId: string
  x: number
  y: number
  occupied: boolean
  playerId?: string
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
  dropPoolId?: string
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
  goldUpgradeCostByLevel?: Record<string, number>
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

export interface PvpPlayerState {
  playerId: string
  status: string
  protectionType?: string
  protectedUntil?: string
  cooldownUntil?: string
  dailyAttackCount: number
  dailyAttackLimit: number
  dailyResetAt?: string
  targetCooldown?: Record<string, string>
  metadata?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface PvpRevengeRecord {
  id: string
  defenderPlayerId: string
  attackerPlayerId: string
  marchId: string
  battleId?: string
  status: string
  createdAt: string
  expiresAt: string
  closedAt?: string
}

export interface PvpStateResponse {
  state: PvpPlayerState
  seasonPoints: number
  rating: number
  attackWins: number
  defenseWins: number
  losses: number
  revengeRecords: PvpRevengeRecord[]
  serverTime: string
}

export interface PvpSeasonSummary {
  id: string
  name: string
  status: string
  startsAt: string
  endsAt: string
  updatedAt: string
}

export interface PvpSeasonRecord extends PvpSeasonSummary {
  settledAt?: string
  rules?: Record<string, unknown>
  rewards?: Record<string, unknown>
  createdAt: string
}

export interface PvpSeasonPlayerRecord {
  seasonId: string
  playerId: string
  nickname?: string
  faction?: string
  rank: number
  points: number
  rating: number
  wins: number
  losses: number
  defenseWins: number
  defenseLosses: number
  lastBattleAt?: string
  rewardMailId?: string
  rewardSentAt?: string
  createdAt: string
  updatedAt: string
}

export interface PvpRankingEntry {
  rank: number
  playerId: string
  nickname: string
  faction: string
  points: number
  rating: number
  attackWins: number
  defenseWins: number
  losses: number
  updatedAt: string
}

export interface PvpMarch {
  id: string
  attackerPlayerId: string
  attackerName: string
  attackerFaction: string
  defenderPlayerId: string
  defenderName: string
  defenderFaction: string
  marchType: string
  status: string
  attackTroops: Record<string, number>
  attackGenerals?: string[]
  speedMultiplier: number
  durationSeconds: number
  startedAt: string
  arrivesAt: string
  returnStartedAt?: string
  returnsAt?: string
  resolvedAt?: string
  attackerReportId?: string
  defenderReportId?: string
  battleId?: string
  acceleratedTimes: number
  createdAt: string
  updatedAt: string
}

export interface PvpMarchActionResponse {
  march: PvpMarch
  army?: Array<{
    unitType: string
    amount: number
  }>
  generals?: Array<{
    id: string
    name: string
    level: number
  }>
  cityGold?: number
  cost?: number
  serverTime: string
}

export interface PvpBattle {
  id: string
  marchId: string
  attackerPlayerId: string
  defenderPlayerId: string
  status: string
  result?: Record<string, unknown>
  losses?: Record<string, unknown>
  plunder?: Record<string, number>
  attackerReportId?: string
  defenderReportId?: string
  resolvedAt?: string
  createdAt: string
  updatedAt: string
}

export interface AdminPvpOverviewResponse {
  playerId?: string
  player?: PvpStateResponse
  season: PvpSeasonSummary
  rankings: PvpRankingEntry[]
  marches: PvpMarch[]
  battles: PvpBattle[]
  serverTime: string
}

export interface AdminPvpSeasonListResponse {
  current: PvpSeasonSummary
  seasons: PvpSeasonRecord[]
  serverTime: string
}

export interface AdminSavePvpSeasonRequest {
  id?: string
  name: string
  status?: string
  startsAt: string
  endsAt: string
  rules?: Record<string, unknown>
  rewards?: Record<string, unknown>
}

export interface AdminSettlePvpSeasonResponse {
  season: PvpSeasonRecord
  players: PvpSeasonPlayerRecord[]
  rewardMail: number
  serverTime: string
}
