import { useCallback, useEffect, useMemo, useState } from 'react'
import { adminApi } from '@/api'
import type { AccountSummary, GameState, HealthState } from '@/types'

export function useAdminDashboard() {
  const [health, setHealth] = useState<HealthState | null>(null)
  const [gameState, setGameState] = useState<GameState | null>(null)
  const [accounts, setAccounts] = useState<AccountSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionMessage, setActionMessage] = useState<string | null>(null)
  const [busyTarget, setBusyTarget] = useState<string | null>(null)

  const loadDashboard = useCallback(async (active = true) => {
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
  }, [])

  useEffect(() => {
    let active = true

    queueMicrotask(() => {
      void loadDashboard(active)
    })

    return () => {
      active = false
    }
  }, [loadDashboard])

  const deletePlayer = async (playerId: string) => {
    setBusyTarget(`player:${playerId}`)
    setActionMessage(null)
    setError(null)
    try {
      await adminApi.deletePlayer(playerId)
      await loadDashboard()
      setActionMessage('云存档已删除')
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '云存档删除失败')
    } finally {
      setBusyTarget(null)
    }
  }

  const deleteAccount = async (accountId: string) => {
    setBusyTarget(`account:${accountId}`)
    setActionMessage(null)
    setError(null)
    try {
      await adminApi.deleteAccount(accountId)
      await loadDashboard()
      setActionMessage('账号及关联云存档已删除')
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '账号删除失败')
    } finally {
      setBusyTarget(null)
    }
  }

  const dashboardStats = useMemo(() => {
    const resources = gameState?.resources
    const totalArmy = gameState?.army.reduce((sum, unit) => sum + unit.amount, 0) ?? 0
    const totalPlayers = accounts.reduce((sum, account) => sum + account.players.length, 0)

    return [
      { label: '后端状态', value: health?.status === 'ok' ? '在线' : '离线', hint: health?.version ?? '等待连接' },
      { label: '注册账号', value: accounts.length.toLocaleString(), hint: `${totalPlayers} 个存档` },
      { label: '总兵力', value: totalArmy.toLocaleString(), hint: `${gameState?.army.length ?? 0} 个兵种` },
      { label: '资源容量', value: resources?.capacity.wood.toLocaleString() ?? '--', hint: '当前仓库上限' },
    ]
  }, [accounts, gameState, health])

  return {
    actionMessage,
    accounts,
    busyTarget,
    dashboardStats,
    deleteAccount,
    deletePlayer,
    error,
    gameState,
    health,
    loading,
    reload: loadDashboard,
  }
}
