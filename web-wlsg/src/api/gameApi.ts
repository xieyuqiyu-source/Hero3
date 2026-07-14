/** V0.2 范围内的 Hero3 业务接口适配层。 */
import type { ApiClient } from './client'
import type { AccountInfo, AccountSession, BootstrapResponse, PlayerSummary } from './types'
import type { BattleReportPageResponse, BattleReportState, CityActionResponse, GameStateResponse, MilitaryActionResponse, MilitaryViewResponse, NpcAttackResponse, NpcCommandAction, NpcRefreshResponse, NpcScoutResponse, NpcStateResponse, PvpDispatchResponse, PvpMarchActionResponse, PvpMarchListItem, ReinforcementActionResponse, ReinforcementDispatchResponse, ReinforcementListItem, ReportActionResponse, ResourceActionResponse, WorldMapMarchAction } from '../game/types'
import type { WorldMapViewResponse } from '../worldMap/types'

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
    /** 使用城金立即完成指定建筑队列。 */
    instantCompleteBuilding: (playerId: string, buildingId: string) => client.post<CityActionResponse>('/city/buildings/instant', { playerId, buildingId }),
    /** 使用城金将当前城池全部资源补至容量上限。 */
    fillResourcesPaid: (playerId: string) => client.post<ResourceActionResponse>('/city/resources/fill-paid', { playerId }),
    /** 读取当前玩家的军事局部视图并结算到期征兵队列。 */
    militaryView: (playerId: string, signal?: AbortSignal) => client.get<MilitaryViewResponse>(`/military/view?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 提交真实征兵请求。 */
    recruit: (playerId: string, unitId: string, amount: number) => client.post<MilitaryActionResponse>('/military/recruit', { playerId, unitId, amount }),
    /** 使用城金立即完成指定征兵队列。 */
    instantCompleteRecruit: (playerId: string, queueId: string) => client.post<MilitaryActionResponse>('/military/recruit/instant', { playerId, queueId }),
    /** 读取真实玩家城池与黄巾营地组成的世界地图视野。 */
    worldMapView: (playerId: string, centerX?: number, centerY?: number, signal?: AbortSignal, radius = 18) => {
      const query = new URLSearchParams({ playerId, radius: String(radius) })
      if (Number.isInteger(centerX) && Number.isInteger(centerY)) {
        query.set('centerX', String(centerX))
        query.set('centerY', String(centerY))
      }
      return client.get<WorldMapViewResponse>(`/world-map/view?${query}`, { signal })
    },
    /** 读取当前存档的真实 NPC 城池状态。 */
    npcCities: (playerId: string, signal?: AbortSignal) => client.get<NpcStateResponse>(`/map/npc-cities?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 扣除 100 账户金币并刷新当前存档的 NPC 城池。 */
    refreshNpcCities: (playerId: string) => client.post<NpcRefreshResponse>('/map/npc-cities/refresh', { playerId }),
    /** 即时结算 NPC 攻击或掠夺，不创建行军队列。 */
    attackNpc: (playerId: string, npcId: string, action: Exclude<NpcCommandAction, 'scout'>, units: Record<string, number>, generalIds: string[]) => client.post<NpcAttackResponse>('/map/npc-cities/attack', { playerId, npcId, mode: action, units, generalIds }),
    /** 自动派出当前阵营侦察兵即时侦查 NPC。 */
    scoutNpc: (playerId: string, npcId: string) => client.post<NpcScoutResponse>('/map/npc-cities/scout', { playerId, npcId }),
    /** 自动派出当前阵营全部侦察兵侦查玩家城池。 */
    scoutPvpTarget: (playerId: string, targetPlayerId: string) => client.post<PvpDispatchResponse>('/pvp/scout', { playerId, targetPlayerId }),
    /** 按 attack 或 plunder 模式派出玩家兵力。 */
    startPvpAttack: (playerId: string, targetPlayerId: string, troops: Record<string, number>, generalIds: string[], marchMode: Extract<WorldMapMarchAction, 'attack' | 'plunder'>) => client.post<PvpDispatchResponse>('/pvp/attacks', { playerId, targetPlayerId, troops, generalIds, marchMode }),
    /** 派出真实兵力和可用武将增援玩家城池。 */
    sendReinforcement: (playerId: string, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]) => client.post<ReinforcementDispatchResponse>('/reinforcements', { playerId, targetPlayerId, troops, generalIds }),
    /** 读取当前玩家相关 PVP 行军，前端再筛选本人派出的活动项。 */
    pvpMarches: (playerId: string, signal?: AbortSignal) => client.get<{ items: PvpMarchListItem[] }>(`/pvp/marches?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 使用 10 城金把本人尚在去程的 PVP 行军剩余时间减半。 */
    acceleratePvpMarch: (playerId: string, marchId: string) => client.post<PvpMarchActionResponse>(`/pvp/marches/${encodeURIComponent(marchId)}/accelerate`, { playerId }),
    /** 召回本人仍处于后端允许召回窗口的 PVP 行军。 */
    recallPvpMarch: (playerId: string, marchId: string) => client.post<PvpMarchActionResponse>(`/pvp/marches/${encodeURIComponent(marchId)}/recall`, { playerId }),
    /** 读取当前玩家派出的增援批次。 */
    sentReinforcements: (playerId: string, signal?: AbortSignal) => client.get<{ items: ReinforcementListItem[] }>(`/reinforcements/sent?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 读取正在增援或驻防当前玩家的增援批次。 */
    receivedReinforcements: (playerId: string, signal?: AbortSignal) => client.get<{ items: ReinforcementListItem[] }>(`/reinforcements/received?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 使用 10 城金把本人派出的行军中增援剩余时间减半。 */
    accelerateReinforcement: (playerId: string, reinforcementId: string) => client.post<ReinforcementActionResponse>(`/reinforcements/${encodeURIComponent(reinforcementId)}/accelerate`, { playerId }),
    /** 召回本人派出的活动增援并让其进入返程。 */
    recallReinforcement: (playerId: string, reinforcementId: string) => client.post<ReinforcementDispatchResponse>(`/reinforcements/${encodeURIComponent(reinforcementId)}/recall`, { playerId }),
    /** 按玩家视角或战斗类型分页读取真实军情。 */
    listReports: (playerId: string, page: number, pageSize: number, filter?: { viewType?: string; battleType?: string }, signal?: AbortSignal) => {
      const query = new URLSearchParams({ playerId, page: String(page), pageSize: String(pageSize) })
      if (filter?.viewType) query.set('viewType', filter.viewType)
      if (filter?.battleType) query.set('battleType', filter.battleType)
      return client.get<BattleReportPageResponse>(`/reports?${query}`, { signal })
    },
    /** 读取当前存档有权查看的一条完整战报。 */
    report: (playerId: string, reportId: string, signal?: AbortSignal) => client.get<BattleReportState>(`/reports/${encodeURIComponent(reportId)}?playerId=${encodeURIComponent(playerId)}`, { signal }),
    /** 将当前存档的一条军情标记为已读。 */
    markReportRead: (playerId: string, reportId: string) => client.post<ReportActionResponse>(`/reports/${encodeURIComponent(reportId)}/read`, { playerId }),
    /** 删除当前存档的一条军情。 */
    deleteReport: (playerId: string, reportId: string) => client.post<ReportActionResponse>(`/reports/${encodeURIComponent(reportId)}/delete`, { playerId }),
  }
}

export type GameApi = ReturnType<typeof createGameApi>
