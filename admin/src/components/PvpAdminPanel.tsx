/* 本文件实现 GM 后台 PVP 只读查询工作台。 */
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { RefreshCw, Swords } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { AccountSummary, AdminPvpOverviewResponse, AdminPvpSeasonListResponse, PlayerSummary, PvpBattle, PvpMarch, PvpRankingEntry } from '@/types'

interface PvpAdminPanelProps {
  accounts: AccountSummary[]
}

// PvpAdminPanel 展示 PVP 状态、排行榜、行军和战斗查询。
export default function PvpAdminPanel({ accounts }: PvpAdminPanelProps) {
  const players = useMemo(() => flattenPlayers(accounts), [accounts])
  const [selectedPlayerId, setSelectedPlayerId] = useState('')
  const [overview, setOverview] = useState<AdminPvpOverviewResponse | null>(null)
  const [seasons, setSeasons] = useState<AdminPvpSeasonListResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [protecting, setProtecting] = useState(false)
  const [settlingSeason, setSettlingSeason] = useState(false)
  const [repairingMarchId, setRepairingMarchId] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    void loadOverview()
  }, [])

  // loadOverview 拉取 PVP 只读总览数据。
  const loadOverview = async () => {
    setLoading(true)
    setError('')
    try {
      const [next, seasonList] = await Promise.all([
        adminApi.getPvpOverview(selectedPlayerId, 100),
        adminApi.getPvpSeasons(),
      ])
      setOverview(next)
      setSeasons(seasonList)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'PVP 数据加载失败')
    } finally {
      setLoading(false)
    }
  }

  // handleSetProtection 设置玩家 PVP 保护并刷新工作台。
  const handleSetProtection = async (protectionType: string, hours: number) => {
    if (!selectedPlayerId) return
    setProtecting(true)
    setError('')
    try {
      await adminApi.setPvpProtection(selectedPlayerId, protectionType, hours, 'GM PVP 保护')
      const next = await adminApi.getPvpOverview(selectedPlayerId, 100)
      setOverview(next)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'PVP 保护设置失败')
    } finally {
      setProtecting(false)
    }
  }

  // reloadCurrentOverview 刷新当前筛选下的 PVP 工作台。
  const reloadCurrentOverview = async () => {
    const [next, seasonList] = await Promise.all([
      adminApi.getPvpOverview(selectedPlayerId, 100),
      adminApi.getPvpSeasons(),
    ])
    setOverview(next)
    setSeasons(seasonList)
  }

  // handleSettleSeason 结算当前 PVP 赛季并刷新工作台。
  const handleSettleSeason = async () => {
    const seasonId = overview?.season?.id || seasons?.current?.id
    if (!seasonId) return
    setSettlingSeason(true)
    setError('')
    try {
      await adminApi.settlePvpSeason(seasonId)
      await reloadCurrentOverview()
    } catch (err) {
      setError(err instanceof Error ? err.message : '赛季结算失败')
    } finally {
      setSettlingSeason(false)
    }
  }

  // handleForceResolve 强制结算指定 PVP 行军。
  const handleForceResolve = async (marchId: string) => {
    setRepairingMarchId(marchId)
    setError('')
    try {
      await adminApi.forceResolvePvpMarch(marchId)
      await reloadCurrentOverview()
    } catch (err) {
      setError(err instanceof Error ? err.message : '强制结算失败')
    } finally {
      setRepairingMarchId('')
    }
  }

  // handleCancelMarch 取消指定 PVP 行军。
  const handleCancelMarch = async (marchId: string) => {
    setRepairingMarchId(marchId)
    setError('')
    try {
      await adminApi.cancelPvpMarch(marchId)
      await reloadCurrentOverview()
    } catch (err) {
      setError(err instanceof Error ? err.message : '取消行军失败')
    } finally {
      setRepairingMarchId('')
    }
  }

  const playerState = overview?.player
  const marches = overview?.marches ?? []
  const battles = overview?.battles ?? []
  const rankings = overview?.rankings ?? []

  return (
    <section className="grid gap-4">
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-sm">
        <div className="flex flex-col gap-3 md:flex-row md:items-center">
          <div className="flex items-center gap-2">
            <Swords size={18} className="text-[var(--color-accent)]" />
            <div>
              <h2 className="text-base font-bold text-[var(--color-text-primary)]">PVP</h2>
              <p className="text-xs text-[var(--color-text-muted)]">查询与修复</p>
            </div>
          </div>
          <div className="flex flex-1 flex-col gap-2 sm:flex-row md:justify-end">
            <select
              value={selectedPlayerId}
              onChange={(event) => setSelectedPlayerId(event.target.value)}
              className="min-h-10 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 text-sm text-[var(--color-text-primary)] outline-none"
            >
              <option value="">全服最近记录</option>
              {players.map((player) => (
                <option key={player.id} value={player.id}>
                  {player.nickname} · {player.faction} · {player.id}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={loadOverview}
              disabled={loading}
              className="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border border-[var(--color-accent-border)] bg-[var(--color-accent-light)] px-4 text-sm font-semibold text-[var(--color-accent)] disabled:opacity-60"
            >
              <RefreshCw size={15} className={loading ? 'animate-spin' : ''} />
              刷新
            </button>
            <button
              type="button"
              onClick={handleSettleSeason}
              disabled={settlingSeason || overview?.season?.status === 'settled'}
              className="inline-flex min-h-10 items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 text-sm font-semibold text-[var(--color-text-secondary)] disabled:opacity-60"
            >
              {settlingSeason ? '结算中' : '结算赛季'}
            </button>
          </div>
        </div>
        {error && <div className="mt-3 rounded-xl border border-red-500/30 bg-red-500/8 px-3 py-2 text-sm text-red-600">{error}</div>}
      </div>

      {seasons && (
        <DataPanel title="赛季">
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {seasons.seasons.slice(0, 6).map((season) => (
              <div key={season.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
                <div className="flex items-center justify-between gap-2">
                  <p className="truncate text-sm font-bold text-[var(--color-text-primary)]">{season.name}</p>
                  <span className="shrink-0 text-xs font-semibold text-[var(--color-accent)]">{season.status}</span>
                </div>
                <p className="mt-1 text-xs text-[var(--color-text-muted)]">{formatShortTime(season.startsAt)} - {formatShortTime(season.endsAt)}</p>
                {season.settledAt && <p className="mt-1 text-xs text-[var(--color-text-secondary)]">结算 {formatShortTime(season.settledAt)}</p>}
              </div>
            ))}
          </div>
        </DataPanel>
      )}

      <div className="grid gap-3 md:grid-cols-4">
        <MetricCard label="赛季" value={overview?.season?.name ?? '-'} sub={overview?.season?.status ?? ''} />
        <MetricCard label="积分" value={playerState ? playerState.seasonPoints.toLocaleString() : '-'} sub={playerState ? `评分 ${playerState.rating}` : '选择玩家查看'} />
        <MetricCard label="行军" value={marches.length.toLocaleString()} sub="最近记录" />
        <MetricCard label="战斗" value={battles.length.toLocaleString()} sub="最近记录" />
      </div>

      {playerState && (
        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-sm">
          <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <h3 className="text-sm font-bold text-[var(--color-text-primary)]">玩家状态</h3>
            <div className="flex flex-wrap gap-1.5">
              <ProtectionButton label="免战 8h" disabled={protecting} onClick={() => handleSetProtection('manual', 8)} />
              <ProtectionButton label="系统 24h" disabled={protecting} onClick={() => handleSetProtection('system', 24)} />
              <ProtectionButton label="维护 2h" disabled={protecting} onClick={() => handleSetProtection('maintenance', 2)} />
            </div>
          </div>
          <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-4">
            <InfoItem label="状态" value={playerState.state.status || '-'} />
            <InfoItem label="保护类型" value={playerState.state.protectionType || '-'} />
            <InfoItem label="保护到" value={formatShortTime(playerState.state.protectedUntil)} />
            <InfoItem label="冷却到" value={formatShortTime(playerState.state.cooldownUntil)} />
            <InfoItem label="次数" value={`${playerState.state.dailyAttackCount}/${playerState.state.dailyAttackLimit}`} />
            <InfoItem label="攻胜" value={playerState.attackWins.toLocaleString()} />
            <InfoItem label="守胜" value={playerState.defenseWins.toLocaleString()} />
            <InfoItem label="失败" value={playerState.losses.toLocaleString()} />
            <InfoItem label="复仇" value={playerState.revengeRecords.length.toLocaleString()} />
          </div>
        </div>
      )}

      <RankingTable items={rankings} />
      <MarchTable
        items={marches}
        repairingMarchId={repairingMarchId}
        onCancel={handleCancelMarch}
        onForceResolve={handleForceResolve}
      />
      <BattleTable items={battles} />
    </section>
  )
}

// MetricCard 展示一个简洁指标块。
function MetricCard({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-sm">
      <p className="text-xs text-[var(--color-text-muted)]">{label}</p>
      <p className="mt-1 truncate text-xl font-black text-[var(--color-text-primary)]">{value}</p>
      <p className="mt-1 truncate text-xs text-[var(--color-text-secondary)]">{sub}</p>
    </div>
  )
}

// InfoItem 展示玩家 PVP 状态字段。
function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl bg-[var(--color-surface-dim)] px-3 py-2">
      <p className="text-[11px] text-[var(--color-text-muted)]">{label}</p>
      <p className="mt-0.5 truncate font-semibold text-[var(--color-text-primary)]">{value}</p>
    </div>
  )
}

// ProtectionButton 展示 GM PVP 保护操作按钮。
function ProtectionButton({ label, disabled, onClick }: { label: string; disabled: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2.5 py-1.5 text-xs font-semibold text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] hover:text-[var(--color-accent)] disabled:opacity-60"
    >
      {label}
    </button>
  )
}

