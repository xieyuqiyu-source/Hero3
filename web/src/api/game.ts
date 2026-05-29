/* 游戏业务 API */

import { api } from './client'
import type { AccountSession, GameState, BattleReport, PlayerSummary, NpcCity } from '@/types/game'
import type { BalanceConfig, FactionConfig, UnitConfig } from '@/store/configStore'

export const gameApi = {
  /** 获取游戏启动配置（含 balance、factions、units） */
  bootstrap() {
    return api.get<{
      gameName: string
      modules: string[]
      balance: BalanceConfig
      factions: Record<string, FactionConfig>
      units: Record<string, Record<string, UnitConfig>>
      message: string
    }>('/game/bootstrap')
  },

  /** 获取完整游戏状态 */
  getState(playerId = 'demo-player') {
    return api.get<GameState>(`/game/state?playerId=${playerId}`)
  },

  /** 创建账号绑定的游戏存档 */
  createPlayer(accountId: string, nickname: string, faction: string, generalId?: string) {
    return api.post<{ playerId: string; state: GameState }>('/players/create', {
      accountId,
      nickname,
      faction,
      generalId,
    })
  },

  registerAccount(username: string, password: string) {
    return api.post<AccountSession>('/accounts/register', { username, password })
  },

  loginAccount(username: string, password: string) {
    return api.post<AccountSession>('/accounts/login', { username, password })
  },

  listAccountPlayers(accountId: string) {
    return api.get<{ players: PlayerSummary[] }>(`/accounts/${accountId}/players`)
  },

  /** 获取账户信息（含最新金币） */
  getAccountInfo(accountId: string) {
    return api.get<AccountSession>(`/accounts/${accountId}`)
  },

  /** 删除存档 */
  deletePlayer(playerId: string) {
    return api.delete<{ status: string }>(`/players/${playerId}`)
  },

  /** 升级建筑 */
  upgradeBuilding(playerId: string, buildingId: string) {
    return api.post<{ state: GameState }>('/city/buildings/upgrade', { playerId, buildingId })
  },

  /** 一键爆仓（GM免费） */
  fillResources(playerId: string) {
    return api.post<{ state: GameState }>('/city/resources/fill', { playerId })
  },

  /** 一键爆仓（消耗城金） */
  fillResourcesPaid(playerId: string) {
    return api.post<{ state: GameState; cost: number }>('/city/resources/fill-paid', { playerId })
  },

  /** 一键升级（批量） */
  upgradeBuildingBatch(playerId: string) {
    return api.post<{ state: GameState; upgraded: number }>('/city/buildings/upgrade-batch', { playerId })
  },

  /** 征兵 */
  recruit(playerId: string, unitId: string, amount: number) {
    return api.post<{ state: GameState }>('/military/recruit', { playerId, unitId, amount })
  },

  /** 极速完成征兵队列 */
  instantCompleteRecruit(playerId: string, queueId: string) {
    return api.post<{ state: GameState }>('/military/recruit/instant', { playerId, queueId })
  },

  /** 极速完成建筑升级 */
  instantCompleteBuilding(playerId: string, buildingId: string) {
    return api.post<{ state: GameState }>('/city/buildings/instant', { playerId, buildingId })
  },

  /** 购买产量加成 */
  purchaseBoost(playerId: string, multiplier: number, hours: number) {
    return api.post<{ state: GameState }>('/city/boost', { playerId, multiplier, hours })
  },

  /** 购买仓库容量加成 */
  purchaseCapacityBoost(playerId: string, multiplier: number, hours: number) {
    return api.post<{ state: GameState }>('/city/capacity-boost', { playerId, multiplier, hours })
  },

  /** 获取加成价格表 */
  getBoostPrices() {
    return api.get<Record<string, number>>('/city/boost/prices')
  },

  /** 获取单条战报（公开，用于分享） */
  getReport(reportId: string) {
    return api.get<BattleReport>(`/reports/${reportId}`)
  },

  /** 攻击地图目标 */
  attackTarget(playerId: string, targetId: string, units: Record<string, number>) {
    return api.post<{ battleReport: BattleReport; resources: GameState['resources']; army: GameState['army'] }>(
      '/map/targets/attack',
      { playerId, targetId, units },
    )
  },

  /** 获取 NPC 城池列表 */
  getNpcCities(playerId: string) {
    return api.get<{ cities: NpcCity[]; lastRefreshedAt: string }>(`/map/npc-cities?playerId=${playerId}`)
  },

  /** 手动刷新 NPC 城池 */
  refreshNpcCities(playerId: string) {
    return api.post<{ cities: NpcCity[]; lastRefreshedAt: string }>('/map/npc-cities/refresh', { playerId })
  },

  /** 攻击 NPC 城池 */
  attackNpc(playerId: string, npcId: string, mode: 'attack' | 'plunder', units: Record<string, number>) {
    return api.post<{ battleReport: BattleReport; state: GameState }>('/map/npc-cities/attack', {
      playerId, npcId, mode, units,
    })
  },

  /** 侦查 NPC 城池 */
  scoutNpc(playerId: string, npcId: string) {
    return api.post<{ success: boolean; battleReport: BattleReport; npcCity: NpcCity | null; state: GameState }>('/map/npc-cities/scout', { playerId, npcId })
  },

  /** 标记军情已读（传 reportId 标记单条，不传标记全部） */
  markReportsRead(playerId: string, reportId?: string) {
    return api.post<{ state: GameState }>('/news/mark-read', { playerId, reportId })
  },

  /** 删除单条战报 */
  deleteReport(playerId: string, reportId: string) {
    return api.post<{ state: GameState }>('/news/delete-report', { playerId, reportId })
  },

  /** 一键删除所有战报 */
  deleteAllReports(playerId: string) {
    return api.post<{ state: GameState }>('/news/delete-all-reports', { playerId })
  },

  /** 金币兑换城金（1金币=10城金，有冷却） */
  exchangeGold(accountId: string, playerId: string, amount: number) {
    return api.post<{ state: GameState; accountGold: number }>('/gold/exchange', { accountId, playerId, amount })
  },

  /** 城金兑换金币（15城金=1金币，有损耗+冷却） */
  reverseExchangeGold(accountId: string, playerId: string, cityGoldAmount: number) {
    return api.post<{ state: GameState; accountGold: number }>('/gold/reverse-exchange', { accountId, playerId, cityGoldAmount })
  },

  /** 上报小游戏记录（钓鱼/赌博） */
  saveMiniGameRecord(playerId: string, gameType: string, resultName: string, rarity: string, rewardUnit: string, rewardAmount: number, betUnit?: string, betAmount?: number) {
    return api.post<{ id: string }>('/minigame/record', { playerId, gameType, resultName, rarity, rewardUnit, rewardAmount, betUnit: betUnit ?? '', betAmount: betAmount ?? 0 })
  },

}
