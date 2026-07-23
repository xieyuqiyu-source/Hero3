/* 本文件编排标准战报详情，优先使用 BattleReport.detail 渲染。 */
import { type FC, useMemo, useState } from 'react'
import { gameApi } from '@/api/game'
import { useAccountStore } from '@/store/accountStore'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type {
  BattleReport,
  BattleReportDetailData,
  BattleReportSide,
  BattleReportUnit,
  DefenseReinforcementUnit,
} from '@/types/game'
import { formatTraitOutcomeDetails } from '@/utils/traits'
import { sortUnitEntries, sortUnitIds } from '@/utils/unitOrder'
import { buildReportShareURL, normalizeBattleReportDetail, reportTraitBelongsToSide } from '../reportPresentation'
import { aggregateReinforcementSnapshots } from '../reportAggregation'
import BattleReportHeader from './report-detail/BattleReportHeader'
import BattleParticipantBlock from './report-detail/BattleParticipantBlock'
import ReportEffectStrip from './report-detail/ReportEffectStrip'
import ReportIntelPanel from './report-detail/ReportIntelPanel'
import ReportReinforcementContext from './report-detail/ReportReinforcementContext'
import ReportScoutContext from './report-detail/ReportScoutContext'
import ReportSweepContext from './report-detail/ReportSweepContext'
import ReportYellowTurbanContext from './report-detail/ReportYellowTurbanContext'

interface BattleReportDetailProps {
  report: BattleReport
  onBack: () => void
}

type UnitsConfigView = Record<string, Record<string, { name?: string; role?: string }>>

interface PvpReportExtra {
  reinforcements?: DefenseReinforcementUnit[]
  reinforcementLosses?: Record<string, Record<string, number>>
  wall?: unknown
}

type PlayerNameMap = Record<string, string>
type ReinforcementDisplaySide = BattleReportSide & {
  generalExpGained?: number
  generalLevelBefore?: number
  generalLevelAfter?: number
}

// readPvpExtra 读取标准详情里的 PVP 扩展数据。
function readPvpExtra(detail: BattleReportDetailData): PvpReportExtra {
  const raw = detail.extra?.pvp
  if (!raw || typeof raw !== 'object') return {}
  return raw as PvpReportExtra
}

// buildAllUnitRows 按阵营配置补齐所有兵种行，确保 0 数量兵种也展示。
function buildAllUnitRows(faction: string | undefined, units: BattleReportUnit[] | undefined, unitsConfig: UnitsConfigView | undefined): BattleReportUnit[] {
  const existing = new Map((units ?? []).map((unit) => [unit.unitType, unit]))
  const configured = Object.entries(unitsConfig?.[faction || ''] ?? {})
    .filter(([, config]) => config.role !== 'transport')
    .map(([unitType]) => unitType)
  const ids = sortUnitIds([...new Set([...configured, ...Array.from(existing.keys())])], faction || '', unitsConfig)
  return ids.map((unitType) => {
    const current = existing.get(unitType)
    if (current) {
      return {
        ...current,
        unitName: current.unitName || unitsConfig?.[current.faction || faction || '']?.[unitType]?.name || unitType,
        faction: current.faction || faction,
      }
    }
    return {
      unitType,
      unitName: unitsConfig?.[faction || '']?.[unitType]?.name || unitType,
      faction,
      amountBefore: 0,
      dispatched: 0,
      lost: 0,
      survived: 0,
    }
  })
}

// buildRowsFromTroopMaps 将增援快照兵力 map 转成完整兵种矩阵。
function buildRowsFromTroopMaps(faction: string, troops: Record<string, number>, losses: Record<string, number>, unitsConfig: UnitsConfigView | undefined): BattleReportUnit[] {
  const configured = Object.entries(unitsConfig?.[faction] ?? {})
    .filter(([, config]) => config.role !== 'transport')
    .map(([unitType]) => unitType)
  const ids = sortUnitIds([...new Set([...configured, ...Object.keys(troops), ...Object.keys(losses)])], faction, unitsConfig)
  return ids.map((unitType) => {
    const dispatched = troops[unitType] ?? 0
    const lost = losses[unitType] ?? 0
    return {
      unitType,
      unitName: unitsConfig?.[faction]?.[unitType]?.name || unitType,
      faction,
      amountBefore: dispatched,
      dispatched,
      lost,
      survived: Math.max(0, dispatched - lost),
    }
  })
}