// RankingTable 展示 PVP 排行榜。
function RankingTable({ items }: { items: PvpRankingEntry[] }) {
  return (
    <DataPanel title="排行榜">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="text-xs text-[var(--color-text-muted)]">
            <tr>
              <th className="py-2 pr-3">名次</th>
              <th className="py-2 pr-3">玩家</th>
              <th className="py-2 pr-3">阵营</th>
              <th className="py-2 pr-3">积分</th>
              <th className="py-2 pr-3">攻胜</th>
              <th className="py-2 pr-3">守胜</th>
              <th className="py-2 pr-3">失败</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.playerId} className="border-t border-[var(--color-border)]">
                <td className="py-2 pr-3 font-bold text-[var(--color-accent)]">{item.rank}</td>
                <td className="py-2 pr-3 text-[var(--color-text-primary)]">{item.nickname}</td>
                <td className="py-2 pr-3 text-[var(--color-text-secondary)]">{item.faction}</td>
                <td className="py-2 pr-3 font-semibold">{item.points}</td>
                <td className="py-2 pr-3">{item.attackWins}</td>
                <td className="py-2 pr-3">{item.defenseWins}</td>
                <td className="py-2 pr-3">{item.losses}</td>
              </tr>
            ))}
            {items.length === 0 && <EmptyRow colSpan={7} />}
          </tbody>
        </table>
      </div>
    </DataPanel>
  )
}

