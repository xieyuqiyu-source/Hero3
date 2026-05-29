import { useState } from 'react'
import { Sparkles, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'

/** 预设的加成属性选项 */
const BUFF_KEY_OPTIONS = [
  { value: 'productionBonus', label: '全资源产量' },
  { value: 'woodProductionBonus', label: '木材产量' },
  { value: 'stoneProductionBonus', label: '石料产量' },
  { value: 'ironProductionBonus', label: '铁矿产量' },
  { value: 'foodProductionBonus', label: '粮食产量' },
  { value: 'capacityBonus', label: '仓库容量' },
  { value: 'attackBonus', label: '攻击力' },
  { value: 'defenseBonus', label: '防御力' },
  { value: 'infantryDefenseBonus', label: '步兵防御' },
  { value: 'cavalryDefenseBonus', label: '骑兵防御' },
  { value: 'buildSpeedBonus', label: '建筑速度' },
  { value: 'recruitSpeedBonus', label: '征兵速度' },
  { value: 'marchSpeedBonus', label: '行军速度' },
]

const MODE_OPTIONS = [
  { value: 'percentAdd', label: '百分比加法 (+X%)' },
  { value: 'percentMultiply', label: '百分比乘法 (×N)' },
  { value: 'flat', label: '固定值 (+N)' },
]

interface Buff {
  id: string
  source: string
  key: string
  value: number
  mode: string
  expiresAt?: string
  createdAt: string
  note?: string
}

export default function BuffPanel() {
  const [playerId, setPlayerId] = useState('')
  const [key, setKey] = useState('productionBonus')
  const [value, setValue] = useState(0.5)
  const [mode, setMode] = useState('percentAdd')
  const [hours, setHours] = useState(24)
  const [permanent, setPermanent] = useState(false)
  const [note, setNote] = useState('')
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)
  const [playerBuffs, setPlayerBuffs] = useState<Buff[]>([])

  const showMsg = (msg: string) => {
    setMessage(msg)
    setTimeout(() => setMessage(''), 4000)
  }

  const handleGrant = async () => {
    if (!playerId) { showMsg('❌ 请输入玩家 ID'); return }
    const desc = `${BUFF_KEY_OPTIONS.find(o => o.value === key)?.label ?? key} ${mode === 'flat' ? `+${value}` : `+${Math.round(value * 100)}%`}${permanent ? '（永久）' : `（${hours}小时）`}`
    if (!confirm(`确认给玩家 ${playerId} 发放加成？\n\n${desc}\n备注：${note || '无'}`)) return

    setLoading(true)
    try {
      const result = await adminApi.grantBuff(playerId, key, value, mode, permanent ? 0 : hours, note)
      setPlayerBuffs(result.state.buffs ?? [])
      showMsg(`✅ 加成发放成功`)
    } catch (e: unknown) {
      showMsg(`❌ 失败: ${e instanceof Error ? e.message : '未知错误'}`)
    } finally {
      setLoading(false)
    }
  }

  const handleRevoke = async (buffId: string) => {
    if (!playerId) return
    if (!confirm('确认撤销该加成？')) return
    setLoading(true)
    try {
      const result = await adminApi.revokeBuff(playerId, buffId)
      setPlayerBuffs(result.state.buffs ?? [])
      showMsg('✅ 已撤销')
    } catch (e: unknown) {
      showMsg(`❌ 失败: ${e instanceof Error ? e.message : '未知错误'}`)
    } finally {
      setLoading(false)
    }
  }

  const handleLoadBuffs = async () => {
    if (!playerId) return
    setLoading(true)
    try {
      const state = await adminApi.getPlayerState(playerId)
      setPlayerBuffs(state.buffs ?? [])
    } catch (e: unknown) {
      showMsg(`❌ 加载失败: ${e instanceof Error ? e.message : '未知错误'}`)
    } finally {
      setLoading(false)
    }
  }

  const formatValue = (v: number, m: string) => {
    if (m === 'flat') return `+${v}`
    if (m === 'percentMultiply') return `×${v + 1}`
    return `+${Math.round(v * 100)}%`
  }

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-4">
        <Sparkles size={16} className="text-amber-500" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">加成管理</h2>
      </div>

      {/* 玩家 ID */}
      <div className="grid gap-2 mb-4">
        <div className="flex gap-2">
          <input
            type="text"
            value={playerId}
            onChange={(e) => setPlayerId(e.target.value)}
            placeholder="玩家 ID (player_xxx)"
            className="flex-1 px-3 py-2 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)] outline-none focus:border-[var(--color-accent-border)]"
          />
          <button
            type="button"
            onClick={handleLoadBuffs}
            disabled={!playerId || loading}
            className="px-3 py-2 rounded-xl text-xs font-medium bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer disabled:opacity-50"
          >
            查询
          </button>
        </div>
      </div>

      {/* 发放表单 */}
      <div className="grid gap-2 mb-4 p-3 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-[10px] text-[var(--color-text-muted)] mb-1 block">属性</label>
            <select
              value={key}
              onChange={(e) => setKey(e.target.value)}
              className="w-full px-2 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none"
            >
              {BUFF_KEY_OPTIONS.map(opt => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-[10px] text-[var(--color-text-muted)] mb-1 block">模式</label>
            <select
              value={mode}
              onChange={(e) => setMode(e.target.value)}
              className="w-full px-2 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none"
            >
              {MODE_OPTIONS.map(opt => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-[10px] text-[var(--color-text-muted)] mb-1 block">
              数值 {mode === 'flat' ? '(绝对值)' : '(小数，0.5=50%)'}
            </label>
            <input
              type="number"
              step="0.01"
              value={value}
              onChange={(e) => setValue(parseFloat(e.target.value) || 0)}
              className="w-full px-2 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none"
            />
          </div>
          <div>
            <label className="text-[10px] text-[var(--color-text-muted)] mb-1 block">时长（小时）</label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                value={hours}
                onChange={(e) => setHours(parseInt(e.target.value) || 0)}
                disabled={permanent}
                className="flex-1 px-2 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none disabled:opacity-50"
              />
              <label className="flex items-center gap-1 text-[10px] text-[var(--color-text-muted)] cursor-pointer whitespace-nowrap">
                <input type="checkbox" checked={permanent} onChange={(e) => setPermanent(e.target.checked)} className="w-3 h-3" />
                永久
              </label>
            </div>
          </div>
        </div>
        <div>
          <label className="text-[10px] text-[var(--color-text-muted)] mb-1 block">备注</label>
          <input
            type="text"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="如：端午节活动、停服补偿..."
            className="w-full px-2 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)] outline-none"
          />
        </div>
        <button
          type="button"
          onClick={handleGrant}
          disabled={loading || !playerId}
          className="mt-1 px-4 py-2.5 rounded-xl text-sm font-bold bg-amber-500 text-white hover:bg-amber-600 cursor-pointer transition-colors disabled:opacity-50"
        >
          发放加成
        </button>
      </div>

      {/* 当前生效的 buff 列表 */}
      {playerBuffs.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">当前生效加成</h3>
          <div className="space-y-1.5">
            {playerBuffs.map(buff => (
              <div key={buff.id} className="flex items-center justify-between px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                <div className="flex-1 min-w-0">
                  <span className="text-xs font-medium text-[var(--color-text-primary)]">
                    {BUFF_KEY_OPTIONS.find(o => o.value === buff.key)?.label ?? buff.key}
                  </span>
                  <span className="text-xs font-bold text-amber-500 ml-2">
                    {formatValue(buff.value, buff.mode)}
                  </span>
                  {buff.expiresAt && (
                    <span className="text-[10px] text-[var(--color-text-muted)] ml-2">
                      到期: {new Date(buff.expiresAt).toLocaleString('zh-CN')}
                    </span>
                  )}
                  {!buff.expiresAt && (
                    <span className="text-[10px] text-green-500 ml-2">永久</span>
                  )}
                  {buff.note && (
                    <span className="text-[10px] text-[var(--color-text-muted)] ml-2">({buff.note})</span>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => handleRevoke(buff.id)}
                  disabled={loading}
                  className="p-1.5 rounded-lg text-red-400 hover:text-red-600 hover:bg-red-500/10 cursor-pointer transition-colors"
                >
                  <Trash2 size={12} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Feedback */}
      {message && (
        <div className="mt-3 px-3 py-2 rounded-lg bg-[var(--color-surface-dim)] text-xs text-[var(--color-text-primary)]">
          {message}
        </div>
      )}
    </div>
  )
}
