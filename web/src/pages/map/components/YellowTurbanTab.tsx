/* 本文件实现据点页中的黄巾起义状态面板。 */
import { useEffect, useMemo, useState, type FC } from 'react'
import { RefreshCw, Shield, Timer, Tent } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import type { YellowTurbanStatusResponse } from '@/types/game'
import { toast } from '@/components/ui/toastStore'

const YellowTurbanTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const loadMilitaryView = useGameStore((s) => s.loadMilitaryView)
  const navigate = useNavigate()
  const [status, setStatus] = useState<YellowTurbanStatusResponse | null>(null)
  const [loading, setLoading] = useState(false)

  const pressurePct = useMemo(() => Math.round((status?.foodPressure.pressure ?? 0) * 100), [status])

  const loadStatus = async () => {
    if (!activePlayerId) return
    setLoading(true)
    try {
      const next = await gameApi.getYellowTurbanStatus(activePlayerId)
      setStatus(next)
      await loadMilitaryView(activePlayerId)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '黄巾状态读取失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadStatus()
  }, [activePlayerId])

  if (!activePlayerId) {
    return <div className="py-16 text-center text-sm text-[var(--color-text-muted)]">请选择存档</div>
  }

  const food = status?.foodPressure

  return (
    <div className="space-y-4">
      <div className="grid gap-3 lg:grid-cols-[1.2fr_0.8fr]">
        <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <Tent size={18} className="text-[var(--color-accent)]" />
              <h2 className="text-base font-semibold text-[var(--color-text-primary)]">黄巾起义据点</h2>
            </div>
            <button
              type="button"
              onClick={loadStatus}
              disabled={loading}
              className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-xs font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] disabled:cursor-not-allowed disabled:opacity-60"
            >
              <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
              刷新
            </button>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            <Metric label="当前口粮" value={`${formatCompactNumber(food?.currentFood)} / ${formatCompactNumber(food?.foodCapacity)}`} danger={food?.overCapacity} />
            <Metric label="压力值" value={`${pressurePct}%`} danger={food?.overCapacity} />
            <Metric label="压力等级" value={food?.riskLevelName ?? '安全'} danger={food?.overCapacity} />
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => navigate('/city', { state: { tab: 'military', focusBuildingType: 'thousand_tent_camp' } })}
              className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]"
            >
              <Shield size={15} />
              升级千帐营
            </button>
          </div>
        </section>

        <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
          <div className="mb-3 flex items-center gap-2">
            <Timer size={17} className="text-[var(--color-accent)]" />
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">检测节奏</h3>
          </div>
          <div className="space-y-2 text-sm text-[var(--color-text-secondary)]">
            <div className="flex justify-between gap-3">
              <span>检测间隔</span>
              <span className="font-semibold text-[var(--color-text-primary)]">{status?.checkIntervalMinutes ?? 30} 分钟</span>
            </div>
            <div className="flex justify-between gap-3">
              <span>下次检测</span>
              <span className="font-semibold text-[var(--color-text-primary)]">{formatTime(status?.nextCheckAt)}</span>
            </div>
            <div className="flex justify-between gap-3">
              <span>千帐营等级</span>
              <span className="font-semibold text-[var(--color-text-primary)]">Lv.{food?.thousandTentLevel ?? 1}</span>
            </div>
          </div>
        </section>
      </div>

    </div>
  )
}

const Metric: FC<{ label: string; value: string; danger?: boolean }> = ({ label, value, danger }) => (
  <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
    <div className="text-xs text-[var(--color-text-muted)]">{label}</div>
    <div className={`mt-1 text-sm font-bold ${danger ? 'text-red-600' : 'text-[var(--color-text-primary)]'}`}>{value}</div>
  </div>
)

const formatCompactNumber = (value?: number) => {
  const amount = value ?? 0
  if (amount >= 100000000) return `${Number((amount / 100000000).toFixed(1))}亿`
  if (amount >= 10000) return `${Number((amount / 10000).toFixed(1))}万`
  return amount.toLocaleString()
}

const formatTime = (value?: string) => {
  if (!value) return '--'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--'
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

export default YellowTurbanTab