// withExpandedUnits 返回补齐全部兵种后的参战方。
function withExpandedUnits(side: BattleReportSide, unitsConfig: UnitsConfigView | undefined): BattleReportSide {
  return {
    ...side,
    units: buildAllUnitRows(side.faction, side.units, unitsConfig),
  }
}

// buildReinforcementSides 从战报 extra 中构造多个协防/增援方区块。
function buildReinforcementSides(detail: BattleReportDetailData, unitsConfig: UnitsConfigView | undefined, playerNameById: PlayerNameMap): ReinforcementDisplaySide[] {
  const pvp = readPvpExtra(detail)
  const groups = aggregateReinforcementSnapshots(pvp.reinforcements ?? [], pvp.reinforcementLosses ?? {})
  return groups.map((group, index) => {
    const playerName = playerNameById[group.sourcePlayerId]
      || group.playerName
      || (group.sourceType === 'obtained' ? '获得驻防' : group.sourceId)
    return {
      role: 'reinforcement',
      targetId: group.groupKey,
      playerId: group.sourcePlayerId,
      playerName,
      cityId: group.sourcePlayerId ? undefined : group.sourceId,
      cityName: group.sourcePlayerId ? undefined : group.playerName || `增援方 ${index + 1}`,
      faction: group.faction,
      factionLabel: group.faction,
      power: 0,
      generalExpGained: group.generalExpGained,
      generalLevelBefore: group.generalLevelBefore,
      generalLevelAfter: group.generalLevelAfter,
      generals: group.generals.map((general) => ({
        id: general.id,
        name: general.name,
        level: general.level,
        role: 'reinforcement',
        generalExpGained: general.generalExpGained,
        generalLevelBefore: general.generalLevelBefore,
        generalLevelAfter: general.generalLevelAfter,
        stats: general.stats,
        effectiveStats: general.effectiveStats,
        attributes: general.attributes,
        buffs: general.buffs,
        traits: general.traits,
      })),
      units: buildRowsFromTroopMaps(group.faction, group.troops, group.losses, unitsConfig),
    }
  })
}

// sideLossFeedback 生成当前区块的战损反馈。
function sideLossFeedback(side: BattleReportSide): string {
  const totalLost = (side.units ?? []).reduce((sum, unit) => sum + (unit.lost || 0), 0)
  if (totalLost <= 0) return '无'
  return `本方阵亡 ${totalLost.toLocaleString()} 兵`
}

// sideTriggeredEffectText 展示本场真实触发结果和结算数值，并兼容旧黄巾防守战报的错误主侧标记。
function sideTriggeredEffectText(
  detail: BattleReportDetailData,
  side: BattleReportSide,
  sideKey: 'primary' | 'secondary' | 'reinforcement',
  unitsConfig?: UnitsConfigView,
): string {
  const visible = (detail.traits ?? []).filter((trait) => reportTraitBelongsToSide(detail, trait, side, sideKey))
  if (visible.length === 0) return '本场无触发效果'
  return visible.map((trait) => {
    const name = trait.traitName || trait.traitId
    const summary = trait.summary?.trim() === name.trim() ? '' : trait.summary
    const detailText = formatTraitOutcomeDetails(trait.detail, {
      faction: side.faction,
      units: unitsConfig,
      sortUnitEntries,
    })
    return [trait.generalName ? `${trait.generalName}：${name}` : name, summary, detailText].filter(Boolean).join(' · ')
  }).join('；')
}

