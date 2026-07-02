// 本文件实现 GM 后台玩家世界坐标查询和调整面板。
import { useCallback, useEffect, useState } from 'react'
import { LocateFixed, Search, Save } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { WorldMapCoordinateCheck, WorldMapOccupancyStats, WorldPosition } from '@/types'

interface WorldPositionPanelProps {
  playerId: string
}

// WorldPositionPanel 展示并保存玩家在世界地图里的权威坐标。
export default function WorldPositionPanel({ playerId }: WorldPositionPanelProps) {
  const [position, setPosition] = useState<WorldPosition | null>(null)
  const [occupancy, setOccupancy] = useState<WorldMapOccupancyStats | null>(null)
  const [coordinateCheck, setCoordinateCheck] = useState<WorldMapCoordinateCheck | null>(null)
  const [form, setForm] = useState({ x: '', y: '' })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [checking, setChecking] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  // loadPosition 读取当前玩家的权威世界坐标。
  const loadPosition = useCallback(async () => {
    setLoading(true)
    setError('')
    setMessage('')
    try {
      const [next, stats] = await Promise.all([
        adminApi.getWorldPosition(playerId),
        adminApi.getWorldMapOccupancy(),
      ])
      setPosition(next)
      setOccupancy(stats)
      setForm({ x: String(next.x), y: String(next.y) })
      setCoordinateCheck({ worldId: next.worldId, x: next.x, y: next.y, occupied: true, playerId: next.playerId })
    } catch (err) {
      setError(err instanceof Error ? err.message : '坐标读取失败')
    } finally {
      setLoading(false)
    }
  }, [playerId])

  useEffect(() => {
    queueMicrotask(() => {
      void loadPosition()
    })
  }, [loadPosition])

  // parseFormCoordinate 解析并校验 GM 输入的世界坐标。
  const parseFormCoordinate = () => {
    const x = Number.parseInt(form.x, 10)
    const y = Number.parseInt(form.y, 10)
    if (!Number.isInteger(x) || !Number.isInteger(y) || x < 0 || x > 99 || y < 0 || y > 99) {
      setError('坐标必须在 0-99 范围内')
      return null
    }
    return { x, y }
  }

  // handleCheckCoordinate 检查 GM 输入坐标是否已经被其它玩家占用。
  const handleCheckCoordinate = async () => {
    const coordinate = parseFormCoordinate()
    if (!coordinate) return null
    setChecking(true)
    setError('')
    setMessage('')
    try {
      const result = await adminApi.checkWorldCoordinate(coordinate.x, coordinate.y)
      setCoordinateCheck(result)
      if (result.occupied && result.playerId !== playerId) {
        setError(`坐标已被玩家 ${result.playerId} 占用`)
      } else if (result.occupied) {
        setMessage('当前玩家已占用该坐标')
      } else {
        setMessage('该坐标为空，可以调整')
      }
      return result
    } catch (err) {
      setError(err instanceof Error ? err.message : '坐标检查失败')
      return null
    } finally {
      setChecking(false)
    }
  }

  // handleSave 保存 GM 输入的世界坐标。
  const handleSave = async () => {
    const coordinate = parseFormCoordinate()
    if (!coordinate) {
      return
    }
    const check = await handleCheckCoordinate()
    if (check?.occupied && check.playerId !== playerId) {
      return
    }
    setSaving(true)
    setError('')
    setMessage('')
    try {
      const next = await adminApi.updateWorldPosition(playerId, coordinate.x, coordinate.y)
      const stats = await adminApi.getWorldMapOccupancy()
      setPosition(next)
      setOccupancy(stats)
      setForm({ x: String(next.x), y: String(next.y) })
      setCoordinateCheck({ worldId: next.worldId, x: next.x, y: next.y, occupied: true, playerId: next.playerId })
      setMessage('世界坐标已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '坐标保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
      <div className="mb-2.5 flex items-center justify-between gap-2">
        <h4 className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)]">
          <LocateFixed size={13} />
          世界坐标
        </h4>
        {position && <span className="text-[10px] text-[var(--color-text-muted)]">{position.worldId} · {position.assignedBy || 'system'}</span>}
      </div>
      {loading ? (
        <div className="py-3 text-xs text-[var(--color-text-muted)]">读取坐标中...</div>
      ) : (
        <div className="grid gap-2 sm:grid-cols-[1fr_1fr_auto_auto]">
          <label className="grid gap-1">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)]">X</span>
            <input
              type="number"
              min={0}
              max={99}
              value={form.x}
              onChange={(event) => {
                setForm((prev) => ({ ...prev, x: event.target.value }))
                setCoordinateCheck(null)
              }}
              className="h-9 rounded-xl border border-[var(--color-border)] bg-white px-3 text-sm font-bold text-[var(--color-text-primary)] outline-none dark:bg-slate-900"
            />
          </label>
          <label className="grid gap-1">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)]">Y</span>
            <input
              type="number"
              min={0}
              max={99}
              value={form.y}
              onChange={(event) => {
                setForm((prev) => ({ ...prev, y: event.target.value }))
                setCoordinateCheck(null)
              }}
              className="h-9 rounded-xl border border-[var(--color-border)] bg-white px-3 text-sm font-bold text-[var(--color-text-primary)] outline-none dark:bg-slate-900"
            />
          </label>
          <button
            type="button"
            onClick={() => void handleCheckCoordinate()}
            disabled={checking || saving}
            className="mt-4 inline-flex h-9 items-center justify-center gap-1.5 rounded-xl border border-[var(--color-border)] px-4 text-xs font-bold text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] disabled:opacity-60 sm:mt-auto"
          >
            <Search size={13} />
            {checking ? '检查中' : '检查'}
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving || checking || (coordinateCheck?.occupied === true && coordinateCheck.playerId !== playerId)}
            className="mt-4 inline-flex h-9 items-center justify-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-4 text-xs font-bold text-white disabled:opacity-60 sm:mt-auto"
          >
            <Save size={13} />
            {saving ? '保存中' : '保存'}
          </button>
        </div>
      )}
      {message && <p className="mt-2 text-xs font-semibold text-emerald-600">{message}</p>}
      {error && <p className="mt-2 text-xs font-semibold text-red-600">{error}</p>}
      {position && (
        <p className="mt-2 text-[10px] text-[var(--color-text-muted)]">
          当前格 ({position.x}, {position.y})，一格等于距离 1，速度 1 兵种每格 5 分钟。
        </p>
      )}
      {coordinateCheck && (
        <p className="mt-2 text-[10px] font-semibold text-[var(--color-text-muted)]">
          检查格 ({coordinateCheck.x}, {coordinateCheck.y})：{coordinateCheck.occupied ? `已占用 ${coordinateCheck.playerId ?? ''}` : '空闲'}
        </p>
      )}
      {occupancy && (
        <div className="mt-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-[10px] text-[var(--color-text-muted)]">
          <div className="mb-1 flex items-center justify-between gap-2">
            <span className="font-bold text-[var(--color-text-secondary)]">地图占用率</span>
            <span>{formatPercent(occupancy.occupancyRate)}</span>
          </div>
          <div className="h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
            <div className="h-full rounded-full bg-[var(--color-accent)]" style={{ width: formatPercent(occupancy.occupancyRate) }} />
          </div>
          <div className="mt-1 flex flex-wrap justify-between gap-2">
            <span>{occupancy.worldId} · {occupancy.width}x{occupancy.height}</span>
            <span>已占 {occupancy.occupiedCells.toLocaleString()} / 可用 {occupancy.availableCells.toLocaleString()}</span>
          </div>
        </div>
      )}
    </section>
  )
}

// formatPercent 把占用率格式化为百分比。
function formatPercent(value: number) {
  return `${Math.max(0, Math.min(100, value * 100)).toFixed(2)}%`
}
