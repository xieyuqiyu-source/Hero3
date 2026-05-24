import { useState, useEffect, type FC } from 'react'
import { Eye, EyeOff, X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { BattleReport } from '@/types/game'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'

interface ScoutResultModalProps {
  report: BattleReport
  onClose: () => void
}

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }

const ScoutResultModal: FC<ScoutResultModalProps> = ({ report, onClose }) => {
  const [visible, setVisible] = useState(false)
  const navigate = useNavigate()
  const nickname = useGameStore((s) => s.state?.player.nickname ?? '我方')
  const defenderFaction = report.defenderFaction
  const defenderUnits = useConfigStore((s) => s.units)?.[defenderFaction] ?? {}

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 200)
  }

  const success = report.defenderRevealed

  const getUnitName = (unitType: string): string => {
    return defenderUnits[unitType]?.name ?? unitType
  }

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className={`absolute inset-0 bg-slate-900/50 backdrop-blur-[4px] transition-opacity duration-200 ${visible ? 'opacity-100' : 'opacity-0'}`}
        onClick={handleClose}
      />

      {/* Modal */}
      <div className={`
        relative w-full max-w-sm rounded-2xl overflow-hidden
        bg-[var(--color-surface)] border border-[var(--color-border)]
        shadow-[0_24px_60px_rgba(15,23,42,0.3)]
        transition-all duration-200
        ${visible ? 'opacity-100 scale-100 translate-y-0' : 'opacity-0 scale-95 translate-y-4'}
      `}>
        {/* Header */}
        <div className={`px-4 py-4 text-center ${success ? 'bg-blue-500/10' : 'bg-red-500/10'}`}>
          {success ? (
            <Eye size={28} className="mx-auto text-blue-500 mb-1" />
          ) : (
            <EyeOff size={28} className="mx-auto text-red-500 mb-1" />
          )}
          <h2 className={`text-lg font-bold ${success ? 'text-blue-600' : 'text-red-600'}`}>
            {success ? '侦查成功' : '侦查失败'}
          </h2>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            {nickname} → {report.targetName}
          </p>
          <button
            type="button"
            onClick={handleClose}
            className="absolute top-3 right-3 p-1 rounded-full hover:bg-white/20 cursor-pointer"
          >
            <X size={16} className="text-[var(--color-text-muted)]" />
          </button>
        </div>

        {/* Body */}
        <div className="px-4 py-3 space-y-3">
          {success ? (
            <>
              {/* 守军 */}
              {Object.keys(report.defenderUnits).length > 0 && (
                <div>
                  <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">守军</h3>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.entries(report.defenderUnits).filter(([, v]) => v > 0).map(([unitType, count]) => (
                      <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-blue-500/10 text-blue-600 font-medium">
                        {getUnitName(unitType)} ×{count.toLocaleString()}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* 资源 */}
              {Object.keys(report.defenderResources).length > 0 && (
                <div>
                  <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">资源</h3>
                  <div className="grid grid-cols-2 gap-1.5">
                    {Object.entries(report.defenderResources).filter(([, v]) => v > 0).map(([res, val]) => (
                      <div key={res} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                        <span className="text-[10px] text-[var(--color-text-secondary)]">{RESOURCE_LABELS[res] ?? res}</span>
                        <span className="text-xs font-bold text-[var(--color-text-primary)]">{val.toLocaleString()}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          ) : (
            <p className="text-xs text-[var(--color-text-muted)] text-center py-2">
              侦察兵全军覆没，无法获取对方情报
            </p>
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-[var(--color-border)] space-y-2">
          <p className="text-[10px] text-[var(--color-text-muted)] text-center">
            详细情报请前往
            <button
              type="button"
              onClick={() => { handleClose(); navigate('/news') }}
              className="text-[var(--color-accent)] font-medium hover:underline cursor-pointer mx-0.5"
            >
              军情
            </button>
            查看
          </p>
          <button
            type="button"
            onClick={handleClose}
            className="w-full px-4 py-2.5 rounded-xl text-sm font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity"
          >
            确定
          </button>
        </div>
      </div>
    </div>
  )
}

export default ScoutResultModal
