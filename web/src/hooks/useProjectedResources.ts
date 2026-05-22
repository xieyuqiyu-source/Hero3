import { useEffect, useRef, useState } from 'react'
import { useGameStore } from '@/store/gameStore'
import type { ResourceState } from '@/types/game'

const TICK_MS = 1000

function growResource(current: number, perHour: number, elapsedSeconds: number, capacity: number) {
  if (current >= capacity || perHour <= 0 || elapsedSeconds <= 0) {
    return Math.min(current, capacity)
  }

  return Math.min(current + Math.floor((perHour * elapsedSeconds) / 3600), capacity)
}

function projectResources(state: ReturnType<typeof useGameStore.getState>['state'], stateReceivedAt: number | null, now: number): ResourceState | null {
  if (!state || !stateReceivedAt) return state?.resources ?? null

  const elapsedSeconds = Math.max(0, (now - stateReceivedAt) / 1000)
  const { resources, resourceProduction } = state

  if (!resourceProduction || Object.keys(resourceProduction).length === 0) {
    return resources
  }

  const items = { ...resources.items }
  for (const [resourceType, perHour] of Object.entries(resourceProduction)) {
    items[resourceType] = growResource(
      resources.items[resourceType] ?? 0,
      perHour,
      elapsedSeconds,
      resources.capacity[resourceType] ?? 0,
    )
  }

  return { items, capacity: resources.capacity }
}

export function useProjectedResources(): ResourceState | null {
  const state = useGameStore((store) => store.state)
  const stateReceivedAt = useGameStore((store) => store.stateReceivedAt)
  const [projected, setProjected] = useState<ResourceState | null>(() =>
    projectResources(state, stateReceivedAt, Date.now())
  )
  const stateRef = useRef(state)
  const receivedAtRef = useRef(stateReceivedAt)

  stateRef.current = state
  receivedAtRef.current = stateReceivedAt

  // 当 state 变化时立即更新
  useEffect(() => {
    setProjected(projectResources(state, stateReceivedAt, Date.now()))
  }, [state, stateReceivedAt])

  // 定时 tick 更新预测值
  useEffect(() => {
    if (!state || !stateReceivedAt) return

    const timer = window.setInterval(() => {
      setProjected(projectResources(stateRef.current, receivedAtRef.current, Date.now()))
    }, TICK_MS)

    return () => window.clearInterval(timer)
  }, [state, stateReceivedAt])

  return projected
}
