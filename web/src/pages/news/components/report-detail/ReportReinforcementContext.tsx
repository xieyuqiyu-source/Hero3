/* 本文件渲染协防战报专用上下文。 */
import type { FC } from 'react'
import type { BattleReportReinforcementExtra } from '@/types/game'

interface ReportReinforcementContextProps {
  reinforcement?: BattleReportReinforcementExtra
}

// sumTroops 汇总兵力 map 数量。
function sumTroops(troops?: Record<string, number>): number {
  return Object.values(troops ?? {}).reduce((sum, amount) => sum + amount, 0)
}

// ReportReinforcementContext 说明协防方帮谁、挡谁、损失多少。
const ReportReinforcementContext: FC<ReportReinforcementContextProps> = ({ reinforcement }) => {
  if (!reinforcement) return null
  const contribution = reinforcement.ownerContribution
  return (
    <section className="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-3">
      <h3 className="text-xs font-bold text-emerald-500">协防上下文</h3>
      <div className="mt-2 grid gap-2 text-[11px] text-[var(--color-text-secondary)] sm:grid-cols-3">
        <div><span className="font-bold text-[var(--color-text-primary)]">被协防</span><br />{reinforcement.hostPlayerName || reinforcement.host?.playerName || '未知'} · {reinforcement.hostCityName || reinforcement.host?.cityName || '未知城池'}</div>
        <div><span className="font-bold text-[var(--color-text-primary)]">攻击来源</span><br />{reinforcement.attackerName || reinforcement.attacker?.playerName || '黄巾/未知'} · {reinforcement.attackerCityName || reinforcement.attacker?.cityName || '未知来源'}</div>
        <div><span className="font-bold text-[var(--color-text-primary)]">我的贡献</span><br />出动 {sumTroops(contribution?.troopsBefore).toLocaleString()} · 阵亡 {sumTroops(contribution?.troopsLost).toLocaleString()} · 经验 {contribution?.generalExp ?? 0}</div>
      </div>
      {reinforcement.battleEventId && <div className="mt-2 text-[10px] text-[var(--color-text-muted)]">事件 ID：{reinforcement.battleEventId}</div>}
    </section>
  )
}

export default ReportReinforcementContext
