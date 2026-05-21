import { useEffect, useMemo, useState } from 'react'
import { adminApi } from './api'
import type { GameState, HealthState } from './types'

export function useAdminDashboard() {
  const [health, setHealth] = useState<HealthState | null>(null)
  const [gameState, setGameState] = useState<GameState | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true

    async function load() {
      setLoading(true)
      setError(null)

      try {
        const [nextHealth, nextGameState] = await Promise.all([
          adminApi.getHealth(),
          adminApi.getGameState(),
        ])
        if (!active) return
        setHealth(nextHealth)
        setGameState(nextGameState)
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

    return [
      { label: '后端状态', value: health?.status === 'ok' ? '在线' : '离线', hint: health?.version ?? '等待连接' },
      { label: '测试玩家', value: gameState?.player.nickname ?? '--', hint: gameState?.player.faction ?? '未同步' },
      { label: '总兵力', value: totalArmy.toLocaleString(), hint: `${gameState?.army.length ?? 0} 个兵种` },
      { label: '资源容量', value: resources?.capacity.toLocaleString() ?? '--', hint: '当前仓库上限' },
    ]
  }, [gameState, health])

  return {
    dashboardStats,
    error,
    gameState,
    health,
    loading,
  }
}