// sidePassiveEffectText 从参战快照展示真实生效的永久被动，不混入触发时间线。
function sidePassiveEffectText(side: BattleReportSide): string {
  const definitions: Record<string, { name: string; unitName: string }> = {
    jixing_benxi: { name: '疾行奔袭', unitName: '骁骑营' },
    huhu_shengwei: { name: '虎虎生威', unitName: '虎豹骑' },
    qibing_raohou: { name: '奇兵绕后', unitName: '南蛮象' },
  }
  const lines = (side.generals ?? []).flatMap((general) => general.traits ?? []).flatMap((trait) => {
    if (trait.traitId === 'shengui_zhicai') {
      const politics = Number(trait.params?.politicsBonus ?? 0)
      const intelligence = Number(trait.params?.intelligenceBonus ?? 0)
      if (politics <= 0 && intelligence <= 0) return []
      return [`神鬼之才 · 内政 +${politics.toLocaleString()}，智谋 +${intelligence.toLocaleString()}`]
    }
    if (trait.traitId === 'rende') {
      const politics = Number(trait.params?.politicsBonus ?? 0)
      const command = Number(trait.params?.commandBonus ?? 0)
      if (politics <= 0 && command <= 0) return []
      return [`仁德天下 · 内政 +${politics.toLocaleString()}，统率 +${command.toLocaleString()}`]
    }
    if (trait.traitId === 'laodang_yizhuang') {
      const force = Number(trait.params?.forceBonus ?? 0)
      const command = Number(trait.params?.commandBonus ?? 0)
      if (force <= 0 && command <= 0) return []
      return [`老当益壮 · 武力 +${force.toLocaleString()}，统率 +${command.toLocaleString()}`]
    }
    const definition = definitions[trait.traitId]
    if (!definition) return []
    const attack = Number(trait.params?.unitAttackFlat ?? 0)
    const speed = Number(trait.params?.unitSpeedFlat ?? 0)
    if (attack <= 0 && speed <= 0) return []
    return [`${definition.name} · ${definition.unitName}攻击 +${attack.toLocaleString()}，移动 +${speed.toLocaleString()}`]
  })
  return lines.length > 0 ? `被动生效：${lines.join('；')}` : ''
}

// participantResult 按战报胜方计算当前参与方的胜败印。
function participantResult(detail: BattleReportDetailData, side: BattleReportSide): 'victory' | 'defeat' | 'draw' | 'none' {
  const winnerSide = detail.winnerSide || (detail.result === 'attacker_victory' ? 'attacker' : detail.result === 'defender_victory' ? 'defender' : detail.result)
  if (winnerSide === 'draw') return 'draw'
  if (winnerSide !== 'attacker' && winnerSide !== 'defender') return 'none'
  const participantSide = side.role === 'reinforcement' ? 'defender' : side.role
  if (participantSide !== 'attacker' && participantSide !== 'defender') return 'none'
  return participantSide === winnerSide ? 'victory' : 'defeat'
}

