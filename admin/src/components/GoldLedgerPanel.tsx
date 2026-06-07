import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Coins, RefreshCw, Search, ArrowDownCircle, ArrowUpCircle } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { GoldLedgerEntry } from '@/types'

const CURRENCY_LABELS: Record<string, string> = {
  gold: '金币',
  cityGold: '城金',
}

const REF_TYPE_LABELS: Record<string, string> = {
  exchange: '兑换',
  admin_adjust: 'GM调整',
  instant_recruit: '极速征兵',
  instant_building: '极速建筑',
  boost_purchase: '购买加成',
  battle_overflow: '溢出转城金',
  resource_fill: '补满资源',
}

const controlClass = 'px-3 py-2 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]'

export default function GoldLedgerPanel() {
  const [entries, setEntries] = useState<GoldLedgerEntry[]>([])
  const [accountId, setAccountId] = useState('')
  const [playerId, setPlayerId] = useState('')
  const [currency, setCurrency] = useState('')
  const [refType, setRefType] = useState('')
  const [limit, setLimit] = useState(200)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const result = await adminApi.getGoldLedger({
        accountId: accountId.trim() || undefined,
        playerId: playerId.trim() || undefined,
        currency: currency === 'gold' || currency === 'cityGold' ? currency : undefined,
        refType: refType.trim() || undefined,
        limit,
      })
      setEntries(Array.isArray(result.entries) ? result.entries : [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const totals = useMemo(() => {
    return (Array.isArray(entries) ? entries : []).reduce<Record<string, { credit: number; debit: number }>>((acc, entry) => {
      const bucket = acc[entry.currency] ?? { credit: 0, debit: 0 }
      if (entry.direction === 'credit' || entry.direction === 'debit') {
        bucket[entry.direction] += entry.amount
      }
      acc[entry.currency] = bucket
      return acc
    }, {})
  }, [entries])

  return (
    <div className="grid gap-4">
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] overflow-hidden">
        <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-[var(--color-border)]">
          <div className="flex items-center gap-2">
            <Coins size={16} className="text-[var(--color-accent)]" />
            <h2 className="text-base font-bold text-[var(--color-text-primary)]">货币流水</h2>
            <span className="text-[11px] text-[var(--color-text-muted)]">{entries.length} 条</span>
          </div>
          <button
            type="button"
            onClick={() => void load()}
            disabled={loading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity disabled:opacity-50"
          >
            <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
            刷新
          </button>
        </div>

        <div className="grid gap-2 lg:grid-cols-[1fr_1fr_120px_150px_100px_auto] px-4 py-3 border-b border-[var(--color-border)] bg-[var(--color-surface-dim)]">
          <FilterInput value={accountId} onChange={setAccountId} placeholder="账号 ID" />
          <FilterInput value={playerId} onChange={setPlayerId} placeholder="玩家 ID" />
          <select value={currency} onChange={(e) => setCurrency(e.target.value)} className={controlClass}>
            <option value="">全部货币</option>
            <option value="gold">金币</option>
            <option value="cityGold">城金</option>
          </select>
          <select value={refType} onChange={(e) => setRefType(e.target.value)} className={controlClass}>
            <option value="">全部类型</option>
            {Object.entries(REF_TYPE_LABELS).map(([key, label]) => (
              <option key={key} value={key}>{label}</option>
            ))}
          </select>
          <input
            type="number"
            min={1}
            max={1000}
            value={limit}
            onChange={(e) => setLimit(Math.max(1, Math.min(1000, parseInt(e.target.value) || 200)))}
            className={controlClass}
          />
          <button
            type="button"
            onClick={() => void load()}
            disabled={loading}
            className="inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl text-xs font-bold border border-[var(--color-accent-border)] bg-[var(--color-accent-light)] text-[var(--color-accent)] cursor-pointer disabled:opacity-50"
          >
            <Search size={13} />
            查询
          </button>
        </div>

        {error && (
          <div className="px-4 py-3 text-sm font-medium text-red-600 border-b border-[var(--color-border)]">{error}</div>
        )}

        <div className="grid grid-cols-2 lg:grid-cols-4 gap-2 px-4 py-3 border-b border-[var(--color-border)]">
          <TotalBadge label="金币收入" value={totals.gold?.credit ?? 0} tone="credit" />
          <TotalBadge label="金币支出" value={totals.gold?.debit ?? 0} tone="debit" />
          <TotalBadge label="城金收入" value={totals.cityGold?.credit ?? 0} tone="credit" />
          <TotalBadge label="城金支出" value={totals.cityGold?.debit ?? 0} tone="debit" />
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                <Th>时间</Th>
                <Th>货币</Th>
                <Th>方向</Th>
                <Th>数量</Th>
                <Th>余额</Th>
                <Th>类型</Th>
                <Th>账号 / 玩家</Th>
                <Th>关联</Th>
                <Th>备注</Th>
              </tr>
            </thead>
            <tbody>
              {entries.length === 0 ? (
                <tr>
                  <td colSpan={9} className="px-4 py-12 text-center text-sm text-[var(--color-text-muted)]">
                    暂无流水
                  </td>
                </tr>
              ) : entries.map((entry) => (
                <LedgerRow key={entry.id} entry={entry} />
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}

function LedgerRow({ entry }: { entry: GoldLedgerEntry }) {
  const isCredit = entry.direction === 'credit'
  const isDebit = entry.direction === 'debit'
  const directionLabel = isCredit ? '收入' : isDebit ? '支出' : '未知'
  const directionTone = isCredit ? 'text-emerald-600' : isDebit ? 'text-red-600' : 'text-[var(--color-text-muted)]'
  const amountPrefix = isCredit ? '+' : isDebit ? '-' : ''

  return (
    <tr className="border-b border-[var(--color-border)] hover:bg-[var(--color-accent-light)]">
                  <Td>{formatTime(entry.createdAt)}</Td>
                  <Td>{CURRENCY_LABELS[entry.currency] ?? entry.currency}</Td>
                  <Td>
                    <span className={`inline-flex items-center gap-1 font-bold ${directionTone}`}>
                      {isCredit ? <ArrowUpCircle size={12} /> : isDebit ? <ArrowDownCircle size={12} /> : null}
                      {directionLabel}
                    </span>
                  </Td>
                  <Td>
                    <span className={`font-black ${directionTone}`}>
                      {amountPrefix}{entry.amount.toLocaleString()}
                    </span>
                  </Td>
                  <Td>{entry.balanceAfter.toLocaleString()}</Td>
                  <Td>{REF_TYPE_LABELS[entry.refType ?? ''] ?? entry.refType ?? '-'}</Td>
                  <Td>
                    <div className="grid gap-0.5">
                      <span className="font-mono text-[10px]">{entry.accountId || '-'}</span>
                      <span className="font-mono text-[10px] text-[var(--color-text-muted)]">{entry.playerId || '-'}</span>
                    </div>
                  </Td>
                  <Td className="font-mono text-[10px]">{entry.refId || '-'}</Td>
                  <Td className="max-w-[180px] truncate" title={entry.reason || ''}>{entry.reason || '-'}</Td>
    </tr>
  )
}

function FilterInput({ value, onChange, placeholder }: { value: string; onChange: (value: string) => void; placeholder: string }) {
  return (
    <input
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className={controlClass}
    />
  )
}

function TotalBadge({ label, value, tone }: { label: string; value: number; tone: 'credit' | 'debit' }) {
  return (
    <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
      <div className="text-[10px] text-[var(--color-text-muted)]">{label}</div>
      <div className={`text-sm font-black ${tone === 'credit' ? 'text-emerald-600' : 'text-red-600'}`}>{value.toLocaleString()}</div>
    </div>
  )
}

function Th({ children }: { children: ReactNode }) {
  return <th className="text-left px-3 py-2.5 font-bold text-[var(--color-text-muted)] uppercase tracking-wider text-[10px]">{children}</th>
}

function Td({ children, className = '', title }: { children: ReactNode; className?: string; title?: string }) {
  return <td title={title} className={`px-3 py-2.5 text-[var(--color-text-primary)] whitespace-nowrap ${className}`}>{children}</td>
}

function formatTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
