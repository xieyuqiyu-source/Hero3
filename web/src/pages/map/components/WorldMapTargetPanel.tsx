// 本文件实现世界地图选中城池或空地的信息面板。
import type { FC, ReactNode } from 'react'
import { LoaderCircle, Search, ShieldAlert, ShieldPlus, Swords, X } from 'lucide-react'
import type { PvpTargetSummary } from '@/types/game'
import { FACTION_COLORS, FACTION_LABELS } from '@/utils/faction'
import { buildWorldMapActionStates, buildWorldMapTargetMetrics, directionFrom, distanceFrom, estimateWorldMarchSeconds, formatDuration, worldMapRelationBadge, worldMapRelationBadgeClass, worldMapRelationLabel, worldMapStatusPillClass, type GridPosition } from '../worldMapGridLogic'

// WorldMapTargetPanel 渲染选中城池、选中空地或未选择状态。
const WorldMapTargetPanel: FC<{
  target: PvpTargetSummary | null
  emptyCell: { x: number; y: number } | null
  selfPosition: GridPosition | null
  busyTarget: string | null
  onScout: (target: PvpTargetSummary) => Promise<void>
  onReinforce: (target: PvpTargetSummary) => void
  onMarch: (target: PvpTargetSummary, mode: 'attack' | 'plunder') => void
  onClearSelection: () => void
}> = ({ target, emptyCell, selfPosition, busyTarget, onScout, onReinforce, onMarch, onClearSelection }) => {
  if (target) {
    const actions = buildWorldMapActionStates(target, busyTarget, target.playerId)
    const actionByKey = Object.fromEntries(actions.map((action) => [action.key, action]))
    const disabledReasons = actions.filter((action) => action.reason)
    const metrics = selfPosition
      ? buildWorldMapTargetMetrics(selfPosition, target.position)
      : { direction: target.direction || '未知', distance: target.distance, seconds: estimateWorldMarchSeconds(target.distance, 1) }
    return (
      <section className="relative rounded-xl border border-[var(--color-accent-border)] bg-[var(--color-surface)] px-3 py-3 pr-10 shadow-sm">
        <PanelCloseButton onClick={onClearSelection} />
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="truncate text-sm font-black text-[var(--color-text-primary)]">{target.nickname}</h3>
              <span className={`text-xs font-bold ${FACTION_COLORS[target.faction] ?? 'text-[var(--color-text-muted)]'}`}>{FACTION_LABELS[target.faction] ?? target.faction}</span>
              <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${worldMapRelationBadgeClass(target.relation)}`}>{worldMapRelationBadge(target.relation)} · {worldMapRelationLabel(target.relation)}</span>
              <span className="rounded bg-[var(--color-surface-dim)] px-1.5 py-0.5 text-[10px] font-bold text-[var(--color-text-secondary)]">Lv.{target.buildingLevel}</span>
              <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${worldMapStatusPillClass(target.status)}`}>{statusLabel(target.status)}</span>
            </div>
            <div className="mt-2 flex flex-wrap gap-1.5 text-[10px] text-[var(--color-text-muted)]">
              <InfoPill value={`坐标 (${target.position.x}, ${target.position.y})`} />
              <InfoPill value={`方位 ${metrics.direction}`} />
              <InfoPill value={`距离 ${metrics.distance} 格`} />
              <InfoPill value={`速度1 预计行军 ${formatDuration(metrics.seconds)}`} tone="green" />
            </div>
          </div>
          <div className="w-full sm:w-auto sm:min-w-72">
            <div className="grid grid-cols-4 gap-1.5">
              <PanelActionButton
                label="侦查"
                icon={<Search size={12} />}
                busy={actionByKey.scout.busy}
                disabled={actionByKey.scout.disabled}
                title={actionByKey.scout.reason || '侦查玩家城池'}
                onClick={() => void onScout(target)}
                tone="blue"
              />
              <PanelActionButton
                label="攻击"
                icon={<Swords size={12} />}
                busy={actionByKey.attack.busy}
                disabled={actionByKey.attack.disabled}
                title={actionByKey.attack.reason || '攻击玩家城池'}
                onClick={() => onMarch(target, 'attack')}
                tone="red"
              />
              <PanelActionButton
                label="掠夺"
                icon={<ShieldAlert size={12} />}
                busy={actionByKey.plunder.busy}
                disabled={actionByKey.plunder.disabled}
                title={actionByKey.plunder.reason || '掠夺玩家城池'}
                onClick={() => onMarch(target, 'plunder')}
                tone="amber"
              />
              <PanelActionButton
                label="增援"
                icon={<ShieldPlus size={12} />}
                busy={actionByKey.reinforce.busy}
                disabled={actionByKey.reinforce.disabled}
                title={actionByKey.reinforce.reason || '向玩家城池派出增援'}
                onClick={() => onReinforce(target)}
                tone="green"
              />
            </div>
            {disabledReasons.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1 text-[10px] font-semibold text-amber-600">
                {disabledReasons.map((action) => (
                  <span key={`${action.key}:${action.reason}`} className="rounded bg-amber-500/10 px-1.5 py-0.5">
                    {action.label}：{action.reason}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
      </section>
    )
  }
  if (emptyCell) {
    const direction = selfPosition ? directionFrom(selfPosition, emptyCell) : '未知'
    const distance = selfPosition ? distanceFrom(selfPosition, emptyCell) : 0
    const estimatedSeconds = estimateWorldMarchSeconds(distance, 1)
    return (
      <section className="relative rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-3 pr-10 text-xs text-[var(--color-text-secondary)]">
        <PanelCloseButton onClick={onClearSelection} />
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-black text-[var(--color-text-primary)]">空地</span>
          <InfoPill value={`坐标 (${emptyCell.x}, ${emptyCell.y})`} />
          <InfoPill value={`方位 ${direction}`} />
          <InfoPill value={`距离 ${distance} 格`} />
          <InfoPill value={`速度1 预计行军 ${formatDuration(estimatedSeconds)}`} tone="green" />
        </div>
        <p className="mt-2 text-[var(--color-text-muted)]">第一版暂无操作</p>
      </section>
    )
  }
  return (
    <section className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-8 text-center text-xs text-[var(--color-text-muted)]">
      点选地图上的城池或草地格
    </section>
  )
}

// InfoPill 展示目标面板中的短信息。
const InfoPill: FC<{ value: string; tone?: 'green' }> = ({ value, tone }) => (
  <span className={`rounded px-1.5 py-0.5 font-semibold ${tone === 'green' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)]'}`}>
    {value}
  </span>
)

// PanelCloseButton 清除当前地图选中目标或空地。
const PanelCloseButton: FC<{ onClick: () => void }> = ({ onClick }) => (
  <button
    type="button"
    onClick={onClick}
    title="取消选择"
    className="absolute right-2 top-2 inline-flex h-7 w-7 items-center justify-center rounded-lg text-[var(--color-text-muted)] hover:bg-[var(--color-surface-dim)] hover:text-[var(--color-accent)]"
  >
    <X size={13} />
  </button>
)

const ACTION_TONE_CLASS: Record<string, string> = {
  blue: 'bg-blue-500/10 text-blue-600 hover:bg-blue-500/20',
  green: 'bg-emerald-500/10 text-emerald-600 hover:bg-emerald-500/20',
  red: 'bg-red-500/10 text-red-600 hover:bg-red-500/20',
  amber: 'bg-amber-500/10 text-amber-600 hover:bg-amber-500/20',
}

// PanelActionButton 渲染选中城池面板中的地图操作按钮。
const PanelActionButton: FC<{ label: string; icon: ReactNode; busy: boolean; disabled: boolean; title: string; onClick: () => void; tone: string }> = ({ label, icon, busy, disabled, title, onClick, tone }) => (
  <button
    type="button"
    disabled={disabled}
    title={title}
    onClick={onClick}
    className={`inline-flex h-9 items-center justify-center gap-1 rounded-lg px-2 text-xs font-bold transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-45 ${ACTION_TONE_CLASS[tone] ?? ACTION_TONE_CLASS.blue}`}
  >
    {busy ? <LoaderCircle size={12} className="animate-spin" /> : icon}
    <span>{label}</span>
  </button>
)

// statusLabel 返回地图目标状态的中文短文本。
function statusLabel(status?: string) {
  if (status === 'self') return '自己'
  if (status === 'protected') return '保护中'
  if (status === 'truce') return '免战中'
  if (status === 'attackable') return '可攻击'
  if (status === 'unavailable') return '不可操作'
  return '正常'
}

export default WorldMapTargetPanel