// BattleReportDetail 渲染新版高密度战报详情。
const BattleReportDetail: FC<BattleReportDetailProps> = ({ report, onBack }) => {
  const detail = useMemo(() => normalizeBattleReportDetail(report), [report])
  const unitsConfig = useConfigStore((state) => state.units) as UnitsConfigView | undefined
  const accountPlayers = useAccountStore((state) => state.players)
  const currentPlayer = useGameStore((state) => state.state?.player)
  const [copied, setCopied] = useState(false)
  const playerNameById = useMemo(() => {
    const entries = accountPlayers.map((player) => [player.id, player.nickname])
    if (currentPlayer) entries.push([currentPlayer.id, currentPlayer.nickname])
    return Object.fromEntries(entries)
  }, [accountPlayers, currentPlayer])
  const attackerSide = withExpandedUnits(detail.primarySide, unitsConfig)
  const defenderSide = detail.secondarySide ? withExpandedUnits(detail.secondarySide, unitsConfig) : null
  const reinforcementSides = buildReinforcementSides(detail, unitsConfig, playerNameById)
  const attackerTraitText = sideTriggeredEffectText(detail, attackerSide, 'primary', unitsConfig)
  const defenderTraitText = defenderSide ? sideTriggeredEffectText(detail, defenderSide, 'secondary', unitsConfig) : '本场无触发效果'
  const attackerGeneralExp = attackerSide.generals?.[0]?.generalExpGained ?? (detail.ownerSide === 'attacker' ? detail.rewards.generalExp : undefined)
  const defenderGeneralExp = defenderSide?.generals?.[0]?.generalExpGained ?? (detail.ownerSide === 'defender' ? detail.rewards.generalExp : undefined)
  const showCombatSettlement = detail.viewType !== 'reinforcement' && detail.viewType !== 'scout' && detail.battleType !== 'scout'

  // handleShare 创建或复用分享 token 并复制公开链接。
  const handleShare = async () => {
    let token = report.share?.token || detail.share?.token
    if (!token) {
      const activePlayerId = useGameStore.getState().activePlayerId
      if (activePlayerId) {
        const link = await gameApi.shareReport(activePlayerId, report.id)
        token = link.token
      }
    }
    const shareURL = buildReportShareURL(window.location.origin, report, token)
    if (!shareURL) return
    await navigator.clipboard.writeText(shareURL)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-3">
      <BattleReportHeader detail={detail} copied={copied} onBack={onBack} onShare={handleShare} />

      <ReportEffectStrip detail={detail} />
      <ReportIntelPanel visibility={detail.visibility} />
      {detail.viewType === 'reinforcement' && <ReportReinforcementContext reinforcement={detail.extra?.reinforcement} />}

      <BattleParticipantBlock
        title={attackerSide.role === 'reinforcement' ? '增援方' : '攻击方'}
        side={attackerSide}
        rewards={detail.ownerSide === 'defender' ? undefined : detail.rewards}
        feedback={sideLossFeedback(attackerSide)}
        effectText={attackerTraitText}
        passiveText={sidePassiveEffectText(attackerSide)}
        effectTone={attackerTraitText === '本场无触发效果' ? 'normal' : 'highlight'}
        result={participantResult(detail, attackerSide)}
        generalExp={attackerGeneralExp}
        generalLevelBefore={attackerSide.generals?.[0]?.generalLevelBefore ?? (detail.ownerSide === 'attacker' ? detail.rewards.generalLevelBefore : undefined)}
        generalLevelAfter={attackerSide.generals?.[0]?.generalLevelAfter ?? (detail.ownerSide === 'attacker' ? detail.rewards.generalLevelAfter : undefined)}
        settlement={showCombatSettlement ? 'attacker' : 'none'}
        showUnits={detail.ownerSide !== 'defender' || detail.visibility.showEnemyRemainingUnits}
        showResources={detail.ownerSide !== 'defender' || detail.visibility.showEnemyResources}
        showGenerals={detail.ownerSide !== 'defender' || detail.visibility.showEnemyGenerals}
      />

      {(defenderSide || reinforcementSides.length > 0) && (
        <div className="relative flex items-center justify-center py-2">
          <div className="absolute left-0 right-0 h-px bg-[var(--color-border)]" />
          <span className="relative rounded-md border border-amber-500/40 bg-[var(--color-surface)] px-5 py-1 text-xs font-black text-amber-500">
            交战
          </span>
        </div>
      )}

      {defenderSide && (
        <BattleParticipantBlock
          title="防守方"
          side={defenderSide}
          rewards={detail.ownerSide === 'defender' ? detail.rewards : undefined}
          feedback={sideLossFeedback(defenderSide)}
          effectText={defenderTraitText}
          passiveText={sidePassiveEffectText(defenderSide)}
          effectTone={defenderTraitText === '本场无触发效果' ? 'normal' : 'highlight'}
          result={participantResult(detail, defenderSide)}
          generalExp={defenderGeneralExp}
          generalLevelBefore={defenderSide.generals?.[0]?.generalLevelBefore ?? (detail.ownerSide === 'defender' ? detail.rewards.generalLevelBefore : undefined)}
          generalLevelAfter={defenderSide.generals?.[0]?.generalLevelAfter ?? (detail.ownerSide === 'defender' ? detail.rewards.generalLevelAfter : undefined)}
          settlement={showCombatSettlement ? 'defender' : 'none'}
          showUnits={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyRemainingUnits}
          showResources={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyResources}
          showGenerals={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyGenerals}
        />
      )}

      {reinforcementSides.map((side) => {
        const reinforcementTraitText = sideTriggeredEffectText(detail, side, 'reinforcement', unitsConfig)
        return (
          <BattleParticipantBlock
            key={side.targetId}
            title="增援方"
            side={side}
            feedback={sideLossFeedback(side)}
            effectText={reinforcementTraitText}
            passiveText={sidePassiveEffectText(side)}
            effectTone={reinforcementTraitText === '本场无触发效果' ? 'normal' : 'highlight'}
            result={participantResult(detail, side)}
            generalExp={side.generalExpGained}
            generalLevelBefore={side.generalLevelBefore}
            generalLevelAfter={side.generalLevelAfter}
            settlement="none"
            showUnits={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyRemainingUnits}
            showResources={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyResources}
            showGenerals={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyGenerals}
          />
        )
      })}

      <ReportScoutContext scout={detail.extra?.scout} />
      <ReportSweepContext sweep={detail.extra?.sweep} />
      {detail.viewType !== 'reinforcement' && <ReportReinforcementContext reinforcement={detail.extra?.reinforcement} />}
      <ReportYellowTurbanContext yellowTurban={detail.extra?.yellowTurban} />
    </div>
  )
}

export default BattleReportDetail
