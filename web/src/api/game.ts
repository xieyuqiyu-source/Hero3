/* 游戏业务 API */

import { api } from './client'
import type { AccountSession, GameState, BattleReport, PlayerSummary } from '@/types/game'

export const gameApi = {
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

  /** 删除存档 */
  deletePlayer(playerId: string) {
    return api.delete<{ status: string }>(`/players/${playerId}`)
  },

  /** 升级建筑 */
  upgradeBuilding(playerId: string, buildingId: string) {
    return api.post<{ state: GameState }>('/city/buildings/upgrade', { playerId, buildingId })
  },

  /** 一键爆仓 */
  fillResources(playerId: string) {
    return api.post<{ state: GameState }>('/city/resources/fill', { playerId })
  },

  /** 一键升级（批量） */
  upgradeBuildingBatch(playerId: string) {
    return api.post<{ state: GameState; upgraded: number }>('/city/buildings/upgrade-batch', { playerId })
  },

  /** 征兵 */
  recruit(playerId: string, unitType: string, amount: number) {
    return api.post<{ resources: GameState['resources']; army: GameState['army']; recruitQueues: GameState['recruitQueues'] }>(
      '/military/recruit',
      { playerId, unitType, amount },
    )
  },

  /** 攻击地图目标 */
  attackTarget(playerId: string, targetId: string, units: Record<string, number>) {
    return api.post<{ battleReport: BattleReport; resources: GameState['resources']; army: GameState['army'] }>(
      '/map/targets/attack',
      { playerId, targetId, units },
    )
  },

  /** 获取战报列表 */
  getReports(playerId: string) {
    return api.get<{ reports: BattleReport[] }>(`/battle/reports?playerId=${playerId}`)
  },
}
