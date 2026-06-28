/* 本文件封装 GM 后台调用后端管理接口的请求方法。 */
import type {
  AccountSummary,
  AdminSavePvpSeasonRequest,
  AdminPvpSeasonListResponse,
  AdminSettlePvpSeasonResponse,
  AdminPvpOverviewResponse,
  AdminAnnouncementPage,
  Announcement,
  BalanceConfig,
  CombatConfig,
  FactionsConfig,
  FishingConfig,
  GameState,
  GeneralsConfig,
  GoldLedgerEntry,
  HealthState,
  InventoryView,
  ItemDefinition,
  ItemLedgerPage,
  Mail,
  MailAttachment,
  MailPage,
  NpcConfig,
  NpcState,
  PvpBattle,
  PvpMarchActionResponse,
  PvpStateResponse,
  SaveAnnouncementPayload,
  TraitRegistryResponse,
  UnitsConfig,
} from '@/types'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api/v1'
const ROOT_BASE = API_BASE.replace(/\/api\/v1$/, '')

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const adminToken = localStorage.getItem('hero3_admin_token') ?? import.meta.env.VITE_ADMIN_TOKEN ?? ''
  const headers: Record<string, string> = {
    ...(init?.headers as Record<string, string> ?? {}),
  }
  if (adminToken) {
    headers['X-Admin-Token'] = adminToken
  }

  const response = await fetch(url, { ...init, headers })
  if (!response.ok) {
    const body = await response.json().catch(() => null)
    const message = body && typeof body === 'object' && 'error' in body
      ? String((body as { error: string }).error)
      : `HTTP ${response.status}`
    throw new Error(message)
  }

  return response.json() as Promise<T>
}

