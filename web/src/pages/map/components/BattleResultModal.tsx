/* 本文件实现战斗结算弹窗，展示战损、奖励、参战武将经验和特性结果。 */
import { useState, useEffect, type FC } from 'react'
import { Trophy, Skull, X, Share2, Check } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { BattleReport, BattleReportSweepDefender } from '@/types/game'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import { formatTraitOutcomeDetail, getTraitMeta } from '@/utils/traits'
import { sortUnitEntries } from '@/utils/unitOrder'
import { mergeBattleReportDrops } from '@/utils/reportDrops'
import { gameApi } from '@/api/game'
import { buildReportShareURL } from '../../news/reportPresentation'

interface BattleResultModalProps {
  report: BattleReport
  onClose: () => void
}

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
const DROP_QUALITY_CLASS: Record<string, string> = {
  common: 'text-slate-600 bg-slate-500/10 border-slate-500/20',
  rare: 'text-blue-600 bg-blue-500/10 border-blue-500/20',
  epic: 'text-purple-600 bg-purple-500/10 border-purple-500/20',
  legendary: 'text-amber-600 bg-amber-500/10 border-amber-500/25',
}

// readSweepDefenders 读取扫荡聚合战报中的多个 NPC 防守方快照。
function readSweepDefenders(report: BattleReport): BattleReportSweepDefender[] {
  const defenders = report.detail?.extra?.sweep?.defenders
  return Array.isArray(defenders) ? defenders : []
}

