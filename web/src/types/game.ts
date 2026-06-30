/* 最小可用类型定义 - 随后端接口稳定后逐步扩展 */

export interface Player {
  id: string
  nickname: string
  faction: string
  mailCode?: string
}

export interface AccountSession {
  accountId: string
  username: string
  gold: number
  token?: string
}

export interface PlayerSummary {
  id: string
  nickname: string
  faction: string
  mailCode?: string
  totalArmy: number
  buildingLevel: number
  updatedAt: string
}

export type AnnouncementType = 'system' | 'maintenance' | 'update' | 'activity' | 'compensation' | 'emergency'

export interface AnnouncementSummary {
  id: string
  title: string
  summary: string
  type: AnnouncementType | string
  status: string
  displayMode: 'center_only' | 'popup' | 'banner' | string
  pinned: boolean
  priority: number
  forcePopup: boolean
  publishedAt?: string
  startsAt?: string
  endsAt?: string
  isRead: boolean
  isPopupShown: boolean
  isDismissed: boolean
}

export interface AnnouncementDetail extends AnnouncementSummary {
  content: string
}

export interface AnnouncementPage {
  items: AnnouncementSummary[]
  page: number
  pageSize: number
  total: number
  unread: boolean
}

export interface AnnouncementReadState {
  announcementId: string
  playerId: string
  accountId?: string
  isRead: boolean
  readAt?: string
  isPopupShown: boolean
  popupShownAt?: string
  isDismissed: boolean
  dismissedAt?: string
  createdAt: string
  updatedAt: string
}

export interface ResourceState {
  items: Record<string, number>
  capacity: Record<string, number>
}

export type ResourceProduction = Record<string, number>

export interface CityActionResult {
  building?: Building
  buildings?: Building[]
  resourceSlots?: ResourceSlot[]
  resources: ResourceState
  resourceProduction: ResourceProduction
  cityGold: number
  activeModifiers?: ModifierBreakdownItem[]
  upgraded?: number
  cost?: number
  serverTime: string
}

export interface ResourceActionResult {
  resources: ResourceState
  resourceProduction: ResourceProduction
  resourceSettledAt: string
  productionBoost?: number
  productionBoostEnd?: string
  capacityBoost?: number
  capacityBoostEnd?: string
  activeModifiers?: ModifierBreakdownItem[]
  cityGold: number
  cost?: number
  serverTime: string
}

export interface MilitaryActionResult {
  army: ArmyUnit[]
  recruitQueues: RecruitQueue[]
  resources: ResourceState
  cityGold: number
  serverTime: string
}

export interface GeneralViewActionResult {
  general?: General
  generals?: General[]
  generalAssignments?: GeneralAssignment[]
  generalChangeUntil?: string
  activeModifiers?: ModifierBreakdownItem[]
  accountGold: number
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
  type: 'general_exp' | 'resources' | 'unit_by_faction' | string
  amount?: number
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
  confirmOnUse?: 'auto' | 'always' | 'never' | string
  effects: ItemEffect[]
  metadata?: Record<string, unknown>
}

export interface Building {
  id: string
  type: string
  level: number
  status?: string
  upgradeEndsAt: string | null
  statusEndsAt?: string | null
}

export interface ResourceSlot {
  id: string
  resourceType: string
  buildingId?: string
  unlockedBy?: string
  unlockedAt?: string
}

export interface ArmyUnit {
  unitType: string
  amount: number
}

export interface Reward {
  type: string
  id: string
  amount: number
  metadata?: Record<string, unknown>
}

export interface ReincarnationLevelConfig {
  level: number
  name: string
  wavePowerBase: number
  playerTroopCap: number
  enemyTroopBase: number
  durationSeconds: number
  rewardExpCap: number
  enabled: boolean
}

export interface ReincarnationConfig {
  levels: ReincarnationLevelConfig[]
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
  bonusResetGoldCost: number
}

export interface ReincarnationBonus {
  side: string
  unitType: string
  stat: string
  value: number
  label: string
  unitName?: string
  faction?: string
}

