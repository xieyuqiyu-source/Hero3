import { useState } from 'react'
import { Gamepad2, Fish, Dice5 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import PlayerSelector from './PlayerSelector'

interface MiniGameRecord {
  id: string
  playerId: string
  gameType: string
  resultName: string
  rarity: string
  rewardUnit: string
  rewardAmount: number
  createdAt: string
}

interface MiniGameSummary {
  totalRecords: number
  records: MiniGameRecord[]
  rewardTotals: Record<string, number>
}

const RARITY_STYLES: Record<string, string> = {
  common: 'text-slate-500 bg-slate-500/10',
  rare: 'text-blue-500 bg-blue-500/10',
  epic: 'text-purple-500 bg-purple-500/10',
  legendary: 'text-amber-500 bg-amber-500/10',
}

const RARITY_LABELS: Record<string, string> = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说',
}

export default function MiniGameRecordsPanel() {
  const [playerId, setPlayerId] = useState('')
  const [summary, setSummary] = useState<MiniGameSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')

  const handleQuery = async () => {
    if (!playerId) return
    setLoading(true)
    setMessage('')
    try {
      const result = await adminApi.getMiniGameRecords(playerId)
      setSummary(result)
    } catch (e: unknown) {
      setMessage(`❌ 查询失败: ${e instanceof Error ? e.message : '未知错误'}`)
      setSummary(null)
    } finally {
      setLoading(false)
    }
  }

  const fishingRecords = summary?.records?.filter(r => r.gameType === 'fishing') ?? []
  const gamblingRecords = summary?.records?.filter(r => r.gameType === 'gambling') ?? []

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-4">
        <Gamepad2 size={16} className="text-cyan-500" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">万象幻境记录</h2>
      </div>

      {/* 玩家选择 */}
      <div className="flex gap-2 mb-4">
        <div className="flex-1">
          <PlayerSelector
            value={playerId}
            onChange={(pid) => { setPlayerId(pid); setSummary(null) }}
            placeholder="选择玩家存档"
          />
        </div>
        <button
          type="button"
          onClick={handleQuery}
          disabled={!playerId || loading}
          className="px-3 py-2 rounded-xl text-xs font-medium bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer disabled:opacity-50"
        >
          {loading ? '查询中...' : '查询'}
        </button>
      </div>

      {message && (
        <div className="mb-3 px-3 py-2 rounded-lg bg-[var(--color-surface-dim)] text-xs text-[var(--color-text-primary)]">
          {message}
        </div>
      )}

      {summary && (
        <div className="space-y-4">
          {/* 汇总 */}
          {Object.keys(summary.rewardTotals).length > 0 && (
            <div className="p-3 rounded-xl bg-amber-500/5 border border-amber-500/20">
              <h3 className="text-xs font-semibold text-amber-600 mb-2">可兑换兵力汇总</h3>
              <div className="grid grid-cols-2 gap-1.5">
                {Object.entries(summary.rewardTotals).map(([unit, amount]) => (
                  <div key={unit} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
                    <span className="text-[10px] text-[var(--color-text-secondary)]">{unit}</span>
                    <span className="text-xs font-bold text-amber-600">{amount.toLocaleString()}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 钓鱼记录 */}
          {fishingRecords.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 mb-2">
                <Fish size={12} className="text-blue-500" />
                <h3 className="text-xs font-semibold text-[var(--color-text-primary)]">钓鱼记录 ({fishingRecords.length})</h3>
              </div>
              <div className="space-y-1 max-h-[200px] overflow-y-auto">
                {fishingRecords.map(record => (
                  <div key={record.id} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                    <div className="flex items-center gap-2">
                      <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded ${RARITY_STYLES[record.rarity] ?? ''}`}>
                        {RARITY_LABELS[record.rarity] ?? record.rarity}
                      </span>
                      <span className="text-xs text-[var(--color-text-primary)]">{record.resultName}</span>
                    </div>
                    <div className="text-right">
                      <span className="text-[10px] text-[var(--color-text-muted)]">{record.rewardUnit} ×{record.rewardAmount.toLocaleString()}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 赌博记录 */}
          {gamblingRecords.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 mb-2">
                <Dice5 size={12} className="text-purple-500" />
                <h3 className="text-xs font-semibold text-[var(--color-text-primary)]">豪赌记录 ({gamblingRecords.length})</h3>
              </div>
              <div className="space-y-1 max-h-[200px] overflow-y-auto">
                {gamblingRecords.map(record => (
                  <div key={record.id} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                    <div className="flex items-center gap-2">
                      <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded ${RARITY_STYLES[record.rarity] ?? ''}`}>
                        {RARITY_LABELS[record.rarity] ?? record.rarity}
                      </span>
                      <span className="text-xs text-[var(--color-text-primary)]">{record.resultName}</span>
                    </div>
                    <div className="text-right">
                      {record.rewardAmount > 0 ? (
                        <span className="text-[10px] text-green-600 font-medium">{record.rewardUnit} ×{record.rewardAmount.toLocaleString()}</span>
                      ) : (
                        <span className="text-[10px] text-red-500">输</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {summary.totalRecords === 0 && (
            <p className="text-xs text-[var(--color-text-muted)] text-center py-4">该玩家暂无小游戏记录</p>
          )}
        </div>
      )}
    </div>
  )
}
