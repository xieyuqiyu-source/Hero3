import { useEffect, useMemo, useState } from 'react'
import { useGameStore } from '@/store/gameStore'
import type { ResourceState } from '@/types/game'

const TICK_MS = 1000

function growResource(current: number, perHour: number, elapsedSeconds: number, capacity: number) {
  if (current >= capacity || perHour <= 0 || elapsedSeconds <= 0) {
    return Math.min(current, capacity)
  }

  return Math.min(current + Math.floor((perHour * elapsedSeconds) / 3600), capacity)
}

export function useProjectedResources(): ResourceState | null {
  const state = useGameStore((store) => store.state)
  const stateReceivedAt = useGameStore((store) => store.stateReceivedAt)
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), TICK_MS)
    return () => window.clearInterval(timer)
  }, [])

  return useMemo(() => {
    if (!state || !stateReceivedAt) return state?.resources ?? null

    const elapsedSeconds = Math.max(0, (now - stateReceivedAt) / 1000)
    const { resources, resourceProduction } = state
    const items = { ...resources.items }

    for (const [resourceType, perHour] of Object.entries(resourceProduction)) {
      items[resourceType] = growResource(
        resources.items[resourceType] ?? 0,
        perHour,
        elapsedSeconds,
        resources.capacity[resourceType] ?? 0,
      )
    }
    return {
      items,
      capacity: resources.capacity,
    }
  }, [now, state, stateReceivedAt])
}