export interface ReincarnationWave {
  id: string
  runId: string
  waveIndex: number
  waveType: 'attack' | 'defense' | string
  enemyFaction: string
  enemyTroops: Record<string, number>
  enemyRemaining: Record<string, number>
  allyBonus: ReincarnationBonus
  enemyBonus: ReincarnationBonus
  rewardPreview: Reward[]
  rewardResult?: Reward[]
  troopCap: number
  status: string
  startedAt: string
  clearedAt?: string
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
  completedAt?: string
  failedAt?: string
  endedReason?: string
  pendingRewards: Reward[]
  rewardGrantedAt?: string
  waves: ReincarnationWave[]
  createdAt: string
  updatedAt: string
}

export interface ReincarnationRunResponse {
  run?: ReincarnationRun
  army?: ArmyUnit[]
  serverTime: string
}

export interface ReincarnationActionResult {
  run: ReincarnationRun
  battleReport?: BattleReport
  army?: ArmyUnit[]
  inventory?: Record<string, ItemStack>
  inventorySlots?: ItemStack[]
  general?: General
  generals?: General[]
  accountGold?: number
  cost?: number
  serverTime: string
}

export interface RecruitQueue {
  id: string
  unitType: string
  amount: number
  endsAt: string
}

export interface MapTarget {
  id: string
  type: string
  level: number
  power: number
  rewards: Record<string, number>
}

export interface BattleReport {
  id: string
  eventId?: string
  playerId: string
  ownerPlayerId?: string
  viewType?: 'attack' | 'defense' | 'reinforcement' | 'scout' | 'system' | string
  sourceType?: 'npc_city' | 'player_city' | 'stronghold' | 'dungeon' | 'resource_point' | 'event_target' | 'world_boss' | 'system' | string
  battleType?: string
  title?: string
  summary?: string
  detail?: BattleReportDetailData
  share?: BattleReportShare
  playerFaction?: string
  playerName?: string
  targetId: string
  targetName: string
  type: 'attack' | 'plunder' | 'scout' | 'reinforce' | 'defense' | string
  result: 'attacker_victory' | 'defender_victory' | 'draw'
  playerPower: number
  enemyPower: number
  dispatchedUnits: Record<string, number>
  lostUnits: Record<string, number>
  defenderFaction: string
  defenderUnits: Record<string, number>
  defenderLostUnits: Record<string, number>
  defenderRevealed: boolean
  defenderResources: Record<string, number>
  rewards: Record<string, number>
  drops?: Array<{ type: string; itemId?: string; name?: string; amount: number; quality?: string }>
  overflow?: Record<string, number>
  overflowCityGold?: number
  generalExpGained?: number
  generalLevelBefore?: number
  generalLevelAfter?: number
  capturedUnits?: Record<string, number>      // 美人计俘虏到军队
  capturedToGarrison?: Record<string, number> // 美人计俘虏到驻防
  revivedUnits?: Record<string, number>       // 仁德复活
  traitTriggered?: string[]                   // 触发的特性 id 列表
  traitOutcomes?: Record<string, {            // 特性触发结果详情
    traitId: string
    name?: string
    detail?: Record<string, number | string | Record<string, number>>
  }>
  pvpPointsDelta?: Record<string, number>
  pvpAttackerGenerals?: PvpGeneralSnapshot[]
  pvpDefenderGenerals?: PvpGeneralSnapshot[]
  pvpReinforcements?: DefenseReinforcementUnit[]
  pvpReinforcementLosses?: Record<string, Record<string, number>>
  read: boolean
  createdAt: string
}

export interface BattleReportShare {
  id?: string
  reportId?: string
  token?: string
  url?: string
  visibility?: string
  expiresAt?: string
  createdAt?: string
}

export interface BattleReportDetailData {
  id: string
  eventId?: string
  ownerPlayerId?: string
  viewType: string
  viewLabel: string
  sourceType: string
  sourceLabel: string
  battleType: string
  result: string
  title: string
  summary?: string
  occurredAt: string
  primarySide: BattleReportSide
  secondarySide?: BattleReportSide
  rewards: BattleReportRewards
  traits?: BattleReportTrait[]
  visibility: BattleReportVisibility
  extra?: BattleReportExtra
  read: boolean
  share?: BattleReportShare
}

