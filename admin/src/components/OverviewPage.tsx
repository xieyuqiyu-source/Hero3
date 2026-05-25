import { useMemo, useState, type ReactNode } from 'react'
import {
  Users,
  Search,
  RefreshCw,
  Shield,
  Swords,
  TrendingUp,
  Server,
  Eye,
  Trash2,
  X,
} from 'lucide-react'
import type { AccountSummary, PlayerSummary } from '@/types'
import PlayerDrawer from '@/components/PlayerDrawer'

interface OverviewPageProps {
  accounts: AccountSummary[]
  busyTarget: string | null
  dashboardStats: Array<{ label: string; value: string; hint: string }>
  health: { status: string; version: string; environment: string; time: string } | null
  loading: boolean
  onDeletePlayer: (player: PlayerSummary) => Promise<void>
  onReload: () => void
}

interface PlayerRow {
  accountId: string
  accountUsername: string
  player: PlayerSummary
}

export default function OverviewPage({
  accounts,
  busyTarget,
  dashboardStats,
  health,
  loading,
  onDeletePlayer,
  onReload,
}: OverviewPageProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedPlayerId, setSelectedPlayerId] = useState<string | null>(null)
  const [factionFilter, setFactionFilter] = useState('all')

  const allPlayers = useMemo<PlayerRow[]>(() => {
    return accounts.flatMap((account) =>
      account.players.map((player) => ({
        accountId: account.id,
        accountUsername: account.username,
        player,
      })),
    )
  }, [accounts])

  const factions = useMemo(() => {
    return Array.from(new Set(allPlayers.map((row) => row.player.faction))).sort()
  }, [allPlayers])

  const filteredPlayers = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    return allPlayers.filter((row) => {
      const matchesSearch =
        q === '' ||
        row.player.nickname.toLowerCase().includes(q) ||
        row.accountUsername.toLowerCase().includes(q) ||
        row.player.id.toLowerCase().includes(q)
      const matchesFaction = factionFilter === 'all' || row.player.faction === factionFilter
      return matchesSearch && matchesFaction
    })
  }, [allPlayers, searchQuery, factionFilter])

  const enhancedStats = useMemo(() => {
    const totalAccounts = accounts.length
    const totalPlayers = allPlayers.length
    const factionCounts = allPlayers.reduce<Record<string, number>>((acc, row) => {
      acc[row.player.faction] = (acc[row.player.faction] ?? 0) + 1
      return acc
    }, {})
    const topFaction = Object.entries(factionCounts).sort((a, b) => b[1] - a[1])[0]

    return {
      totalAccounts,
      totalPlayers,
      topFaction: topFaction ? `${topFaction[0]} (${topFaction[1]})` : '--',
      serverStatus: health?.status === 'ok' ? '在线' : '离线',
      version: health?.version ?? '--',
    }
  }, [accounts, allPlayers, health])

  const totalArmyStat = dashboardStats[2]
  const resourceCapStat = dashboardStats[3]

  return (
    <div className="grid gap-5">
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-3">
        <StatCard
          icon={<Server size={16} />}
          label="服务器"
          value={enhancedStats.serverStatus}
          hint={enhancedStats.version}
          accent={health?.status === 'ok' ? 'emerald' : 'red'}
        />
        <StatCard
          icon={<Users size={16} />}
          label="账号"
          value={String(enhancedStats.totalAccounts)}
          hint={`${enhancedStats.totalPlayers} 个存档`}
          accent="indigo"
        />
        <StatCard
          icon={<Swords size={16} />}
          label="总兵力"
          value={totalArmyStat?.value ?? '--'}
          hint={totalArmyStat?.hint ?? ''}
          accent="amber"
        />
        <StatCard
          icon={<Shield size={16} />}
          label="最大阵营"
          value={enhancedStats.topFaction}
          hint="按当前存档统计"
          accent="purple"
        />
        <StatCard
          icon={<TrendingUp size={16} />}
          label="资源上限"
          value={resourceCapStat?.value ?? '--'}
          hint={resourceCapStat?.hint ?? ''}
          accent="sky"
        />
      </div>

      <div className="flex items-center gap-2 flex-wrap">
        <button
          type="button"
          onClick={onReload}
          disabled={loading}
          className="inline-flex items-center gap-1.5 px-3 py-2 rounded-xl text-xs font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
          刷新数据
        </button>
        <span className="text-[11px] text-[var(--color-text-muted)] ml-1">
          {health?.time ? `最后同步 ${new Date(health.time).toLocaleTimeString()}` : ''}
        </span>
      </div>

      <div className="flex gap-4 max-lg:flex-col">
        <div className="flex-1 min-w-0 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)]">
            <div className="flex items-center gap-2">
              <Users size={16} className="text-[var(--color-accent)]" />
              <h2 className="text-sm font-bold text-[var(--color-text-primary)]">玩家管理</h2>
              <span className="px-2 py-0.5 rounded-full text-[10px] font-bold text-amber-700 bg-[var(--color-gold-soft)]">
                {filteredPlayers.length} / {allPlayers.length}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-2 px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <div className="relative flex-1 max-w-[280px]">
              <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)]" />
              <input
                type="text"
                placeholder="搜索昵称、账号、ID..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-8 pr-8 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-accent-border)]"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery('')}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer"
                >
                  <X size={12} />
                </button>
              )}
            </div>
            <select
              value={factionFilter}
              onChange={(e) => setFactionFilter(e.target.value)}
              className="px-2.5 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-accent-border)] cursor-pointer"
            >
              <option value="all">全部阵营</option>
              {factions.map((faction) => (
                <option key={faction} value={faction}>
                  {faction}
                </option>
              ))}
            </select>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                  <th className="text-left px-4 py-2.5 font-bold text-[var(--color-text-muted)] uppercase tracking-wider text-[10px]">玩家</th>
                  <th className="text-left px-3 py-2.5 font-bold text-[var(--color-text-muted)] uppercase tracking-wider text-[10px]">阵营</th>
                  <th className="text-left px-3 py-2.5 font-bold text-[var(--color-text-muted)] uppercase tracking-wider text-[10px]">账号</th>
                  <th className="text-left px-3 py-2.5 font-bold text-[var(--color-text-muted)] uppercase tracking-wider text-[10px]">更新时间</th>
                  <th className="text-right px-4 py-2.5 font-bold text-[var(--color-text-muted)] uppercase tracking-wider text-[10px]">操作</th>
                </tr>
              </thead>
              <tbody>
                {filteredPlayers.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-12 text-center text-sm text-[var(--color-text-muted)]">
                      {searchQuery || factionFilter !== 'all' ? '没有匹配的玩家' : '暂无玩家数据'}
                    </td>
                  </tr>
                ) : (
                  filteredPlayers.map((row) => (
                    <tr
                      key={row.player.id}
                      className={`border-b border-[var(--color-border)] hover:bg-[var(--color-accent-light)] transition-colors cursor-pointer ${selectedPlayerId === row.player.id ? 'bg-[var(--color-accent-light)]' : ''}`}
                      onClick={() => setSelectedPlayerId(row.player.id)}
                    >
                      <td className="px-4 py-2.5">
                        <div className="flex items-center gap-2">
                          <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white text-[10px] font-bold shrink-0">
                            {row.player.nickname.charAt(0)}
                          </div>
                          <div>
                            <strong className="text-[var(--color-text-primary)] font-bold block">{row.player.nickname}</strong>
                            <span className="text-[10px] text-[var(--color-text-muted)]">{row.player.id.slice(0, 8)}...</span>
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-2.5">
                        <FactionBadge faction={row.player.faction} />
                      </td>
                      <td className="px-3 py-2.5 text-[var(--color-text-secondary)]">{row.accountUsername}</td>
                      <td className="px-3 py-2.5 text-[var(--color-text-muted)]">
                        {row.player.updatedAt ? formatRelativeTime(row.player.updatedAt) : '--'}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation()
                              setSelectedPlayerId(row.player.id)
                            }}
                            className="p-1.5 rounded-lg text-[var(--color-accent)] bg-[var(--color-accent-light)] border border-[var(--color-accent-border)] hover:bg-[var(--color-accent)]/15 cursor-pointer transition-colors"
                            title="查看详情"
                          >
                            <Eye size={12} />
                          </button>
                          <button
                            type="button"
                            disabled={busyTarget !== null}
                            onClick={(e) => {
                              e.stopPropagation()
                              void onDeletePlayer(row.player)
                            }}
                            className="p-1.5 rounded-lg text-red-600 bg-red-500/8 border border-red-500/20 hover:bg-red-500/15 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            title="删除存档"
                          >
                            <Trash2 size={12} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <div className="flex items-center justify-between px-4 py-2.5 border-t border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[11px] text-[var(--color-text-muted)]">
              共 {accounts.length} 个账号，{allPlayers.length} 个存档
            </span>
            <span className="text-[11px] text-[var(--color-text-muted)]">点击行查看详情</span>
          </div>
        </div>

        {selectedPlayerId && (
          <PlayerDrawer playerId={selectedPlayerId} onClose={() => setSelectedPlayerId(null)} />
        )}
      </div>
    </div>
  )
}

