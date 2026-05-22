import { useEffect, useMemo, useState } from 'react'
import { adminApi } from '@/api'
import type { AccountSummary, GameState, HealthState } from '@/types'

export function useAdminDashboard() {
  const [health, setHealth] = useState<HealthState | null>(null)
  const [gameState, setGameState] = useState<GameState | null>(null)
  const [accounts, setAccounts] = useState<AccountSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true

    async function load() {
      setLoading(true)
      setError(null)

      try {
        const [nextHealth, nextGameState, accountResult] = await Promise.all([
          adminApi.getHealth(),
          adminApi.getGameState(),
          adminApi.getAccounts(),
        ])
        if (!active) return
        setHealth(nextHealth)
        setGameState(nextGameState)
        setAccounts(accountResult.accounts)
      } catch (loadError) {
        if (!active) return
        setError(loadError instanceof Error ? loadError.message : '后台状态加载失败')
      } finally {
        if (active) {
          setLoading(false)
        }
      }
    }

    void load()

    return () => {
      active = false
    }
  }, [])

  const dashboardStats = useMemo(() => {
    const resources = gameState?.resources
    const totalArmy = gameState?.army.reduce((sum, unit) => sum + unit.amount, 0) ?? 0
    const totalPlayers = accounts.reduce((sum, account) => sum + account.players.length, 0)

    return [
      { label: '后端状态', value: health?.status === 'ok' ? '在线' : '离线', hint: health?.version ?? '等待连接' },
      { label: '注册账号', value: accounts.length.toLocaleString(), hint: `${totalPlayers} 个存档` },
      { label: '总兵力', value: totalArmy.toLocaleString(), hint: `${gameState?.army.length ?? 0} 个兵种` },
      { label: '资源容量', value: resources?.capacity.toLocaleString() ?? '--', hint: '当前仓库上限' },
    ]
  }, [accounts, gameState, health])

  return {
    accounts,
    dashboardStats,
    error,
    gameState,
    health,
    loading,
  }
}
