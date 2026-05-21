/* 游戏业务 API */

import { api } from './client'
import type { GameState, BattleReport } from '@/types/game'

export const gameApi = {
  /** 获取完整游戏状态 */
  getState(playerId = 'demo-player') {
    return api.get<GameState>(`/game/state?playerId=${playerId}`)
  },

  /** 创建玩家 */
  createPlayer(nickname: string, faction: string) {
    return api.post<{ playerId: string; state: GameState }>('/player/create', {
      nickname,
      faction,
    })
  },

  /** 升级建筑 */
  upgradeBuilding(playerId: string, buildingType: string) {
    return api.post<{ resources: GameState['resources']; buildings: GameState['buildings'] }>(
      '/city/buildings/upgrade',
      { playerId, buildingType },
    )
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
