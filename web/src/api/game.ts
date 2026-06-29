/* 游戏业务 API */

import { api } from './client'
import type { AccountSession, GameState, BattleReport, PlayerSummary, NpcCity, Mail, MailClaimResult, ServerBroadcastMailResult, MiniGameRecord, MiniGameSummary, MiniGameRedeemResult, MiniGameRedeemAllResult, FishingBaitUseResult, ItemDefinition, GeneralViewActionResult, ReinforcementListResponse, ReinforcementResponse, Reinforcement, CityActionResult, ResourceActionResult, MilitaryActionResult, ResourceState, ArmyUnit, General, CurrencyActionResult, ReportActionResult, UseItemResult, AnnouncementPage, AnnouncementDetail, AnnouncementSummary, AnnouncementReadState, PvpTargetsResponse, PvpAttackResponse, PvpMarchActionResponse, PvpMarch, PvpBattle, PvpStateResponse, PvpRevengeRecord, PvpSeasonResponse, PvpRankingResponse, ReincarnationConfig, ReincarnationRunResponse, ReincarnationActionResult } from '@/types/game'
import type { BalanceConfig, FactionConfig, FishingConfig, UnitConfig } from '@/store/configStore'

export interface CombatUnit {
  id: string
  category: string
  count: number
  attack: number
  infantryDefense: number
  cavalryDefense: number
  carryCapacity: number
  upkeep: number
}

export interface CombatArmy {
  faction: string
  units: CombatUnit[]
}

export interface CombatUnitLoss {
  id: string
  count: number
  losses: number
}

export interface CombatResult {
  winner: 'attacker' | 'defender' | 'draw'
  mode: 'attack' | 'plunder'
  attackerLosses: CombatUnitLoss[]
  defenderLosses: CombatUnitLoss[]
  attackerLossRate: number
  defenderLossRate: number
  attackPower: number
  defensePower: number
  survivingCarry: number
}

export interface BattleSimulationResponse {
  result: CombatResult
  attacker: CombatArmy
  defender: CombatArmy
}

export interface BattleReportPage {
  reports: BattleReport[]
  page: number
  pageSize: number
  total: number
}

export interface MailPage {
  mails: Mail[]
  page: number
  pageSize: number
  total: number
  unread: number
}

export interface HelpDocumentSummary {
  id: string
  title: string
  excerpt: string
  updatedAt: string
}

export interface HelpDocument extends HelpDocumentSummary {
  content: string
}

