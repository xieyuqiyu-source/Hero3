import { useState, useEffect, type FC } from 'react'
import { Trophy, Skull, X, Share2, Check } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { BattleReport } from '@/types/game'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import { getTraitMeta } from '@/utils/traits'

interface BattleResultModalProps {
  report: BattleReport
  onClose: () => void
}

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }

const BattleResultModal: FC<BattleResultModalProps> = ({ report, onClose }) => {
  const [visible, setVisible] = useState(false)
  const [copied, setCopied] = useState(false)
  const navigate = useNavigate()
  const nickname = useGameStore((s) => s.state?.player.nickname ?? '我方')
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const units = useConfigStore((s) => s.units)
  const factionUnits = units?.[faction] ?? {}

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 200)
  }

  const isVictory = report.result === 'attacker_victory'
  const isDraw = report.result === 'draw'

  const getUnitName = (unitType: string): string => {
    return factionUnits[unitType]?.name ?? unitType
  }

  const formatOutcomeDetail = (key: string, value: number | string): string => {
    const labels: Record<string, string> = {
      totalCaptured: '俘虏',
      totalRevived: '复活',
      extraDamage: '额外伤害',
      damagePercent: '伤害比例',
    }
    const label = labels[key] ?? key
    if (typeof value === 'number') {
      if (key.endsWith('Percent') || key.endsWith('Rate')) {
        return `${label}: ${Math.round(value * 100)}%`
      }
      return `${label}: ${value.toLocaleString()}`
    }
    return `${label}: ${value}`
  }

  const hasRewards = Object.values(report.rewards).some(v => v > 0)
  const hasLosses = Object.values(report.lostUnits).some(v => v > 0)

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
        <div className={`px-4 py-4 text-center ${isVictory ? 'bg-green-500/10' : isDraw ? 'bg-slate-500/10' : 'bg-red-500/10'}`}>
          {isVictory ? (
            <Trophy size={28} className="mx-auto text-green-500 mb-1" />
          ) : (
            <Skull size={28} className={`mx-auto ${isDraw ? 'text-slate-400' : 'text-red-500'} mb-1`} />
          )}
          <h2 className={`text-lg font-bold ${isVictory ? 'text-green-600' : isDraw ? 'text-slate-500' : 'text-red-600'}`}>
            {isVictory ? '战斗胜利！' : isDraw ? '两败俱伤' : '战斗失败'}
          </h2>
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
          {/* Player VS NPC */}
          <div className="flex items-center justify-center gap-3 px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
            <span className="text-sm font-bold text-[var(--color-text-primary)]">{nickname}</span>
            <span className="text-xs font-bold text-[var(--color-text-muted)]">VS</span>
            <span className="text-sm font-bold text-[var(--color-text-primary)]">{report.targetName}</span>
          </div>

          {/* Losses */}
          {hasLosses && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">我方损失</h3>
              <div className="flex flex-wrap gap-1.5">
                {Object.entries(report.lostUnits).filter(([, v]) => v > 0).map(([unitType, count]) => (
                  <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-red-500/10 text-red-600 font-medium">
                    {getUnitName(unitType)} ×{count}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Rewards */}
          {hasRewards && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">掠夺资源</h3>
              <div className="grid grid-cols-2 gap-1.5">
                {Object.entries(report.rewards).filter(([, v]) => v > 0).map(([res, val]) => (
                  <div key={res} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-green-500/10 border border-green-500/20">
                    <span className="text-[10px] text-[var(--color-text-secondary)]">{RESOURCE_LABELS[res] ?? res}</span>
                    <span className="text-xs font-bold text-green-600">+{val.toLocaleString()}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {!hasRewards && isVictory && (
            <p className="text-xs text-[var(--color-text-muted)] text-center">敌方城池资源已空</p>
          )}

          {/* Overflow → CityGold */}
          {(report.overflowCityGold ?? 0) > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">仓库溢出转城金</h3>
              <div className="grid grid-cols-2 gap-1.5">
                {Object.entries(report.overflow ?? {}).filter(([, v]) => v > 0).map(([res, val]) => (
                  <div key={res} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-amber-500/5 border border-amber-500/20">
                    <span className="text-[10px] text-[var(--color-text-secondary)]">{RESOURCE_LABELS[res] ?? res}</span>
                    <span className="text-[10px] text-amber-600">溢出 {val.toLocaleString()}</span>
                  </div>
                ))}
              </div>
              <p className="text-[10px] text-amber-600 font-medium mt-1.5 text-center">
                🪙 +{report.overflowCityGold} 城金
              </p>
            </div>
          )}

          {/* 触发的特性 */}
          {report.traitTriggered && report.traitTriggered.length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">将领特性触发</h3>
              <div className="space-y-1.5">
                {report.traitTriggered.map((traitId) => {
                  const meta = getTraitMeta(traitId)
                  const outcome = report.traitOutcomes?.[traitId]
                  return (
                    <div key={traitId} className="flex items-start gap-2 px-2.5 py-1.5 rounded-lg bg-amber-500/10 border border-amber-500/30">
                      <span className="text-base">{meta.icon}</span>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-bold text-amber-600">{meta.name}</span>
                          <span className="text-[10px] text-amber-600/70">{meta.trigger}</span>
                        </div>
                        {outcome?.detail && (
                          <div className="mt-0.5 text-[10px] text-[var(--color-text-secondary)]">
                            {Object.entries(outcome.detail).map(([k, v]) => (
                              <span key={k} className="mr-2">
                                {formatOutcomeDetail(k, v)}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* 美人计：俘虏到军队 */}
          {report.capturedUnits && Object.keys(report.capturedUnits).length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-pink-500 mb-1.5">🌸 美人计·俘虏归队</h3>
              <div className="flex flex-wrap gap-1.5">
                {Object.entries(report.capturedUnits).filter(([, v]) => v > 0).map(([unitType, count]) => (
                  <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                    {getUnitName(unitType)} +{count}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* 美人计：俘虏到驻防 */}
          {report.capturedToGarrison && Object.keys(report.capturedToGarrison).length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-pink-500 mb-1.5">🌸 美人计·俘虏驻防</h3>
              <div className="flex flex-wrap gap-1.5">
                {Object.entries(report.capturedToGarrison).filter(([, v]) => v > 0).map(([unitType, count]) => (
                  <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                    {getUnitName(unitType)} +{count}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* 仁德：复活 */}
          {report.revivedUnits && Object.keys(report.revivedUnits).length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-emerald-500 mb-1.5">🕊️ 仁德·复活归队</h3>
              <div className="flex flex-wrap gap-1.5">
                {Object.entries(report.revivedUnits).filter(([, v]) => v > 0).map(([unitType, count]) => (
                  <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-emerald-500/10 text-emerald-600 font-medium">
                    {getUnitName(unitType)} +{count}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-[var(--color-border)] space-y-2">
          <p className="text-[10px] text-[var(--color-text-muted)] text-center">
            详细情报请前往
            <button
              type="button"
              onClick={() => { handleClose(); navigate(`/report/${report.id}`) }}
              className="text-[var(--color-accent)] font-medium hover:underline cursor-pointer mx-0.5"
            >
              战报详情
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
          <button
            type="button"
            onClick={() => {
              const url = `${window.location.origin}/report/${report.id}`
              navigator.clipboard.writeText(url)
              setCopied(true)
              setTimeout(() => setCopied(false), 2000)
            }}
            className="w-full flex items-center justify-center gap-1.5 px-4 py-2 rounded-xl text-xs font-medium text-blue-500 bg-blue-500/10 hover:bg-blue-500/20 cursor-pointer transition-colors"
          >
            {copied ? <Check size={12} /> : <Share2 size={12} />}
            {copied ? '链接已复制' : '分享战报'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default BattleResultModal