const BattleResultModal: FC<BattleResultModalProps> = ({ report, onClose }) => {
  const [visible, setVisible] = useState(false)
  const [copied, setCopied] = useState(false)
  const navigate = useNavigate()
  const nickname = useGameStore((s) => s.state?.player.nickname ?? '我方')
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const activePlayerId = useGameStore((s) => s.activePlayerId)
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
    for (const f of Object.values(units ?? {})) {
      if (f[unitType]?.name) return f[unitType].name
    }
    return factionUnits[unitType]?.name ?? unitType
  }

  const hasRewards = Object.values(report.rewards).some(v => v > 0)
  const hasLosses = Object.values(report.lostUnits).some(v => v > 0)
  const pvpPointEntries = Object.entries(report.pvpPointsDelta ?? {}).filter(([, amount]) => amount !== 0)
  const pvpReinforcements = report.pvpReinforcements ?? []
  const pvpReinforcementCount = pvpReinforcements.length
  const pvpGeneralCount = (report.pvpAttackerGenerals?.length ?? 0) + (report.pvpDefenderGenerals?.length ?? 0)
  const generalExpGained = report.generalExpGained ?? report.detail?.rewards?.generalExp ?? 0
  const generalLevelBefore = report.generalLevelBefore ?? report.detail?.rewards?.generalLevelBefore
  const generalLevelAfter = report.generalLevelAfter ?? report.detail?.rewards?.generalLevelAfter
  const drops = report.drops ?? report.detail?.rewards?.drops ?? []
  const mergedDrops = mergeBattleReportDrops(drops)
  const sweepDefenders = readSweepDefenders(report)
  const isSweepReport = report.battleType === 'sweep' || sweepDefenders.length > 0
  const sweepExtra = report.detail?.extra?.sweep
  const sweepLossTotal = Object.values(report.defenderLostUnits ?? {}).reduce((sum, amount) => sum + Math.max(0, amount), 0)

  // handleShare 创建公开令牌后复制分享地址，内部战报 ID 不再作为匿名入口。
  const handleShare = async () => {
    let token = report.share?.token || report.detail?.share?.token
    if (!token && activePlayerId) {
      const link = await gameApi.shareReport(activePlayerId, report.id)
      token = link.token
    }
    if (!token) return
    const shareURL = buildReportShareURL(window.location.origin, report, token)
    if (!shareURL) return
    await navigator.clipboard.writeText(shareURL)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
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
        relative w-full max-w-sm max-h-[calc(100vh-2rem)] rounded-2xl overflow-hidden flex flex-col
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
        <div className="px-4 py-3 space-y-3 overflow-y-auto">
          {/* Player VS NPC */}
          <div className="flex items-center justify-center gap-3 px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
            <span className="text-sm font-bold text-[var(--color-text-primary)]">{nickname}</span>
            <span className="text-xs font-bold text-[var(--color-text-muted)]">VS</span>
            <span className="text-sm font-bold text-[var(--color-text-primary)]">{report.targetName}</span>
          </div>

          {isSweepReport && (
            <div className="rounded-xl border border-emerald-500/25 bg-emerald-500/10 px-3 py-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-[11px] font-semibold text-emerald-600">扫荡汇总</span>
                <span className="text-[10px] font-bold text-emerald-600">
                  成功 {(sweepExtra?.success ?? sweepDefenders.length).toLocaleString()} / 请求 {(sweepExtra?.requested ?? sweepDefenders.length).toLocaleString()}
                </span>
              </div>
              <div className="mt-2 grid grid-cols-3 gap-1.5">
                <SummaryPill label="失败" value={(sweepExtra?.failed ?? 0).toLocaleString()} />
                <SummaryPill label="击溃" value={sweepLossTotal.toLocaleString()} />
                <SummaryPill label="状态" value={sweepExtra?.stopped ? '中止' : '完成'} />
              </div>
              <p className="mt-2 text-[10px] leading-relaxed text-[var(--color-text-muted)]">
                这里只保留本次扫荡结果摘要，逐城仅保存目标和胜负，完整过程请查看服务日志。
              </p>
            </div>
          )}

          {generalExpGained > 0 && (
            <div className="flex items-center justify-between px-3 py-2 rounded-xl bg-amber-500/10 border border-amber-500/25">
              <span className="text-[11px] font-semibold text-amber-600">将领经验</span>
              <span className="text-[11px] font-bold text-amber-600">
                +{generalExpGained}
                {generalLevelAfter && generalLevelBefore && generalLevelAfter > generalLevelBefore
                  ? ` Lv.${generalLevelBefore} → Lv.${generalLevelAfter}`
                  : ''}
              </span>
            </div>
          )}

          {(pvpPointEntries.length > 0 || pvpGeneralCount > 0 || pvpReinforcementCount > 0) && (
            <div className="rounded-xl bg-indigo-500/10 border border-indigo-500/25 px-3 py-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-[11px] font-semibold text-indigo-600">参战信息</span>
                <span className="text-[10px] text-indigo-600">
                  {[pvpGeneralCount > 0 ? `武将 ${pvpGeneralCount} 位` : '', pvpReinforcementCount > 0 ? `驻防/援军 ${pvpReinforcementCount} 队` : ''].filter(Boolean).join(' · ')}
                </span>
              </div>
              {pvpPointEntries.length > 0 && (
                <div className="mt-1 flex flex-wrap gap-1.5">
                  {pvpPointEntries.map(([key, amount]) => (
                    <span key={key} className={`text-[10px] px-2 py-0.5 rounded-lg font-medium ${amount > 0 ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'}`}>
                      {key === 'self' ? '我方' : key === 'target' ? '对方' : key} {amount > 0 ? '+' : ''}{amount}
                    </span>
                  ))}
                </div>
              )}
              {pvpReinforcements.length > 0 && (
                <div className="mt-2 space-y-1.5">
                  {pvpReinforcements.map((item) => (
                    <div key={item.reinforcementId} className="rounded-lg border border-indigo-500/15 bg-white/55 px-2 py-1.5 dark:bg-white/5">
                      <div className="flex items-center gap-2">
                        <span className="shrink-0 text-[10px] font-black text-indigo-600">{item.sourceTags?.source_type === 'obtained' ? '获得驻防' : '增援驻防'}</span>
                        <span className="min-w-0 flex-1 truncate text-[10px] text-[var(--color-text-muted)]">{item.fromPlayerId}</span>
                      </div>
                      <div className="mt-1 flex flex-wrap gap-1">
                        {Object.entries(item.troops ?? {}).filter(([, amount]) => amount > 0).map(([unitType, amount]) => (
                          <span key={unitType} className="rounded bg-indigo-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-indigo-600">
                            {getUnitName(unitType)} {amount.toLocaleString()}
                          </span>
                        ))}
                      </div>
                      {item.generals && item.generals.length > 0 && (
                        <div className="mt-1 text-[10px] font-semibold text-emerald-600">
                          武将：{item.generals.map((general) => `${general.name || general.id}${general.level ? ` Lv.${general.level}` : ''}`).join('、')}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Losses */}
          {hasLosses && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">我方损失</h3>
              <div className="flex flex-wrap gap-1.5">
                {sortUnitEntries(report.lostUnits, faction, units ?? undefined).filter(([, v]) => v > 0).map(([unitType, count]) => (
                  <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-red-500/10 text-red-600 font-medium">
                    {getUnitName(unitType)} ×{count}
                  </span>
                ))}
              </div>
            </div>
          )}

          {!isSweepReport && sweepDefenders.length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">防守方战损</h3>
              <div className="space-y-1.5">
                {sweepDefenders.map((defender) => {
                  const losses = (defender.units ?? []).filter((unit) => unit.lost > 0)
                  const total = losses.reduce((sum, unit) => sum + unit.lost, 0)
                  return (
                    <div key={defender.targetId} className="rounded-lg border border-blue-500/15 bg-blue-500/5 px-2.5 py-1.5">
                      <div className="flex items-center justify-between gap-2">
                        <span className="min-w-0 truncate text-[10px] font-bold text-blue-600">{defender.targetName || defender.targetId}</span>
                        <span className="shrink-0 text-[10px] font-bold text-red-600">阵亡 {total.toLocaleString()}</span>
                      </div>
                      <div className="mt-1 flex flex-wrap gap-1">
                        {losses.length > 0 ? losses.map((unit) => (
                          <span key={unit.unitType} className="rounded bg-red-500/10 px-1.5 py-0.5 text-[9px] font-medium text-red-600">
                            {unit.unitName || getUnitName(unit.unitType)} ×{unit.lost.toLocaleString()}
                          </span>
                        )) : (
                          <span className="text-[9px] text-[var(--color-text-muted)]">无可见战损</span>
                        )}
                      </div>
                    </div>
                  )
                })}
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

          {mergedDrops.length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">宝物掉落</h3>
              <div className="flex flex-wrap gap-1.5">
                {mergedDrops.map((drop) => (
                  <span
                    key={`${drop.itemId ?? drop.name ?? 'drop'}-${drop.quality ?? ''}`}
                    className={`rounded-lg border px-2 py-1 text-[10px] font-bold ${DROP_QUALITY_CLASS[drop.quality ?? ''] ?? DROP_QUALITY_CLASS.common}`}
                  >
                    {drop.name ?? drop.itemId} ×{drop.amount.toLocaleString()}
                  </span>
                ))}
              </div>
            </div>
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
          {!isSweepReport && report.traitTriggered && report.traitTriggered.length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-[var(--color-text-primary)] mb-1.5">将领特性触发</h3>
              <div className="space-y-1.5">
                {report.traitTriggered.map((traitId) => {
                  const outcome = report.traitOutcomes?.[traitId]
                  const displayTraitId = outcome?.traitId || traitId
                  const meta = getTraitMeta(displayTraitId)
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
                                {formatTraitOutcomeDetail(k, v, { faction, units: units ?? undefined, sortUnitEntries })}
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
          {!isSweepReport && report.capturedUnits && Object.keys(report.capturedUnits).length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-pink-500 mb-1.5">🌸 美人计·俘虏归队</h3>
              <div className="flex flex-wrap gap-1.5">
                {sortUnitEntries(report.capturedUnits, faction, units ?? undefined).filter(([, v]) => v > 0).map(([unitType, count]) => (
                  <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                    {getUnitName(unitType)} +{count}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* 美人计：俘虏到驻防 */}
          {!isSweepReport && report.capturedToGarrison && Object.keys(report.capturedToGarrison).length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-pink-500 mb-1.5">🌸 美人计·俘虏驻防</h3>
              <div className="flex flex-wrap gap-1.5">
                {sortUnitEntries(report.capturedToGarrison, faction, units ?? undefined).filter(([, v]) => v > 0).map(([unitType, count]) => (
                  <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                    {getUnitName(unitType)} +{count}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* 战后复活或最终减损返还 */}
          {!isSweepReport && report.revivedUnits && Object.keys(report.revivedUnits).length > 0 && (
            <div>
              <h3 className="text-[11px] font-semibold text-emerald-500 mb-1.5">战后归队兵力</h3>
              <div className="flex flex-wrap gap-1.5">
                {sortUnitEntries(report.revivedUnits, faction, units ?? undefined).filter(([, v]) => v > 0).map(([unitType, count]) => (
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
            onClick={handleShare}
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

// SummaryPill 渲染扫荡摘要中的紧凑指标。
const SummaryPill: FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="rounded-lg border border-emerald-500/20 bg-white/60 px-2 py-1 text-center dark:bg-emerald-500/10">
    <div className="text-[9px] text-[var(--color-text-muted)]">{label}</div>
    <div className="mt-0.5 truncate text-[10px] font-bold text-[var(--color-text-primary)]">{value}</div>
  </div>
)

export default BattleResultModal