export const adminApi = {
  getHealth() {
    return request<HealthState>(`${ROOT_BASE}/healthz`)
  },
  getAccounts() {
    return request<{ accounts: AccountSummary[] }>(`${API_BASE}/admin/accounts`)
  },
  getPlayerState(playerId: string) {
    return request<GameState>(`${API_BASE}/admin/players/${playerId}/state`)
  },
  getPvpOverview(playerId = '', limit = 100) {
    const params = new URLSearchParams()
    if (playerId) params.set('playerId', playerId)
    if (limit > 0) params.set('limit', String(limit))
    const query = params.toString()
    return request<AdminPvpOverviewResponse>(`${API_BASE}/admin/pvp/overview${query ? `?${query}` : ''}`)
  },
  getPvpSeasons() {
    return request<AdminPvpSeasonListResponse>(`${API_BASE}/admin/pvp/seasons`)
  },
  createPvpSeason(payload: AdminSavePvpSeasonRequest) {
    return request<AdminPvpSeasonListResponse['seasons'][number]>(`${API_BASE}/admin/pvp/seasons`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  updatePvpSeason(seasonId: string, payload: AdminSavePvpSeasonRequest) {
    return request<AdminPvpSeasonListResponse['seasons'][number]>(`${API_BASE}/admin/pvp/seasons/${seasonId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  settlePvpSeason(seasonId: string) {
    return request<AdminSettlePvpSeasonResponse>(`${API_BASE}/admin/pvp/seasons/${seasonId}/settle`, { method: 'POST' })
  },
  setPvpProtection(playerId: string, protectionType: string, hours: number, reason = '') {
    return request<PvpStateResponse>(`${API_BASE}/admin/pvp/players/${playerId}/protection`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ protectionType, hours, reason }),
    })
  },
  forceResolvePvpMarch(marchId: string) {
    return request<PvpBattle>(`${API_BASE}/admin/pvp/marches/${marchId}/force-resolve`, { method: 'POST' })
  },
  cancelPvpMarch(marchId: string) {
    return request<PvpMarchActionResponse>(`${API_BASE}/admin/pvp/marches/${marchId}/cancel`, { method: 'POST' })
  },
  adjustResources(playerId: string, adjustments: Record<string, number>) {
    return request<{ state: GameState }>(`${API_BASE}/admin/resources/adjust`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, adjustments }),
    })
  },
  getItemsConfig() {
    return request<Record<string, ItemDefinition>>(`${API_BASE}/items/config`)
  },
  getAdminItemsConfig() {
    return request<Record<string, ItemDefinition>>(`${API_BASE}/admin/items/config`)
  },
  updateItemsConfig(config: Record<string, ItemDefinition>) {
    return request<Record<string, ItemDefinition>>(`${API_BASE}/admin/items/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  validateItemsConfig(config: Record<string, ItemDefinition>) {
    return request<{ ok: boolean; error?: string }>(`${API_BASE}/admin/items/config/validate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  getPlayerInventory(playerId: string) {
    return request<InventoryView>(`${API_BASE}/admin/items/inventory?playerId=${encodeURIComponent(playerId)}`)
  },
  getItemLedger(params: { playerId?: string; itemId?: string; refType?: string; limit?: number; offset?: number }) {
    const query = new URLSearchParams()
    if (params.playerId) query.set('playerId', params.playerId)
    if (params.itemId) query.set('itemId', params.itemId)
    if (params.refType) query.set('refType', params.refType)
    if (params.limit) query.set('limit', String(params.limit))
    if (params.offset) query.set('offset', String(params.offset))
    return request<ItemLedgerPage>(`${API_BASE}/admin/items/ledger?${query.toString()}`)
  },
  grantItem(playerId: string, itemId: string, amount: number) {
    return request<{ state: GameState }>(`${API_BASE}/admin/items/grant`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, itemId, amount }),
    })
  },
  instantCompleteRecruit(playerId: string, queueId: string) {
    return request<{ state: GameState }>(`${API_BASE}/military/recruit/instant`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, queueId }),
    })
  },
  resetGeneralStats(playerId: string) {
    return request<{ state: GameState; accountGold: number }>(`${API_BASE}/military/general/reset-stats`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId }),
    })
  },
  changeGeneral(playerId: string, generalId: string, itemId?: string) {
    return request<{ state: GameState; accountGold: number }>(`${API_BASE}/military/general/change`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, generalId, itemId }),
    })
  },
  getNpcCities(playerId: string) {
    return request<NpcState>(`${API_BASE}/map/npc-cities?playerId=${encodeURIComponent(playerId)}`)
  },
  refreshNpcCities(playerId: string) {
    return request<NpcState>(`${API_BASE}/map/npc-cities/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId }),
    })
  },
  scoutNpc(playerId: string, npcId: string) {
    return request<{ npcCity: NpcState['cities'][number]; state: GameState }>(`${API_BASE}/map/npc-cities/scout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, npcId }),
    })
  },
  attackNpc(playerId: string, npcId: string, mode: 'attack' | 'plunder', units: Record<string, number>) {
    return request<{ battleReport: GameState['recentBattleReports'][number]; state: GameState }>(
      `${API_BASE}/map/npc-cities/attack`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ playerId, npcId, mode, units }),
      },
    )
  },
  getBalance() {
    return request<BalanceConfig>(`${API_BASE}/admin/balance`)
  },
  updateBalance(balance: BalanceConfig) {
    return request<BalanceConfig>(`${API_BASE}/admin/balance`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(balance),
    })
  },
  getNpcConfig() {
    return request<NpcConfig>(`${API_BASE}/admin/npc-config`)
  },
  updateNpcConfig(config: NpcConfig) {
    return request<NpcConfig>(`${API_BASE}/admin/npc-config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  getCombatConfig() {
    return request<CombatConfig>(`${API_BASE}/admin/combat-config`)
  },
  updateCombatConfig(config: CombatConfig) {
    return request<CombatConfig>(`${API_BASE}/admin/combat-config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  getFactionsConfig() {
    return request<FactionsConfig>(`${API_BASE}/admin/factions-config`)
  },
  updateFactionsConfig(config: FactionsConfig) {
    return request<FactionsConfig>(`${API_BASE}/admin/factions-config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  getUnitsConfig() {
    return request<UnitsConfig>(`${API_BASE}/admin/units-config`)
  },
  updateUnitsConfig(faction: string, config: UnitsConfig[string]) {
    return request<UnitsConfig[string]>(`${API_BASE}/admin/units-config/${faction}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  deleteAccount(accountId: string) {
    return request<{ status: string }>(`${API_BASE}/accounts/${accountId}`, {
      method: 'DELETE',
    })
  },
  deletePlayer(playerId: string) {
    return request<{ status: string }>(`${API_BASE}/players/${playerId}`, {
      method: 'DELETE',
    })
  },
  addAccountGold(accountId: string, amount: number) {
    return request<{ gold: number }>(`${API_BASE}/admin/gold/add-account`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ accountId, amount }),
    })
  },
  addCityGold(playerId: string, amount: number) {
    return request<{ state: GameState }>(`${API_BASE}/admin/gold/add`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, amount, reason: 'GM补发' }),
    })
  },
  deductCityGold(playerId: string, amount: number) {
    return request<{ state: GameState }>(`${API_BASE}/admin/gold/deduct`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, amount, reason: 'GM扣减' }),
    })
  },
  getGoldLedger(filter: {
    accountId?: string
    playerId?: string
    currency?: 'gold' | 'cityGold'
    refType?: string
    from?: string
    to?: string
    limit?: number
  } = {}) {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(filter)) {
      if (value !== undefined && value !== '') params.set(key, String(value))
    }
    const query = params.toString()
    return request<{ entries: GoldLedgerEntry[] }>(`${API_BASE}/admin/gold/ledger${query ? `?${query}` : ''}`)
  },
  grantBuff(playerId: string, key: string, value: number, mode: string, hours: number, note: string) {
    return request<{ state: GameState }>(`${API_BASE}/admin/buff/grant`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, key, value, mode, hours, note }),
    })
  },
  revokeBuff(playerId: string, buffId: string) {
    return request<{ state: GameState }>(`${API_BASE}/admin/buff/${buffId}?playerId=${encodeURIComponent(playerId)}`, {
      method: 'DELETE',
    })
  },
  getMiniGameRecords(playerId: string, limit = 100, offset = 0, gameType = '') {
    const typeQuery = gameType ? `&gameType=${encodeURIComponent(gameType)}` : ''
    return request<{ totalRecords: number; limit: number; offset: number; hasMore: boolean; records: Array<{ id: string; playerId: string; gameType: string; resultName: string; rarity: string; rewardUnit: string; rewardAmount: number; remainingAmount: number; createdAt: string }>; rewardTotals: Record<string, number> }>(
      `${API_BASE}/admin/minigame/records?playerId=${encodeURIComponent(playerId)}&limit=${limit}&offset=${offset}${typeQuery}`,
    )
  },
  getGeneralsConfig() {
    return request<GeneralsConfig>(`${API_BASE}/admin/generals-config`)
  },
  updateGeneralsConfig(config: GeneralsConfig) {
    return request<GeneralsConfig>(`${API_BASE}/admin/generals-config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  getGeneralTraitRegistry() {
    return request<TraitRegistryResponse>(`${API_BASE}/admin/general-traits`)
  },
  getFishingConfig() {
    return request<FishingConfig>(`${API_BASE}/admin/fishing-config`)
  },
  updateFishingConfig(config: FishingConfig) {
    return request<FishingConfig>(`${API_BASE}/admin/fishing-config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  sendMail(payload: {
    playerId: string
    mailType: string
    title: string
    content: string
    attachments?: MailAttachment[]
    expiresAt?: string
  }) {
    return request<Mail>(`${API_BASE}/admin/mails/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  getPlayerMails(playerId: string, page = 1, pageSize = 10) {
    return request<MailPage>(`${API_BASE}/admin/players/${playerId}/mails?page=${page}&pageSize=${pageSize}`)
  },
  listAnnouncements(filter: { type?: string; status?: string; page?: number; pageSize?: number } = {}) {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(filter)) {
      if (value !== undefined && value !== '') params.set(key, String(value))
    }
    const query = params.toString()
    return request<AdminAnnouncementPage>(`${API_BASE}/admin/announcements${query ? `?${query}` : ''}`)
  },
  createAnnouncement(payload: SaveAnnouncementPayload) {
    return request<Announcement>(`${API_BASE}/admin/announcements`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  updateAnnouncement(announcementId: string, payload: SaveAnnouncementPayload) {
    return request<Announcement>(`${API_BASE}/admin/announcements/${announcementId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  publishAnnouncement(announcementId: string) {
    return request<Announcement>(`${API_BASE}/admin/announcements/${announcementId}/publish`, { method: 'POST' })
  },
  scheduleAnnouncement(announcementId: string, startsAt: string) {
    return request<Announcement>(`${API_BASE}/admin/announcements/${announcementId}/schedule`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ startsAt }),
    })
  },
  withdrawAnnouncement(announcementId: string) {
    return request<Announcement>(`${API_BASE}/admin/announcements/${announcementId}/withdraw`, { method: 'POST' })
  },
  archiveAnnouncement(announcementId: string) {
    return request<Announcement>(`${API_BASE}/admin/announcements/${announcementId}/archive`, { method: 'POST' })
  },
  deleteAnnouncement(announcementId: string) {
    return request<{ status: string }>(`${API_BASE}/admin/announcements/${announcementId}`, { method: 'DELETE' })
  },
}
