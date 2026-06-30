/* 本文件提供存档延迟删除的前端展示工具。 */
import type { PlayerSummary } from '@/types/game'

export function isPlayerDeletionPending(player: PlayerSummary): boolean {
  return Boolean(player.deleteScheduledAt)
}

export function deletionRemainingMs(player: PlayerSummary, now: number): number {
  if (!player.deleteScheduledAt) return 0
  const scheduledAt = new Date(player.deleteScheduledAt).getTime()
  if (!Number.isFinite(scheduledAt)) return 0
  return Math.max(0, scheduledAt - now)
}

export function formatDeletionCountdown(ms: number): string {
  if (ms <= 0) return '即将删除'
  const totalSeconds = Math.ceil(ms / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}
