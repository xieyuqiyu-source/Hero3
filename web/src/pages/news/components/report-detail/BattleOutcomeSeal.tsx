/* 本文件渲染战报当前玩家视角的结果印章。 */
import type { FC } from 'react'

const OUTCOME_LABELS: Record<string, string> = {
  victory: '胜利',
  defeat: '失败',
  draw: '平局',
  intel_success: '侦查成功',
  intel_failed: '侦查失败',
  notice: '通知',
}

const OUTCOME_CLASSES: Record<string, string> = {
  victory: 'border-emerald-500/60 bg-emerald-500/10 text-emerald-500',
  defeat: 'border-red-500/60 bg-red-500/10 text-red-500',
  draw: 'border-slate-400/60 bg-slate-400/10 text-slate-400',
  intel_success: 'border-emerald-500/60 bg-emerald-500/10 text-emerald-500',
  intel_failed: 'border-red-500/60 bg-red-500/10 text-red-500',
  notice: 'border-amber-500/60 bg-amber-500/10 text-amber-500',
}

interface BattleOutcomeSealProps {
  outcome?: string
}

// BattleOutcomeSeal 用紧凑印章表达当前玩家结果。
const BattleOutcomeSeal: FC<BattleOutcomeSealProps> = ({ outcome = 'notice' }) => {
  const label = OUTCOME_LABELS[outcome] ?? OUTCOME_LABELS.notice
  const className = OUTCOME_CLASSES[outcome] ?? OUTCOME_CLASSES.notice
  return (
    <span className={`inline-flex h-10 min-w-20 items-center justify-center rounded-lg border px-3 text-sm font-black tracking-normal ${className}`}>
      {label}
    </span>
  )
}

export default BattleOutcomeSeal
