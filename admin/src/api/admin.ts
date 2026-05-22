import type { AccountSummary, GameState, HealthState } from '@/types'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api/v1'
const ROOT_BASE = API_BASE.replace(/\/api\/v1$/, '')

async function request<T>(url: string): Promise<T> {
  const response = await fetch(url)
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
}