function StatCard({
  icon,
  label,
  value,
  hint,
  accent,
}: {
  icon: ReactNode
  label: string
  value: string
  hint: string
  accent: string
}) {
  const accentColors: Record<string, string> = {
    emerald: 'from-emerald-500 to-emerald-600',
    red: 'from-red-500 to-red-600',
    indigo: 'from-indigo-500 to-indigo-600',
    amber: 'from-amber-500 to-amber-600',
    purple: 'from-purple-500 to-purple-600',
    sky: 'from-sky-500 to-sky-600',
  }

  return (
    <div className="relative overflow-hidden px-3.5 py-3 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)]">
      <div className={`absolute top-0 left-0 w-full h-[3px] bg-gradient-to-r ${accentColors[accent] ?? accentColors.indigo}`} />
      <div className="flex items-center gap-2 mb-1.5">
        <span className="text-[var(--color-text-muted)]">{icon}</span>
        <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">{label}</span>
      </div>
      <strong className="block text-xl font-black text-[var(--color-text-primary)] leading-none">{value}</strong>
      <small className="text-[11px] text-[var(--color-text-muted)] mt-1 block">{hint}</small>
    </div>
  )
}

function FactionBadge({ faction }: { faction: string }) {
  const colors: Record<string, string> = {
    '精灵': 'bg-green-500/10 text-green-700 border-green-500/20',
    '人类': 'bg-blue-500/10 text-blue-700 border-blue-500/20',
    '兽人': 'bg-red-500/10 text-red-700 border-red-500/20',
  }
  const cls = colors[faction] ?? 'bg-gray-500/10 text-gray-700 border-gray-500/20'

  return (
    <span className={`inline-flex px-2 py-0.5 rounded-md text-[11px] font-bold border ${cls}`}>
      {faction}
    </span>
  )
}

function formatRelativeTime(dateStr: string): string {
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return '刚刚'
    if (diffMin < 60) return `${diffMin}分钟前`
    const diffHour = Math.floor(diffMin / 60)
    if (diffHour < 24) return `${diffHour}小时前`
    const diffDay = Math.floor(diffHour / 24)
    if (diffDay < 30) return `${diffDay}天前`
    return date.toLocaleDateString()
  } catch {
    return dateStr
  }
}