export interface BattleReportExtra {
  sweep?: BattleReportSweepExtra
  [key: string]: unknown
}

export interface BattleReportSweepExtra {
  requested?: number
  success?: number
  failed?: number
  stopped?: boolean
  mode?: string
}

export interface BattleReportSide {
  role: string
  playerId?: string
  playerName?: string
  cityId?: string
  cityName?: string
  faction?: string
  factionLabel?: string
  targetType?: string
  targetId?: string
  targetName?: string
  level?: number
  power: number
  generals?: BattleReportGeneral[]
  units: BattleReportUnit[]
  resources?: Record<string, number>
}

export interface BattleReportUnit {
  unitType: string
  unitName?: string
  faction?: string
  amountBefore: number
  dispatched: number
  lost: number
  survived: number
}

export interface BattleReportGeneral {
  id: string
  name?: string
  level?: number
  role?: string
  power?: number
  attributes?: Record<string, number>
}

export interface BattleReportTrait {
  traitId: string
  traitName?: string
  ownerSide?: string
  ownerRole?: string
  generalId?: string
  generalName?: string
  summary?: string
  detail?: Record<string, number | string | Record<string, number>>
}

export interface BattleReportRewards {
  resources?: Record<string, number>
  drops?: Array<{ type: string; itemId?: string; name?: string; amount: number; quality?: string }>
  cityGold?: number
  generalExp?: number
  generalLevelBefore?: number
  generalLevelAfter?: number
  overflow?: Record<string, number>
}

export interface BattleReportVisibility {
  showEnemyRemainingUnits: boolean
  showEnemyResources: boolean
  showEnemyGenerals: boolean
  showEnemyCityDefense: boolean
  reason?: string
  threshold?: number
  actualLossRatio?: number
}

export interface MailAttachment {
  type: string
  itemId: string
  amount: number
  metadata?: Record<string, unknown>
}

export interface Mail {
  id: string
  playerId: string
  mailType: 'gm_notice' | 'compensation' | 'reward' | 'event_reward' | 'system_notice' | 'player_message' | 'pvp_season_reward' | 'server_broadcast' | string
  senderType: 'system' | 'gm' | 'player'
  senderId?: string
  senderName: string
  title: string
  content: string
  attachments?: MailAttachment[]
  sourceType?: string
  sourceId?: string
  isRead: boolean
  isClaimed: boolean
  deletedByPlayer?: boolean
  expiresAt?: string
  createdAt: string
  readAt?: string
  claimedAt?: string
}

export interface MailClaimResult {
  mail: Mail
  resources: ResourceState
  inventory?: Record<string, ItemStack>
  cityGold: number
  accountGold?: number
  grantedItems: Record<string, number>
}

export interface ServerBroadcastMailResult {
  cost: number
  cityGold: number
  recipientCount: number
  serverTime: string
}

export interface MiniGameRecord {
  id: string
  playerId: string
  gameType: 'fishing' | 'gambling' | string
  resultName: string
  rarity: 'common' | 'rare' | 'epic' | 'legendary' | string
  rewardUnit: string
  rewardAmount: number
  remainingAmount: number
  betUnit?: string
  betAmount?: number
  createdAt: string
}

export interface MiniGameSummary {
  totalRecords: number
  limit: number
  offset: number
  hasMore: boolean
  records: MiniGameRecord[]
  rewardTotals: Record<string, number>
}

export interface MiniGameRedeemResult {
  record: MiniGameRecord
  army?: ArmyUnit[]
  serverTime: string
  redeemedUnitId: string
  redeemedUnit: string
  redeemedAmount: number
  redeemedTarget: 'army' | 'garrison'
  garrison?: Reinforcement
}

export interface MiniGameRedeemAllResult {
  army?: ArmyUnit[]
  serverTime: string
  redeemedUnits: Record<string, number>
  redeemedAmount: number
  redeemedRecords: number
  garrisonedUnits?: Record<string, number>
  garrisonRecords?: number
  skippedUnits: Record<string, number>
  skippedRecords: number
}

export interface FishingBaitUseResult {
  baitId: string
  cityGold?: number
  serverTime: string
  cityGoldCost: number
  cityGoldRemain?: number
}

