// 本 Hook 每秒刷新曹操魏武号令的前端兵力投影，后端状态始终保持权威。
import { useEffect, useState } from 'react'
import { useGameStore } from '@/store/gameStore'
import { projectGuardArmy } from '@/utils/guardProjection'

/** 返回当前后端兵力叠加魏武号令即时显示增量后的只读列表。 */
export function useProjectedArmy() {
  const state = useGameStore((store) => store.state)
  const stateReceivedAt = useGameStore((store) => store.stateReceivedAt)
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (!state || !stateReceivedAt) return
    setNow(Date.now())
    const timer = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [state, stateReceivedAt])

  return projectGuardArmy(state, stateReceivedAt, now)
}
