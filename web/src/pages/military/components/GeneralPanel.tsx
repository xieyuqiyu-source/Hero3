import { type FC } from 'react'
import { useGameStore } from '@/store/gameStore'

const INVENTORY_SLOTS = 20

const GeneralPanel: FC = () => {
  const general = useGameStore((s) => s.state?.general)

  if (!general) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="text-sm text-[var(--color-text-muted)]">暂无将领，请重新创建存档选择将领</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col lg:flex-row gap-4 h-[calc(100vh-220px)] min-h-[400px]">
      {/* Left: General Info */}
      <div className="flex-1 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 flex flex-col">
        {/* Header */}
        <div className="flex items-center gap-3 mb-4 pb-3 border-b border-[var(--color-border)]">
          <div className="w-14 h-14 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] flex items-center justify-center">
            <span className="text-2xl">⚔️</span>
          </div>
          <div>
            <h2 className="text-lg font-bold text-[var(--color-text-primary)]">{general.name}</h2>
            <div className="flex items-center gap-2 mt-0.5">
              <span className="text-xs px-2 py-0.5 rounded-md bg-amber-500/15 text-amber-600 font-bold">Lv.{general.level}</span>
              <span className="text-[10px] text-[var(--color-text-muted)]">EXP {general.exp}</span>
            </div>
          </div>
        </div>

        {/* Attributes */}
        <div className="mb-4">
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">属性</h3>
          <div className="grid grid-cols-2 gap-2">
            {[
              ['武力', '—'],
              ['智力', '—'],
              ['政治', '—'],
              ['统率', '—'],
            ].map(([label, value]) => (
              <div key={label} className="flex items-center justify-between px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                <span className="text-[11px] text-[var(--color-text-secondary)]">{label}</span>
                <span className="text-xs font-bold text-[var(--color-text-primary)]">{value}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Buffs */}
        <div className="mb-4">
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">加成效果</h3>
          {Object.keys(general.buffs).length > 0 ? (
            <div className="space-y-1.5">
              {Object.entries(general.buffs).map(([key, val]) => (
                <div key={key} className="flex items-center justify-between px-3 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                  <span className="text-[10px] text-[var(--color-text-secondary)]">{key}</span>
                  <span className="text-[10px] font-bold text-green-500">+{Math.round((val - 1) * 100)}%</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-[11px] text-[var(--color-text-muted)]">暂无加成，升级将领解锁</p>
          )}
        </div>

        {/* Skills placeholder */}
        <div className="flex-1">
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">技能</h3>
          <p className="text-[11px] text-[var(--color-text-muted)]">将领技能系统开发中</p>
        </div>
      </div>

      {/* Right: Inventory Grid */}
      <div className="flex-1 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 flex flex-col">
        <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-3">背包</h3>
        <div className="grid grid-cols-5 gap-2 flex-1 content-start">
          {Array.from({ length: INVENTORY_SLOTS }).map((_, i) => (
            <div
              key={i}
              className="aspect-square rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)] flex items-center justify-center"
            >
              <span className="text-[10px] text-[var(--color-text-muted)]">{i + 1}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default GeneralPanel
