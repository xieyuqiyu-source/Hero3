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
  BattleReportTrait,
  BattleReportUnit,
  DefenseReinforcementUnit,
} from '@/types/game'
import { sortUnitIds } from '@/utils/unitOrder'
import { buildReportShareURL, resolveReportOutcome } from '../reportPresentation'
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

// buildFallbackDetail 为历史战报生成最低限度详情结构。
function buildFallbackDetail(report: BattleReport): BattleReportDetailData {
  const outcome = resolveReportOutcome(report)
  return {
    id: report.id,
    eventId: report.eventId,
    ownerPlayerId: report.ownerPlayerId || report.playerId,
    viewType: report.viewType || (report.type === 'reinforce' ? 'reinforcement' : report.type === 'scout' ? 'scout' : 'attack'),
    viewLabel: report.viewType || '进攻',
    sourceType: report.sourceType || 'npc_city',
    sourceLabel: report.sourceType || 'NPC',
    battleType: report.battleType || report.type,
    result: report.result,
    winnerSide: report.winnerSide,
    ownerSide: report.ownerSide,
    ownerOutcome: outcome,
    title: report.title || `${report.playerName || '我方'} ${report.type} ${report.targetName || report.targetId}`,
    summary: report.summary,
    occurredAt: report.createdAt,
    primarySide: {
      role: report.viewType === 'reinforcement' ? 'reinforcement' : 'attacker',
      playerId: report.playerId,
      playerName: report.playerName,
      cityName: report.playerName,
      faction: report.playerFaction,
      power: report.playerPower,
      units: Object.entries(report.dispatchedUnits ?? {}).map(([unitType, dispatched]) => ({
        unitType,
        amountBefore: dispatched,
        dispatched,
        lost: report.lostUnits?.[unitType] ?? 0,
        survived: Math.max(0, dispatched - (report.lostUnits?.[unitType] ?? 0)),
      })),
    },
    secondarySide: {
      role: 'defender',
      targetId: report.targetId,
      targetName: report.targetName,
      cityName: report.targetName,
      faction: report.defenderFaction,
      power: report.enemyPower,
      units: Object.entries(report.defenderUnits ?? {}).map(([unitType, dispatched]) => ({
        unitType,
        amountBefore: dispatched,
        dispatched,
        lost: report.defenderLostUnits?.[unitType] ?? 0,
        survived: Math.max(0, dispatched - (report.defenderLostUnits?.[unitType] ?? 0)),
      })),
      resources: report.defenderResources,
    },
    rewards: {
      resources: report.rewards,
      drops: report.drops,
      cityGold: report.overflowCityGold,
      generalExp: report.generalExpGained,
      generalLevelBefore: report.generalLevelBefore,
      generalLevelAfter: report.generalLevelAfter,
      overflow: report.overflow,
    },
    traits: [],
    visibility: {
      showEnemyRemainingUnits: report.defenderRevealed,
      showEnemyResources: report.defenderRevealed,
      showEnemyGenerals: report.defenderRevealed,
      showEnemyCityDefense: report.defenderRevealed,
    },
    read: report.read,
    share: report.share,
  }
}

// normalizeDetail 确保详情有新语义字段，旧数据走本地兜底。
function normalizeDetail(report: BattleReport): BattleReportDetailData {
  const detail = report.detail ?? buildFallbackDetail(report)
  return {
    ...detail,
    ownerOutcome: detail.ownerOutcome || report.ownerOutcome || resolveReportOutcome(report),
    winnerSide: detail.winnerSide || report.winnerSide,
    ownerSide: detail.ownerSide || report.ownerSide,
    share: detail.share || report.share,
  }
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
    if (current) return current
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
function buildReinforcementSides(detail: BattleReportDetailData, unitsConfig: UnitsConfigView | undefined, playerNameById: PlayerNameMap): BattleReportSide[] {
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
      generals: group.generals.map((general) => ({
        id: general.id,
        name: general.name,
        level: general.level,
        role: 'reinforcement',
        attributes: general.attributes,
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

// sideTriggeredEffectText 只展示本场真实触发结果，不用当前武将静态特性冒充战斗效果。
function sideTriggeredEffectText(traits: BattleReportTrait[] | undefined, side: BattleReportSide, sideKey: 'primary' | 'secondary' | 'reinforcement'): string {
  const generalIds = new Set((side.generals ?? []).map((general) => general.id))
  const visible = (traits ?? []).filter((trait) => {
    if (trait.generalId && generalIds.has(trait.generalId)) return true
    if (trait.ownerSide === sideKey) return true
    if (trait.ownerRole && trait.ownerRole === side.role) return true
    return sideKey === 'primary' && !trait.ownerSide && !trait.ownerRole && !trait.generalId
  })
  if (visible.length === 0) return '本场无触发效果'
  return visible.map((trait) => {
    const name = trait.traitName || trait.traitId
    return [trait.generalName ? `${trait.generalName}：${name}` : name, trait.summary].filter(Boolean).join(' · ')
  }).join('；')
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
  const detail = useMemo(() => normalizeDetail(report), [report])
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
  const reinforcementSides = detail.viewType === 'reinforcement' ? [] : buildReinforcementSides(detail, unitsConfig, playerNameById)
  const attackerTraitText = sideTriggeredEffectText(detail.traits, attackerSide, 'primary')
  const defenderTraitText = defenderSide ? sideTriggeredEffectText(detail.traits, defenderSide, 'secondary') : '本场无触发效果'
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
        title={detail.viewType === 'reinforcement' ? '增援方' : '攻击方'}
        side={attackerSide}
        rewards={detail.ownerSide === 'defender' ? undefined : detail.rewards}
        feedback={sideLossFeedback(attackerSide)}
        effectText={attackerTraitText}
        effectTone={attackerTraitText === '本场无触发效果' ? 'normal' : 'highlight'}
        result={participantResult(detail, attackerSide)}
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
          effectTone={defenderTraitText === '本场无触发效果' ? 'normal' : 'highlight'}
          result={participantResult(detail, defenderSide)}
          settlement={showCombatSettlement ? 'defender' : 'none'}
          showUnits={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyRemainingUnits}
          showResources={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyResources}
          showGenerals={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyGenerals}
        />
      )}

      {reinforcementSides.map((side) => {
        const reinforcementTraitText = sideTriggeredEffectText(detail.traits, side, 'reinforcement')
        return (
          <BattleParticipantBlock
            key={side.targetId}
            title="增援方"
            side={side}
            feedback={sideLossFeedback(side)}
            effectText={reinforcementTraitText}
            effectTone={reinforcementTraitText === '本场无触发效果' ? 'normal' : 'highlight'}
            result={participantResult(detail, side)}
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