export interface CurrencyActionResult {
  cityGold: number
  accountGold?: number
  lastExchangeAt?: string
  serverTime: string
}

export interface ReportActionResult {
  unreadMessageCount: number
  serverTime: string
}

export interface ItemActionResult {
  inventory?: Record<string, ItemStack>
  inventorySlots?: ItemStack[]
  resources?: ResourceState
  army?: ArmyUnit[]
  general?: General
  generals?: General[]
  generalAssignments?: GeneralAssignment[]
  activeModifiers?: ModifierBreakdownItem[]
  buffs?: Buff[]
  cityGold: number
  serverTime: string
}

export interface UseItemResult {
  patch: ItemActionResult
  itemId: string
  used: number
  effects: Record<string, number>
}

export interface GarrisonActionResult {
  army?: ArmyUnit[]
  generals?: General[]
  generalAssignments?: GeneralAssignment[]
  serverTime: string
}

export interface ReinforcementGeneralSnapshot {
  id: string
  name?: string
  level?: number
  buffs?: Record<string, number>
  assignment?: string
}

export interface GarrisonRules {
  canRecall: boolean
  canExpel: boolean
  canReturn: boolean
  canFight: boolean
  canConvert: boolean
  canRelease: boolean
}

export interface Reinforcement {
  reinforcementId: string
  fromPlayerId: string
  fromPlayerName?: string
  fromPlayerFaction?: string
  toPlayerId: string
  toPlayerName?: string
  toPlayerFaction?: string
  ownerPlayerId?: string
  hostPlayerId?: string
  sourceType?: 'reinforcement' | 'obtained' | 'captured' | 'mercenary' | 'event_reward' | 'system'
  sourceId?: string
  targetType: string
  targetId: string
  status: 'marching' | 'stationed' | 'fighting' | 'returning' | 'completed' | 'cancelled' | 'failed'
  troops: Record<string, number>
  remainingTroops: Record<string, number>
  generals?: ReinforcementGeneralSnapshot[]
  losses?: Record<string, number>
  rules: GarrisonRules
  speedMultiplier: number
  marchSeconds: number
  returnSeconds: number
  sentAt: string
  arriveAt?: string
  arrivedAt?: string
  recalledAt?: string
  expelledAt?: string
  returnStartedAt?: string
  expectedReturnedAt?: string
  returnedAt?: string
  lastBattleReportId?: string
  lastBattleAt?: string
  isAnnihilated: boolean
  createdAt: string
  updatedAt: string
}

export interface ReinforcementListResponse {
  items: Reinforcement[]
}

export interface ReinforcementResponse {
  reinforcement: Reinforcement
  patch?: GarrisonActionResult
}

export interface DefenseReinforcementUnit {
  reinforcementId: string
  fromPlayerId: string
  faction: string
  troops: Record<string, number>
  generals?: ReinforcementGeneralSnapshot[]
  buffs?: ModifierBreakdownItem[]
  sourceTags?: Record<string, string>
}

export interface PvpWorldPosition {
  worldId: string
  x: number
  y: number
  regionId: number
}

export interface PvpTargetSummary {
  playerId: string
  nickname: string
  faction: string
  position: PvpWorldPosition
  distance: number
  totalArmy: number
  buildingLevel: number
  canAttack: boolean
  canReinforce: boolean
  protected: boolean
  protectedUntil?: string
  cooldownUntil?: string
  reason?: string
}

export interface PvpTargetsResponse {
  items: PvpTargetSummary[]
  self: PvpWorldPosition
  worldSize: number
  centerX: number
  centerY: number
  radius: number
}

