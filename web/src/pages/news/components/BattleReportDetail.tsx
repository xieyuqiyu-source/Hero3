/* 本文件编排标准战报详情，优先使用 BattleReport.detail 渲染。 */
import { type FC, useMemo, useState } from 'react'
import { gameApi } from '@/api/game'
import { useAccountStore } from '@/store/accountStore'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type { BattleReport, BattleReportDetailData, BattleReportGeneral, BattleReportSide, BattleReportUnit, DefenseReinforcementUnit, General, GeneralTraitInstance } from '@/types/game'
import { GENERAL_TRAITS, formatParamLabel, formatParamValue, getTraitMeta } from '@/utils/traits'
import { sortUnitIds } from '@/utils/unitOrder'
import { buildReportShareURL, resolveReportOutcome } from '../reportPresentation'
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
  return (pvp.reinforcements ?? []).map((reinforcement, index) => {
    const view = reinforcement as DefenseReinforcementUnit & { fromPlayerName?: string; sourceName?: string }
    const losses = pvp.reinforcementLosses?.[reinforcement.reinforcementId] ?? {}
    const sourceType = reinforcement.sourceTags?.source_type
    const sourcePlayerId = reinforcement.sourceTags?.source_player_id || reinforcement.fromPlayerId
    const playerName = view.fromPlayerName || view.sourceName || playerNameById[sourcePlayerId] || (sourceType === 'obtained' ? '获得驻防' : reinforcement.fromPlayerId)
    return {
      role: 'reinforcement',
      playerId: reinforcement.fromPlayerId,
      playerName,
      cityName: `增援方 ${index + 1}`,
      faction: reinforcement.faction,
      factionLabel: reinforcement.faction,
      power: 0,
      generals: (reinforcement.generals ?? []).map((general) => ({
        id: general.id,
        name: general.name,
        level: general.level,
        role: 'reinforcement',
        attributes: general.attributes,
        traits: general.traits,
      })),
      units: buildRowsFromTroopMaps(reinforcement.faction, reinforcement.troops ?? {}, losses, unitsConfig),
    }
  })
}

// sideLossFeedback 生成当前区块的战损反馈。
function sideLossFeedback(side: BattleReportSide): string {
  const totalLost = (side.units ?? []).reduce((sum, unit) => sum + (unit.lost || 0), 0)
  if (totalLost <= 0) return '无'
  return `本方阵亡 ${totalLost.toLocaleString()} 兵`
}

// fallbackGeneralTraits 为旧战报补全未持久化的武将特性。
function fallbackGeneralTraits(general: BattleReportGeneral, currentGenerals: General[]): GeneralTraitInstance[] {
  if ((general.traits ?? []).length > 0) return general.traits ?? []
  const current = currentGenerals.find((item) => item.id === general.id)
  if ((current?.traits ?? []).length > 0) return current?.traits ?? []
  return (GENERAL_TRAITS[general.id] ?? []).map((traitId) => {
    const meta = getTraitMeta(traitId)
    return { traitId, name: meta.name, params: {} }
  })
}

// sideTraitText 将当前参战方将领自带特性压缩成详细文字。
function sideTraitText(side: BattleReportSide, currentGenerals: General[]): string {
  const generals = side.generals ?? []
  if (generals.length === 0) return '-'
  const items = generals.flatMap((general) => {
    const traits = fallbackGeneralTraits(general, currentGenerals)
    return traits.map((trait) => {
      const meta = getTraitMeta(trait.traitId)
      const params = Object.entries(trait.params ?? {})
        .map(([key, value]) => `${formatParamLabel(key)} ${formatParamValue(key, value)}`)
        .join('，')
      return [
        `${general.name || general.id}：${meta.name}`,
        meta.trigger ? `触发 ${meta.trigger}` : '',
        meta.description,
        params ? `参数 ${params}` : '',
      ].filter(Boolean).join('，')
    })
  })
  return items.join('；') || '-'
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
  const currentState = useGameStore((state) => state.state)
  const currentGenerals = currentState?.generals ?? []
  const [copied, setCopied] = useState(false)
  const playerNameById = useMemo(() => {
    const entries = accountPlayers.map((player) => [player.id, player.nickname])
    if (currentState?.player) entries.push([currentState.player.id, currentState.player.nickname])
    return Object.fromEntries(entries)
  }, [accountPlayers, currentState?.player])
  const attackerSide = withExpandedUnits(detail.primarySide, unitsConfig)
  const defenderSide = detail.secondarySide ? withExpandedUnits(detail.secondarySide, unitsConfig) : null
  const reinforcementSides = detail.viewType === 'reinforcement' ? [] : buildReinforcementSides(detail, unitsConfig, playerNameById)
  const attackerTraitText = sideTraitText(attackerSide, currentGenerals)
  const defenderTraitText = defenderSide ? sideTraitText(defenderSide, currentGenerals) : '-'

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

      <BattleParticipantBlock
        title={detail.viewType === 'reinforcement' ? '增援方' : '攻击方'}
        side={attackerSide}
        rewards={detail.ownerSide === 'defender' ? undefined : detail.rewards}
        feedback={sideLossFeedback(attackerSide)}
        effectText={attackerTraitText}
        effectTone={attackerTraitText !== '-' ? 'highlight' : 'normal'}
        result={participantResult(detail, attackerSide)}
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
          effectTone={defenderTraitText !== '-' ? 'highlight' : 'normal'}
          result={participantResult(detail, defenderSide)}
          showUnits={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyRemainingUnits}
          showResources={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyResources}
          showGenerals={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyGenerals}
        />
      )}

      {reinforcementSides.map((side, index) => {
        const reinforcementTraitText = sideTraitText(side, currentGenerals)
        return (
          <BattleParticipantBlock
            key={`${side.playerId || 'reinforcement'}-${index}`}
            title="增援方"
            side={side}
            feedback={sideLossFeedback(side)}
            effectText={reinforcementTraitText}
            effectTone={reinforcementTraitText !== '-' ? 'highlight' : 'normal'}
            result={participantResult(detail, side)}
            showUnits={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyRemainingUnits}
            showResources={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyResources}
            showGenerals={!['attacker', 'scout'].includes(detail.ownerSide || '') || detail.visibility.showEnemyGenerals}
          />
        )
      })}

      <ReportScoutContext scout={detail.extra?.scout} />
      <ReportSweepContext sweep={detail.extra?.sweep} />
      <ReportReinforcementContext reinforcement={detail.extra?.reinforcement} />
      <ReportYellowTurbanContext yellowTurban={detail.extra?.yellowTurban} />
    </div>
  )
}

export default BattleReportDetail