// MarchTable 展示 PVP 行军列表。
function MarchTable({
  items,
  repairingMarchId,
  onCancel,
  onForceResolve,
}: {
  items: PvpMarch[]
  repairingMarchId: string
  onCancel: (marchId: string) => void
  onForceResolve: (marchId: string) => void
}) {
  return (
    <DataPanel title="行军">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[960px] text-left text-sm">
          <thead className="text-xs text-[var(--color-text-muted)]">
            <tr>
              <th className="py-2 pr-3">状态</th>
              <th className="py-2 pr-3">模式</th>
              <th className="py-2 pr-3">攻击方</th>
              <th className="py-2 pr-3">防守方</th>
              <th className="py-2 pr-3">兵力</th>
              <th className="py-2 pr-3">到达</th>
              <th className="py-2 pr-3">修复</th>
              <th className="py-2 pr-3">ID</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const canForceResolve = item.status === 'marching' || item.status === 'resolving'
              const canCancel = !item.battleId && ['marching', 'returning', 'resolving'].includes(item.status)
              const busy = repairingMarchId === item.id
              return (
                <tr key={item.id} className="border-t border-[var(--color-border)]">
                  <td className="py-2 pr-3 font-semibold text-[var(--color-accent)]">{item.status}</td>
                  <td className="py-2 pr-3">{item.marchType}</td>
                  <td className="py-2 pr-3">{item.attackerName || item.attackerPlayerId}</td>
                  <td className="py-2 pr-3">{item.defenderName || item.defenderPlayerId}</td>
                  <td className="py-2 pr-3">{sumValues(item.attackTroops).toLocaleString()}</td>
                  <td className="py-2 pr-3">{formatShortTime(item.arrivesAt)}</td>
                  <td className="py-2 pr-3">
                    <div className="flex flex-wrap gap-1">
                      {canForceResolve && (
                        <RepairButton label="结算" disabled={busy} onClick={() => onForceResolve(item.id)} />
                      )}
                      {canCancel && (
                        <RepairButton label="取消" disabled={busy} onClick={() => onCancel(item.id)} />
                      )}
                      {!canForceResolve && !canCancel && <span className="text-xs text-[var(--color-text-muted)]">-</span>}
                    </div>
                  </td>
                  <td className="py-2 pr-3 font-mono text-xs text-[var(--color-text-muted)]">{item.id}</td>
                </tr>
              )
            })}
            {items.length === 0 && <EmptyRow colSpan={8} />}
          </tbody>
        </table>
      </div>
    </DataPanel>
  )
}