export interface PvpMarch {
  id: string
  attackerPlayerId: string
  attackerName: string
  attackerFaction: string
  defenderPlayerId: string
  defenderName: string
  defenderFaction: string
  marchType: 'attack' | 'plunder'
  status: 'marching' | 'returning' | 'resolving' | 'resolved' | 'recalled' | 'cancelled' | 'failed'
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

export interface PvpBattle {
  id: string
  marchId: string
  attackerPlayerId: string
  defenderPlayerId: string
  status: 'created' | 'resolving' | 'resolved' | 'failed'
  attackerSnapshot?: Record<string, unknown>
  defenderSnapshot?: Record<string, unknown>
  reinforcementSnapshot?: DefenseReinforcementUnit[]
  result?: Record<string, unknown>
  losses?: Record<string, unknown>
  plunder?: Record<string, number>
  attackerReportId?: string
  defenderReportId?: string
  resolvedAt?: string
  createdAt: string
  updatedAt: string
}

export interface PvpGeneralSnapshot {
  id: string
  name?: string
  level?: number
  buffs?: Record<string, number>
}

export interface PvpRevengeRecord {
  id: string
  defenderPlayerId: string
  attackerPlayerId: string
  marchId: string
  battleId?: string
  status: 'open' | 'closed' | string
  createdAt: string
  expiresAt: string
  closedAt?: string
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

export interface PvpSeasonResponse {
  season: PvpSeasonSummary
  self?: PvpRankingEntry
  serverTime: string
}

export interface PvpRankingResponse {
  season: PvpSeasonSummary
  items: PvpRankingEntry[]
  self?: PvpRankingEntry
  serverTime: string
}

export interface PvpAttackResponse {
  march: PvpMarch
  army: ArmyUnit[]
  generals?: General[]
  generalAssignments?: GeneralAssignment[]
  serverTime: string
}

export interface PvpMarchActionResponse {
  march: PvpMarch
  army?: ArmyUnit[]
  generals?: General[]
  cityGold?: number
  cost?: number
  serverTime: string
}

export interface GeneralActionResult {
  state: GameState
  accountGold: number
}

export interface GeneralTraitInstance {
  traitId: string
  name: string
  params: Record<string, number>
}

export interface General {
  id: string
  name: string
  level: number
  exp: number
  currentLevelExp?: number
  nextLevelExp?: number
  availableStatPoints?: number
  stats?: Record<string, number>
  attributes?: Record<string, number>
  attributeBreakdown?: Record<string, Array<{ source: string; value: number }>>
  buffs: Record<string, number>
  traits?: GeneralTraitInstance[]
}

export interface GeneralAssignment {
  id: string
  generalId: string
  slot: string
  moduleId?: string
  status?: string
  assignedAt?: string
  endsAt?: string
}

export interface GameState {
  player: Player
  resources: ResourceState
  inventory?: Record<string, ItemStack>
  inventorySlots?: ItemStack[]
  resourceProduction: ResourceProduction
  resourceSettledAt: string
  /** 存档级城金 */
  cityGold: number
  /** 上次兑换时间（冷却用） */
  lastExchangeAt?: string
  /** 产量加成倍率（1=无加成，2/4/8/16） */
  productionBoost?: number
  /** 产量加成到期时间 */
  productionBoostEnd?: string
  /** 仓库容量加成倍率（1=无加成，2/4/8/16） */
  capacityBoost?: number
  /** 容量加成到期时间 */
  capacityBoostEnd?: string
  buildings: Building[]
  resourceSlots?: ResourceSlot[]
  general: General | null
  generals?: General[]
  generalAssignments?: GeneralAssignment[]
  generalChangeUntil?: string
  army: ArmyUnit[]
  recruitQueues: RecruitQueue[]
  npcState?: NpcState | null
  mapTargets: MapTarget[]
  recentBattleReports: BattleReport[]
  unreadMessageCount: number
  unreadMailCount: number
  /** 当前生效的加成明细（用于 tooltip 展示） */
  activeModifiers?: ModifierBreakdownItem[]
  serverTime: string
}

/** 加成明细条目 */
export interface ModifierBreakdownItem {
  source: string   // 来源名称，如 "将领", "购买加成"
  key: string      // 属性键名，如 "productionBonus"
  value: number    // 数值
  mode: string     // "flat" | "percentAdd" | "percentMultiply"
}

export interface Buff {
  id: string
  source: string
  modifierKey: string
  value: number
  mode: string
  startsAt?: string
  endsAt?: string
}

// --- NPC 城池类型 ---

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
  army: ArmyUnit[]
  maxArmy: ArmyUnit[]
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