export const gameApi = {
  /** 获取游戏启动配置（含 balance、factions、units） */
  bootstrap() {
    return api.get<{
      gameName: string
      modules: string[]
      balance: BalanceConfig
      factions: Record<string, FactionConfig>
      units: Record<string, Record<string, UnitConfig>>
      items: Record<string, ItemDefinition>
      reincarnation: ReincarnationConfig
      fishing: FishingConfig
      message: string
    }>('/game/bootstrap')
  },

  /** 获取帮助文档列表 */
  listHelpDocuments() {
    return api.get<{ documents: HelpDocumentSummary[] }>('/help/docs')
  },

  /** 获取玩家公告列表 */
  listAnnouncements(playerId: string, params?: { type?: string; includeArchived?: boolean; page?: number; pageSize?: number }) {
    const query = new URLSearchParams({ playerId })
    if (params?.type) query.set('type', params.type)
    if (params?.includeArchived) query.set('includeArchived', 'true')
    if (params?.page) query.set('page', String(params.page))
    if (params?.pageSize) query.set('pageSize', String(params.pageSize))
    return api.get<AnnouncementPage>(`/announcements?${query.toString()}`)
  },

  /** 获取公告详情 */
  getAnnouncement(playerId: string, announcementId: string) {
    return api.get<AnnouncementDetail>(`/announcements/${announcementId}?playerId=${playerId}`)
  },

  /** 标记公告已读 */
  markAnnouncementRead(playerId: string, announcementId: string) {
    return api.post<AnnouncementReadState>(`/announcements/${announcementId}/read`, { playerId })
  },

  /** 记录公告弹窗已展示 */
  markAnnouncementPopupShown(playerId: string, announcementId: string) {
    return api.post<AnnouncementReadState>(`/announcements/${announcementId}/popup-shown`, { playerId })
  },

  /** 关闭公告弹窗 */
  dismissAnnouncement(playerId: string, announcementId: string) {
    return api.post<AnnouncementReadState>(`/announcements/${announcementId}/dismiss`, { playerId })
  },

  /** 获取公告弹窗队列 */
  listAnnouncementPopups(playerId: string) {
    return api.get<{ items: AnnouncementSummary[] }>(`/announcements/popups?playerId=${playerId}`)
  },

  /** 获取单篇帮助文档 */
  getHelpDocument(documentId: string) {
    const encodedId = documentId.split('/').map((part) => encodeURIComponent(part)).join('/')
    return api.get<{ document: HelpDocument }>(`/help/docs/${encodedId}`)
  },

  /** 获取完整游戏状态 */
  getState(playerId: string) {
    return api.get<GameState>(`/game/state?playerId=${playerId}`)
  },

  /** 获取玩家摘要视图 */
  getSummaryView(playerId: string) {
    return api.get<Pick<GameState, 'player' | 'cityGold' | 'unreadMessageCount' | 'unreadMailCount' | 'serverTime'>>(
      `/game/summary?playerId=${playerId}`,
    )
  },

  /** 获取城池视图 */
  getCityView(playerId: string) {
    return api.get<
      Pick<GameState, 'player' | 'buildings' | 'resourceSlots' | 'resources' | 'resourceProduction' | 'cityGold' | 'activeModifiers' | 'serverTime'>
    >(`/city/view?playerId=${playerId}`)
  },

  /** 获取资源视图 */
  getResourceView(playerId: string) {
    return api.get<
      Pick<
        GameState,
        | 'resources'
        | 'resourceProduction'
        | 'resourceSettledAt'
        | 'productionBoost'
        | 'productionBoostEnd'
        | 'capacityBoost'
        | 'capacityBoostEnd'
        | 'activeModifiers'
        | 'serverTime'
      >
    >(`/resources/view?playerId=${playerId}`)
  },

  /** 获取军事视图 */
  getMilitaryView(playerId: string) {
    return api.get<Pick<GameState, 'army' | 'recruitQueues' | 'resources' | 'cityGold' | 'buildings' | 'activeModifiers' | 'general' | 'generals' | 'generalAssignments' | 'serverTime'>>(
      `/military/view?playerId=${playerId}`,
    )
  },

  /** 获取背包视图 */
  getInventoryView(playerId: string) {
    return api.get<Pick<GameState, 'inventory' | 'inventorySlots' | 'serverTime'>>(`/inventory/view?playerId=${playerId}`)
  },

  /** 获取武将视图 */
  getGeneralsView(playerId: string) {
    return api.get<Pick<GameState, 'general' | 'generals' | 'generalAssignments' | 'activeModifiers' | 'serverTime'>>(
      `/generals/view?playerId=${playerId}`,
    )
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
    return api.post<CityActionResult>('/city/buildings/upgrade', { playerId, buildingId })
  },

  /** 一键爆仓（GM免费） */
  fillResources(playerId: string) {
    return api.post<ResourceActionResult>('/city/resources/fill', { playerId })
  },

  /** 一键爆仓（消耗城金） */
  fillResourcesPaid(playerId: string) {
    return api.post<ResourceActionResult>('/city/resources/fill-paid', { playerId })
  },

  /** 一键升级（批量） */
  upgradeBuildingBatch(playerId: string) {
    return api.post<CityActionResult>('/city/buildings/upgrade-batch', { playerId })
  },

  /** 征兵 */
  recruit(playerId: string, unitId: string, amount: number) {
    return api.post<MilitaryActionResult>('/military/recruit', { playerId, unitId, amount })
  },

  /** 极速完成征兵队列 */
  instantCompleteRecruit(playerId: string, queueId: string) {
    return api.post<MilitaryActionResult>('/military/recruit/instant', { playerId, queueId })
  },

  /** 将领四维加点 */
  allocateGeneralStat(playerId: string, statKey: string, amount = 1) {
    return api.post<GeneralViewActionResult>('/military/general/stat', { playerId, statKey, amount })
  },

  /** 将领洗点 */
  resetGeneralStats(playerId: string) {
    return api.post<GeneralViewActionResult>('/military/general/reset-stats', { playerId })
  },

  /** 更换将领 */
  changeGeneral(playerId: string, generalId: string, itemId?: string) {
    return api.post<GeneralViewActionResult>('/military/general/change', { playerId, generalId, itemId })
  },

  /** 发起增援 */
  sendReinforcement(playerId: string, targetPlayerId: string, troops: Record<string, number>, generalIds: string[] = [], speedMultiplier?: number) {
    return api.post<ReinforcementResponse>('/reinforcements', { playerId, targetPlayerId, troops, generalIds, speedMultiplier })
  },

  /** 查看我派出的增援 */
  listSentReinforcements(playerId: string) {
    return api.get<ReinforcementListResponse>(`/reinforcements/sent?playerId=${playerId}`)
  },

  /** 查看我收到的增援 */
  listReceivedReinforcements(playerId: string) {
    return api.get<ReinforcementListResponse>(`/reinforcements/received?playerId=${playerId}`)
  },

  /** 查看单个增援批次 */
  getReinforcement(playerId: string, reinforcementId: string) {
    return api.get<{ reinforcement: Reinforcement }>(`/reinforcements/${reinforcementId}?playerId=${playerId}`)
  },

  /** 召回援军 */
  recallReinforcement(playerId: string, reinforcementId: string) {
    return api.post<ReinforcementResponse>(`/reinforcements/${reinforcementId}/recall`, { playerId })
  },

  /** 遣返援军 */
  expelReinforcement(playerId: string, reinforcementId: string) {
    return api.post<ReinforcementResponse>(`/reinforcements/${reinforcementId}/expel`, { playerId })
  },

  /** 极速完成建筑升级 */
  instantCompleteBuilding(playerId: string, buildingId: string) {
    return api.post<CityActionResult>('/city/buildings/instant', { playerId, buildingId })
  },

  /** 购买产量加成 */
  purchaseBoost(playerId: string, multiplier: number, hours: number) {
    return api.post<ResourceActionResult>('/city/boost', { playerId, multiplier, hours })
  },

  /** 购买仓库容量加成 */
  purchaseCapacityBoost(playerId: string, multiplier: number, hours: number) {
    return api.post<ResourceActionResult>('/city/capacity-boost', { playerId, multiplier, hours })
  },

  /** 获取加成价格表 */
  getBoostPrices() {
    return api.get<Record<string, number>>('/city/boost/prices')
  },

  /** 获取单条玩家战报 */
  getReport(reportId: string, playerId?: string) {
    const query = playerId ? `?playerId=${encodeURIComponent(playerId)}` : ''
    return api.get<BattleReport>(`/reports/${reportId}${query}`)
  },

  /** 通过分享 token 获取公开战报 */
  getSharedReport(token: string) {
    return api.get<BattleReport>(`/reports/shared/${encodeURIComponent(token)}`)
  },

  /** 创建战报分享 token */
  shareReport(playerId: string, reportId: string) {
    return api.post<{ id: string; reportId: string; token: string; visibility: string; expiresAt?: string; createdAt: string }>(
      `/reports/${reportId}/share`,
      { playerId },
    )
  },

  /** 分页获取军情战报 */
  listReports(playerId: string, page: number, pageSize: number, params?: { viewType?: string; sourceType?: string; battleType?: string; result?: string }) {
    const query = new URLSearchParams({ playerId, page: String(page), pageSize: String(pageSize) })
    if (params?.viewType) query.set('viewType', params.viewType)
    if (params?.sourceType) query.set('sourceType', params.sourceType)
    if (params?.battleType) query.set('battleType', params.battleType)
    if (params?.result) query.set('result', params.result)
    return api.get<BattleReportPage>(`/reports?${query.toString()}`)
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
  attackNpc(playerId: string, npcId: string, mode: 'attack' | 'plunder', units: Record<string, number>, generalIds: string[] = []) {
    return api.post<{ battleReport: BattleReport; resources: ResourceState; army: ArmyUnit[]; general?: General; generals?: General[]; cityGold: number; npcState?: GameState['npcState']; serverTime: string }>('/map/npc-cities/attack', {
      playerId, npcId, mode, units, generalIds,
    })
  },

  /** 侦查 NPC 城池 */
  scoutNpc(playerId: string, npcId: string) {
    return api.post<{ success: boolean; battleReport: BattleReport; npcCity: NpcCity | null; army: ArmyUnit[]; npcState?: GameState['npcState']; serverTime: string }>('/map/npc-cities/scout', { playerId, npcId })
  },

  /** 获取轮回绝境配置 */
  getReincarnationConfig() {
    return api.get<ReincarnationConfig>('/dungeons/reincarnation/config')
  },

  /** 获取当前轮回绝境实例 */
  getReincarnationRun(playerId: string) {
    return api.get<ReincarnationRunResponse>(`/dungeons/reincarnation/run?playerId=${playerId}`)
  },

  /** 开启轮回绝境 */
  startReincarnation(playerId: string, level: number) {
    return api.post<ReincarnationActionResult>('/dungeons/reincarnation/start', { playerId, level })
  },

  /** 进攻轮回绝境波次 */
  attackReincarnationWave(playerId: string, waveId: string, troops: Record<string, number>, generalIds: string[] = [], clientActionId?: string) {
    return api.post<ReincarnationActionResult>(`/dungeons/reincarnation/waves/${waveId}/attack`, { playerId, troops, generalIds, clientActionId })
  },

  /** 防守轮回绝境波次 */
  readyReincarnationDefense(playerId: string, waveId: string, troops: Record<string, number>, generalIds: string[] = [], clientActionId?: string) {
    return api.post<ReincarnationActionResult>(`/dungeons/reincarnation/waves/${waveId}/defense-ready`, { playerId, troops, generalIds, clientActionId })
  },

  /** 重置轮回绝境当前波随机加成 */
  resetReincarnationBonus(playerId: string, waveId: string) {
    return api.post<ReincarnationActionResult>(`/dungeons/reincarnation/waves/${waveId}/bonus-reset`, { playerId })
  },

  /** 结算轮回绝境 */
  settleReincarnation(playerId: string) {
    return api.post<ReincarnationActionResult>('/dungeons/reincarnation/settle', { playerId })
  },

  /** 获取轮回绝境战报 */
  listReincarnationReports(playerId: string, page = 1, pageSize = 10) {
    return api.get<BattleReportPage>(`/dungeons/reincarnation/reports?playerId=${playerId}&page=${page}&pageSize=${pageSize}`)
  },

  /** 获取 PVP 玩家目标 */
  listPvpTargets(playerId: string, params?: { centerX?: number; centerY?: number; radius?: number; limit?: number }) {
    const query = new URLSearchParams({ playerId })
    if (params?.centerX) query.set('centerX', String(params.centerX))
    if (params?.centerY) query.set('centerY', String(params.centerY))
    if (params?.radius) query.set('radius', String(params.radius))
    if (params?.limit) query.set('limit', String(params.limit))
    return api.get<PvpTargetsResponse>(`/pvp/targets?${query.toString()}`)
  },

  /** 侦查 PVP 玩家 */
  scoutPvpTarget(playerId: string, targetPlayerId: string) {
    return api.post<{ success: boolean; battleReport: BattleReport; serverTime: string }>('/pvp/scout', { playerId, targetPlayerId })
  },

  /** 发起 PVP 攻击或掠夺行军 */
  startPvpAttack(playerId: string, targetPlayerId: string, marchMode: 'attack' | 'plunder', troops: Record<string, number>, generalIds: string[] = []) {
    return api.post<PvpAttackResponse>('/pvp/attacks', { playerId, targetPlayerId, marchMode, troops, generalIds })
  },

  /** 获取我的 PVP 行军 */
  listPvpMarches(playerId: string) {
    return api.get<{ items: PvpMarch[] }>(`/pvp/marches?playerId=${playerId}`)
  },

  /** 召回 PVP 行军 */
  recallPvpMarch(playerId: string, marchId: string) {
    return api.post<PvpMarchActionResponse>(`/pvp/marches/${marchId}/recall`, { playerId })
  },

  /** 加速 PVP 行军 */
  acceleratePvpMarch(playerId: string, marchId: string) {
    return api.post<PvpMarchActionResponse>(`/pvp/marches/${marchId}/accelerate`, { playerId })
  },

  /** 获取我的 PVP 战斗记录 */
  listPvpBattles(playerId: string) {
    return api.get<{ items: PvpBattle[] }>(`/pvp/battles?playerId=${playerId}`)
  },

  /** 获取单场 PVP 战斗详情 */
  getPvpBattle(playerId: string, battleId: string) {
    return api.get<PvpBattle>(`/pvp/battles/${battleId}?playerId=${playerId}`)
  },

  /** 获取我的 PVP 状态 */
  getPvpState(playerId: string) {
    return api.get<PvpStateResponse>(`/pvp/state?playerId=${playerId}`)
  },

  /** 获取我的 PVP 复仇记录 */
  listPvpRevenge(playerId: string) {
    return api.get<{ items: PvpRevengeRecord[]; serverTime: string }>(`/pvp/revenge?playerId=${playerId}`)
  },

  /** 获取当前 PVP 赛季 */
  getPvpSeason(playerId: string) {
    return api.get<PvpSeasonResponse>(`/pvp/season?playerId=${playerId}`)
  },

  /** 获取当前 PVP 排行榜 */
  listPvpRankings(playerId: string, limit = 20) {
    return api.get<PvpRankingResponse>(`/pvp/rankings?playerId=${playerId}&limit=${limit}`)
  },

  /** 战斗模拟：只计算，不扣兵不保存战报 */
  simulateBattle(payload: {
    playerId: string
    mode: 'attack' | 'plunder'
    attackerFaction: string
    defenderFaction: string
    attackerUnits: Record<string, number>
    defenderUnits: Record<string, number>
    applyAttackerBonuses: boolean
    applyDefenderBonuses: boolean
  }) {
    return api.post<BattleSimulationResponse>('/combat/simulate', payload)
  },

  /** 标记军情已读（传 reportId 标记单条，不传则按视角或全部标记） */
  markReportsRead(playerId: string, reportId?: string, viewType?: string) {
    if (reportId) {
      return api.post<ReportActionResult>(`/reports/${reportId}/read`, { playerId })
    }
    return api.post<ReportActionResult>('/reports/read-all', { playerId, viewType })
  },

  /** 删除单条战报 */
  deleteReport(playerId: string, reportId: string) {
    return api.post<ReportActionResult>(`/reports/${reportId}/delete`, { playerId })
  },

  /** 一键删除指定视角或全部战报 */
  deleteAllReports(playerId: string, viewType?: string) {
    return api.post<ReportActionResult>('/reports/delete-all', { playerId, viewType })
  },

  /** 分页获取信函 */
  listMails(playerId: string, page: number, pageSize: number, mailType?: string) {
    const params = new URLSearchParams({
      playerId,
      page: String(page),
      pageSize: String(pageSize),
    })
    if (mailType && mailType !== 'all') params.set('mailType', mailType)
    return api.get<MailPage>(`/mails?${params.toString()}`)
  },

  /** 获取信函详情并标记已读 */
  getMail(playerId: string, mailId: string) {
    return api.get<Mail>(`/mails/${mailId}?playerId=${playerId}`)
  },

  /** 删除单封信函 */
  deleteMail(playerId: string, mailId: string) {
    return api.post<{ status: string }>(`/mails/${mailId}/delete`, { playerId })
  },

  /** 领取信函附件 */
  claimMailAttachments(playerId: string, mailId: string) {
    return api.post<MailClaimResult>(`/mails/${mailId}/claim`, { playerId })
  },

  /** 玩家互发纯文本信函 */
  sendPlayerMail(senderPlayerId: string, recipient: string, title: string, content: string) {
    return api.post<Mail>('/mails/send-player', {
      senderPlayerId,
      recipient,
      title,
      content,
    })
  },

  /** 玩家消耗城金发送全服喊话信函 */
  sendServerBroadcastMail(senderPlayerId: string, title: string, content: string) {
    return api.post<ServerBroadcastMailResult>('/mails/server-broadcast', {
      senderPlayerId,
      title,
      content,
    })
  },

  /** 金币兑换城金（1金币=10城金，有冷却） */
  exchangeGold(accountId: string, playerId: string, amount: number) {
    return api.post<CurrencyActionResult>('/gold/exchange', { accountId, playerId, amount })
  },

  /** 城金兑换金币（15城金=1金币，有损耗+冷却） */
  reverseExchangeGold(accountId: string, playerId: string, cityGoldAmount: number) {
    return api.post<CurrencyActionResult>('/gold/reverse-exchange', { accountId, playerId, cityGoldAmount })
  },

  /** 获取物品配置 */
  getItemsConfig() {
    return api.get<Record<string, ItemDefinition>>('/items/config')
  },

  /** 使用物品 */
  useItem(playerId: string, itemId: string, amount = 1) {
    return api.post<UseItemResult>('/items/use', {
      playerId,
      itemId,
      amount,
    })
  },

  /** 上报小游戏记录（钓鱼/赌博） */
  saveMiniGameRecord(playerId: string, gameType: string, resultName: string, rarity: string, rewardUnit: string, rewardAmount: number, betUnit?: string, betAmount?: number) {
    return api.post<MiniGameRecord>('/minigame/record', { playerId, gameType, resultName, rarity, rewardUnit, rewardAmount, betUnit: betUnit ?? '', betAmount: betAmount ?? 0 })
  },

  /** 获取自己的小游戏库存/记录 */
  listMiniGameRecords(playerId: string, limit = 100, offset = 0, gameType = '') {
    const typeQuery = gameType ? `&gameType=${encodeURIComponent(gameType)}` : ''
    return api.get<MiniGameSummary>(`/minigame/records?playerId=${playerId}&limit=${limit}&offset=${offset}${typeQuery}`)
  },

  /** 使用钓鱼鱼饵并扣除城金 */
  useFishingBait(playerId: string, baitId: string) {
    return api.post<FishingBaitUseResult>('/minigame/fishing/use-bait', { playerId, baitId })
  },

  /** 兑换小游戏库存奖励 */
  redeemMiniGameReward(playerId: string, recordId: string, amount: number) {
    return api.post<MiniGameRedeemResult>('/minigame/redeem', { playerId, recordId, amount })
  },

  /** 一次性兑换当前阵营的全部小游戏库存奖励 */
  redeemAllMiniGameRewards(playerId: string, gameType = 'fishing') {
    return api.post<MiniGameRedeemAllResult>('/minigame/redeem-all', { playerId, gameType })
  },

}
