/* 本文件渲染协防战报专用上下文。 */
import type { FC } from 'react'
import type { BattleReportReinforcementExtra } from '@/types/game'

interface ReportReinforcementContextProps {
  reinforcement?: BattleReportReinforcementExtra
}

// ReportReinforcementContext 只说明协防关系，出动和阵亡由参战方区块统一展示。
const ReportReinforcementContext: FC<ReportReinforcementContextProps> = ({ reinforcement }) => {
  if (!reinforcement) return null
  return (
    <section className="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-3">
      <h3 className="text-xs font-bold text-emerald-500">协防关系</h3>
      <div className="mt-2 grid gap-2 text-[11px] text-[var(--color-text-secondary)] sm:grid-cols-2">
        <div><span className="font-bold text-[var(--color-text-primary)]">被协防</span><br />{reinforcement.hostPlayerName || reinforcement.host?.playerName || '未知'} · {reinforcement.hostCityName || reinforcement.host?.cityName || '未知城池'}</div>
        <div><span className="font-bold text-[var(--color-text-primary)]">对抗方</span><br />{reinforcement.attackerName || reinforcement.attacker?.playerName || '黄巾/未知'} · {reinforcement.attackerCityName || reinforcement.attacker?.cityName || '未知来源'}</div>
      </div>
    </section>
  )
}

export default ReportReinforcementContext
