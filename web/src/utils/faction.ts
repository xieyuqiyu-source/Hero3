/** 阵营相关的全局映射工具 — 所有组件统一从这里取 */

export const FACTION_LABELS: Record<string, string> = {
  wei: '魏',
  shu: '蜀',
  wu: '吴',
}

export const FACTION_COLORS: Record<string, string> = {
  wei: 'text-blue-400',
  shu: 'text-green-400',
  wu: 'text-red-400',
}

export const FACTION_BG_COLORS: Record<string, string> = {
  wei: 'bg-blue-400/10',
  shu: 'bg-green-400/10',
  wu: 'bg-red-400/10',
}

export function getFactionLabel(faction: string): string {
  return FACTION_LABELS[faction] ?? faction
}

export function getFactionColor(faction: string): string {
  return FACTION_COLORS[faction] ?? 'text-[var(--color-text-muted)]'
}
