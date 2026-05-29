import type { AccountSummary, BalanceConfig, GameState, HealthState, NpcConfig, NpcState } from '@/types'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api/v1'
const ROOT_BASE = API_BASE.replace(/\/api\/v1$/, '')

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init)
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  return response.json() as Promise<T>
}

export const adminApi = {
  getHealth() {
    return request<HealthState>(`${ROOT_BASE}/healthz`)
  },
  getGameState() {
    return request<GameState>(`${API_BASE}/game/state`)
  },
  getAccounts() {
    return request<{ accounts: AccountSummary[] }>(`${API_BASE}/admin/accounts`)
  },
  getPlayerState(playerId: string) {
    return request<GameState>(`${API_BASE}/admin/players/${playerId}/state`)
  },
  adjustResources(playerId: string, adjustments: Record<string, number>) {
    return request<{ state: GameState }>(`${API_BASE}/admin/resources/adjust`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, adjustments }),
    })
  },
  instantCompleteRecruit(playerId: string, queueId: string) {
    return request<{ state: GameState }>(`${API_BASE}/military/recruit/instant`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, queueId }),
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
    return request<object>(`${API_BASE}/admin/combat-config`)
  },
  updateCombatConfig(config: object) {
    return request<object>(`${API_BASE}/admin/combat-config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  getFactionsConfig() {
    return request<object>(`${API_BASE}/admin/factions-config`)
  },
  updateFactionsConfig(config: object) {
    return request<object>(`${API_BASE}/admin/factions-config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
  },
  getUnitsConfig() {
    return request<Record<string, object>>(`${API_BASE}/admin/units-config`)
  },
  updateUnitsConfig(faction: string, config: object) {
    return request<object>(`${API_BASE}/admin/units-config/${faction}`, {
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
  getMiniGameRecords(playerId: string) {
    return request<{ totalRecords: number; records: Array<{ id: string; playerId: string; gameType: string; resultName: string; rarity: string; rewardUnit: string; rewardAmount: number; createdAt: string }>; rewardTotals: Record<string, number> }>(
      `${API_BASE}/admin/minigame/records?playerId=${encodeURIComponent(playerId)}`,
    )
  },
}
