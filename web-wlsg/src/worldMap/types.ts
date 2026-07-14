/** 世界地图只读接口、目标与前端状态类型。 */
export interface WorldPosition {
  playerId: string
  worldId: string
  x: number
  y: number
  assignedBy: string
  createdAt?: string
  updatedAt?: string
}

export type WorldMapTargetType = 'player_city' | 'yellow_turban'
export type WorldMapRelation = 'self' | 'ally' | 'other'

export interface WorldMapTarget {
  targetType: WorldMapTargetType
  targetId: string
  playerId?: string
  name: string
  faction: string
  relation: WorldMapRelation
  sameAccount: boolean
  level: number
  status: string
  x: number
  y: number
  distance: number
  direction: string
  canScout: boolean
  canAttack: boolean
  canPlunder: boolean
  canReinforce: boolean
  reason?: string
  scoutReason?: string
  attackReason?: string
  plunderReason?: string
  reinforceReason?: string
}

export interface WorldMapViewResponse {
  worldId: string
  width: number
  height: number
  self: WorldPosition
  centerX: number
  centerY: number
  radius: number
  targets: WorldMapTarget[]
  serverTime: string
}
