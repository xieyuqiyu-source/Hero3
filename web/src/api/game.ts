/* 游戏业务 API */

import { api } from './client'
import type { AccountSession, GameState, BattleReport, PlayerSummary, NpcCity, Mail, MailClaimResult, MiniGameRecord, MiniGameSummary, MiniGameRedeemResult, MiniGameRedeemAllResult, FishingBaitUseResult, ItemDefinition, GeneralActionResult } from '@/types/game'
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
      fishing: FishingConfig
      message: string
    }>('/game/bootstrap')
  },

  /** 获取帮助文档列表 */
  listHelpDocuments() {
    return api.get<{ documents: HelpDocumentSummary[] }>('/help/docs')
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

  /** 将领四维加点 */
  allocateGeneralStat(playerId: string, statKey: string) {
    return api.post<{ state: GameState }>('/military/general/stat', { playerId, statKey })
  },

  /** 将领洗点 */
  resetGeneralStats(playerId: string) {
    return api.post<GeneralActionResult>('/military/general/reset-stats', { playerId })
  },

  /** 更换将领 */
  changeGeneral(playerId: string, generalId: string, itemId?: string) {
    return api.post<GeneralActionResult>('/military/general/change', { playerId, generalId, itemId })
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

  /** 分页获取军情战报 */
  listReports(playerId: string, page: number, pageSize: number) {
    return api.get<BattleReportPage>(`/news/reports?playerId=${playerId}&page=${page}&pageSize=${pageSize}`)
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

  /** 金币兑换城金（1金币=10城金，有冷却） */
  exchangeGold(accountId: string, playerId: string, amount: number) {
    return api.post<{ state: GameState; accountGold: number }>('/gold/exchange', { accountId, playerId, amount })
  },

  /** 城金兑换金币（15城金=1金币，有损耗+冷却） */
  reverseExchangeGold(accountId: string, playerId: string, cityGoldAmount: number) {
    return api.post<{ state: GameState; accountGold: number }>('/gold/reverse-exchange', { accountId, playerId, cityGoldAmount })
  },

  /** 获取物品配置 */
  getItemsConfig() {
    return api.get<Record<string, ItemDefinition>>('/items/config')
  },

  /** 使用物品 */
  useItem(playerId: string, itemId: string, amount = 1) {
    return api.post<{ state: GameState; itemId: string; used: number; effects: Record<string, number> }>('/items/use', {
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
