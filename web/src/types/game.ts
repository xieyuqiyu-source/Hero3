/* 最小可用类型定义 - 随后端接口稳定后逐步扩展 */

export interface Player {
  id: string
  nickname: string
  faction: string
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
  totalArmy: number
  buildingLevel: number
  updatedAt: string
}

export interface ResourceState {
  items: Record<string, number>
  capacity: Record<string, number>
}

export type ResourceProduction = Record<string, number>

export interface Building {
  id: string
  type: string
  level: number
  upgradeEndsAt: string | null
}

export interface ArmyUnit {
  unitType: string
  amount: number
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
  playerId: string
  playerFaction?: string
  playerName?: string
  targetId: string
  targetName: string
  type: 'attack' | 'plunder' | 'scout' | 'reinforce'
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
  overflow?: Record<string, number>
  overflowCityGold?: number
  capturedUnits?: Record<string, number>      // 美人计俘虏到军队
  capturedToGarrison?: Record<string, number> // 美人计俘虏到驻防
  revivedUnits?: Record<string, number>       // 仁德复活
  traitTriggered?: string[]                   // 触发的特性 id 列表
  traitOutcomes?: Record<string, {            // 特性触发结果详情
    traitId: string
    name?: string
    detail?: Record<string, number | string>
  }>
  read: boolean
  createdAt: string
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
  buffs: Record<string, number>
  traits?: GeneralTraitInstance[]
}

export interface GameState {
  player: Player
  resources: ResourceState
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
  general: General | null
  army: ArmyUnit[]
  recruitQueues: RecruitQueue[]
  npcState?: NpcState | null
  mapTargets: MapTarget[]
  recentBattleReports: BattleReport[]
  unreadMessageCount: number
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
