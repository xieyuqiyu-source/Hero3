import type { GameState, ModifierBreakdownItem } from '@/types/game'

type BreakdownItem = {
  source: string
  value: number
  mode: string
}

export type EffectiveUnitStat = {
  base: number
  final: number
  breakdown: BreakdownItem[]
}

const STAT_MODIFIER_KEYS: Record<string, string[]> = {
  attack: ['attackBonus'],
  infantryDefense: ['defenseBonus', 'infantryDefenseBonus'],
  cavalryDefense: ['defenseBonus', 'cavalryDefenseBonus'],
}

export function getEffectiveUnitStat(state: GameState | null | undefined, statKey: string, baseValue: number): EffectiveUnitStat {
  const modifierKeys = STAT_MODIFIER_KEYS[statKey]
  if (!state || !modifierKeys) {
    return { base: baseValue, final: baseValue, breakdown: [] }
  }

  let value = baseValue
  const breakdown: BreakdownItem[] = []

  for (const modifierKey of modifierKeys) {
    const modifiers = collectStatModifiers(state, modifierKey)
    breakdown.push(...modifiers)
    value = applyModifiers(value, modifiers)
  }

  return {
    base: baseValue,
    final: Math.floor(value),
    breakdown,
  }
}

export function getEffectiveRecruitSeconds(state: GameState | null | undefined, category: string, baseSeconds: number): EffectiveUnitStat {
  if (!state) {
    return { base: baseSeconds, final: baseSeconds, breakdown: [] }
  }

  let seconds = baseSeconds
  const breakdown: BreakdownItem[] = []

  const globalModifiers = collectStatModifiers(state, 'recruitSpeedBonus')
  breakdown.push(...globalModifiers)
  seconds = applySpeedModifiers(seconds, globalModifiers)

  const categoryKey = category === 'infantry'
    ? 'infantryRecruitSpeedBonus'
    : category === 'cavalry'
      ? 'cavalryRecruitSpeedBonus'
      : ''
  if (categoryKey) {
    const categoryModifiers = collectStatModifiers(state, categoryKey)
    breakdown.push(...categoryModifiers)
    seconds = applySpeedModifiers(seconds, categoryModifiers)
  }

  return {
    base: baseSeconds,
    final: Math.max(1, Math.floor(seconds)),
    breakdown,
  }
}

export function formatUnitStatTitle(label: string, stat: EffectiveUnitStat): string {
  const lines = [`${label}: ${formatBaseFinal(stat)}`]
  if (stat.breakdown.length > 0) {
    lines.push('来源:')
    for (const item of stat.breakdown) {
      lines.push(`${item.source} ${formatModifierValue(item)}`)
    }
  }
  return lines.join('\n')
}

export function formatBaseFinal(stat: EffectiveUnitStat): string {
  return stat.final === stat.base ? `${stat.base}` : `${stat.base} → ${stat.final}`
}

export function formatSecondsBaseFinal(stat: EffectiveUnitStat): string {
  return stat.final === stat.base ? `${stat.base}s` : `${stat.base}s → ${stat.final}s`
}

export function formatModifierValue(item: BreakdownItem): string {
  if (item.mode === 'flat') {
    return `${item.value >= 0 ? '+' : ''}${item.value}`
  }
  if (item.mode === 'percentMultiply') {
    return `×${(1 + item.value).toFixed(2)}`
  }
  return `${item.value >= 0 ? '+' : ''}${Math.round(item.value * 100)}%`
}

function collectStatModifiers(state: GameState, key: string): BreakdownItem[] {
  const result: BreakdownItem[] = []
  const generalBreakdown = state.general?.attributeBreakdown?.[key]

  if (generalBreakdown && generalBreakdown.length > 0) {
    for (const item of generalBreakdown) {
      result.push({ source: item.source, value: item.value, mode: 'percentAdd' })
    }
  }

  for (const item of state.activeModifiers ?? []) {
    if (item.key !== key) continue
    if (item.source === '将领' && generalBreakdown && generalBreakdown.length > 0) continue
    result.push(toBreakdownItem(item))
  }

  return result.filter((item) => item.value !== 0)
}

function toBreakdownItem(item: ModifierBreakdownItem): BreakdownItem {
  return {
    source: item.source,
    value: item.value,
    mode: item.mode,
  }
}

function applyModifiers(base: number, modifiers: BreakdownItem[]): number {
  let flatSum = 0
  let percentAddSum = 0
  let multiplier = 1

  for (const item of modifiers) {
    if (item.mode === 'flat') {
      flatSum += item.value
    } else if (item.mode === 'percentMultiply') {
      multiplier *= 1 + item.value
    } else {
      percentAddSum += item.value
    }
  }

  return (base + flatSum) * (1 + percentAddSum) * multiplier
}

function applySpeedModifiers(baseSeconds: number, modifiers: BreakdownItem[]): number {
  let additive = 0
  let multiplier = 1

  for (const item of modifiers) {
    if (item.mode === 'percentMultiply') {
      multiplier *= 1 + item.value
    } else {
      additive += item.value
    }
  }

  const speedFactor = (1 + additive) * multiplier
  if (speedFactor <= 0) return baseSeconds
  return baseSeconds / speedFactor
}
