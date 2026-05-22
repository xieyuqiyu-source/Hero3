import type { AccountSummary, BalanceConfig, GameState, HealthState } from '@/types'

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
}
