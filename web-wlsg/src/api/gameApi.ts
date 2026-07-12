/** V0.2 范围内的 Hero3 业务接口适配层。 */
import type { ApiClient } from './client'
import type { AccountInfo, AccountSession, BootstrapResponse, PlayerSummary } from './types'
import type { CityActionResponse, GameStateResponse, MilitaryActionResponse, MilitaryViewResponse, ResourceActionResponse } from '../game/types'

/** 基于统一客户端创建登录选档 API。 */
export function createGameApi(client: ApiClient) {
  return {
    /** 加载公共启动配置。 */
    bootstrap: () => client.get<BootstrapResponse>('/game/bootstrap'),
    /** 登录现有账号。 */
    login: (username: string, password: string) => client.post<AccountSession>('/accounts/login', { username, password }),
    /** 验证会话并读取账号信息。 */
    accountInfo: (accountId: string) => client.get<AccountInfo>(`/accounts/${encodeURIComponent(accountId)}`),
    /** 获取当前账号拥有的存档。 */
    players: (accountId: string) => client.get<{ players: PlayerSummary[] }>(`/accounts/${encodeURIComponent(accountId)}/players`),
    /** 读取当前玩家完整游戏状态。 */
    gameState: (playerId: string, signal?: AbortSignal) => client.get<GameStateResponse>(`/game/state?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 启动指定建筑的下一级建造。 */
    upgradeBuilding: (playerId: string, buildingId: string) => client.post<CityActionResponse>('/city/buildings/upgrade', { playerId, buildingId }),
    /** 使用城金将当前城池全部资源补至容量上限。 */
    fillResourcesPaid: (playerId: string) => client.post<ResourceActionResponse>('/city/resources/fill-paid', { playerId }),
    /** 读取当前玩家的军事局部视图并结算到期征兵队列。 */
    militaryView: (playerId: string, signal?: AbortSignal) => client.get<MilitaryViewResponse>(`/military/view?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 提交真实征兵请求。 */
    recruit: (playerId: string, unitId: string, amount: number) => client.post<MilitaryActionResponse>('/military/recruit', { playerId, unitId, amount }),
    /** 使用城金立即完成指定征兵队列。 */
    instantCompleteRecruit: (playerId: string, queueId: string) => client.post<MilitaryActionResponse>('/military/recruit/instant', { playerId, queueId }),
  }
}

export type GameApi = ReturnType<typeof createGameApi>