// RepairButton 展示 GM 行军修复操作按钮。
function RepairButton({ label, disabled, onClick }: { label: string; disabled: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="rounded-md border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 py-1 text-[11px] font-semibold text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] hover:text-[var(--color-accent)] disabled:opacity-60"
    >
      {disabled ? '处理中' : label}
    </button>
  )
}

// BattleTable 展示 PVP 战斗列表。
function BattleTable({ items }: { items: PvpBattle[] }) {
  return (
    <DataPanel title="战斗">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[820px] text-left text-sm">
          <thead className="text-xs text-[var(--color-text-muted)]">
            <tr>
              <th className="py-2 pr-3">状态</th>
              <th className="py-2 pr-3">胜者</th>
              <th className="py-2 pr-3">攻击方</th>
              <th className="py-2 pr-3">防守方</th>
              <th className="py-2 pr-3">掠夺</th>
              <th className="py-2 pr-3">结算</th>
              <th className="py-2 pr-3">ID</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-t border-[var(--color-border)]">
                <td className="py-2 pr-3 font-semibold text-[var(--color-accent)]">{item.status}</td>
                <td className="py-2 pr-3">{String(item.result?.winner ?? '-')}</td>
                <td className="py-2 pr-3">{item.attackerPlayerId}</td>
                <td className="py-2 pr-3">{item.defenderPlayerId}</td>
                <td className="py-2 pr-3">{sumValues(item.plunder).toLocaleString()}</td>
                <td className="py-2 pr-3">{formatShortTime(item.resolvedAt || item.updatedAt)}</td>
                <td className="py-2 pr-3 font-mono text-xs text-[var(--color-text-muted)]">{item.id}</td>
              </tr>
            ))}
            {items.length === 0 && <EmptyRow colSpan={7} />}
          </tbody>
        </table>
      </div>
    </DataPanel>
  )
}

// DataPanel 包装后台 PVP 数据表。
function DataPanel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-sm">
      <h3 className="mb-3 text-sm font-bold text-[var(--color-text-primary)]">{title}</h3>
      {children}
    </div>
  )
}

// EmptyRow 展示空数据占位行。
function EmptyRow({ colSpan }: { colSpan: number }) {
  return (
    <tr>
      <td colSpan={colSpan} className="py-6 text-center text-sm text-[var(--color-text-muted)]">
        暂无记录
      </td>
    </tr>
  )
}

// flattenPlayers 从账号列表中摊平玩家选项。
function flattenPlayers(accounts: AccountSummary[]): PlayerSummary[] {
  return accounts.flatMap((account) => account.players ?? [])
}

// sumValues 汇总数字对象中的正数值。
function sumValues(values?: Record<string, number>) {
  return Object.values(values ?? {}).reduce((sum, value) => sum + Math.max(0, value), 0)
}

// formatShortTime 格式化后台表格时间。
function formatShortTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
