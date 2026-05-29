import { useState, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import { adminApi } from '@/api/admin'

interface PlayerOption {
  playerId: string
  accountId: string
  label: string // "昵称 (账号名)"
  nickname: string
  faction: string
  username: string
}

interface PlayerSelectorProps {
  /** 当前选中的 playerId */
  value: string
  /** 选中变化回调，同时返回 accountId */
  onChange: (playerId: string, accountId: string) => void
  /** placeholder 文字 */
  placeholder?: string
  /** 是否同时显示 accountId 选择 */
  showAccountId?: boolean
  /** 外部传入的 accountId 回调 */
  onAccountChange?: (accountId: string) => void
}

export default function PlayerSelector({ value, onChange, placeholder = '选择玩家存档', onAccountChange }: PlayerSelectorProps) {
  const [options, setOptions] = useState<PlayerOption[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')

  useEffect(() => {
    loadAccounts()
  }, [])

  const loadAccounts = async () => {
    setLoading(true)
    try {
      const result = await adminApi.getAccounts()
      const opts: PlayerOption[] = []
      for (const acc of result.accounts) {
        for (const player of acc.players) {
          opts.push({
            playerId: player.id,
            accountId: acc.id,
            label: `${player.nickname} (${acc.username})`,
            nickname: player.nickname,
            faction: player.faction,
            username: acc.username,
          })
        }
      }
      setOptions(opts)
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }

  const selected = options.find(o => o.playerId === value)
  const filtered = search
    ? options.filter(o => o.label.toLowerCase().includes(search.toLowerCase()) || o.playerId.includes(search))
    : options

  const handleSelect = (opt: PlayerOption) => {
    onChange(opt.playerId, opt.accountId)
    onAccountChange?.(opt.accountId)
    setOpen(false)
    setSearch('')
  }

  return (
    <div className="relative">
      {/* Trigger */}
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)] cursor-pointer hover:border-[var(--color-accent-border)] transition-colors"
      >
        <span className={selected ? '' : 'text-[var(--color-text-muted)]'}>
          {loading ? '加载中...' : selected ? selected.label : placeholder}
        </span>
        <ChevronDown size={12} className={`text-[var(--color-text-muted)] transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-lg overflow-hidden">
          {/* Search */}
          <div className="p-2 border-b border-[var(--color-border)]">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="搜索昵称/账号/ID..."
              autoFocus
              className="w-full px-2 py-1.5 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)] outline-none focus:border-[var(--color-accent-border)]"
            />
          </div>
          {/* Options */}
          <div className="max-h-[200px] overflow-y-auto">
            {filtered.length === 0 ? (
              <div className="px-3 py-2 text-xs text-[var(--color-text-muted)]">无匹配结果</div>
            ) : (
              filtered.map(opt => (
                <button
                  key={opt.playerId}
                  type="button"
                  onClick={() => handleSelect(opt)}
                  className={`w-full text-left px-3 py-2 text-xs hover:bg-[var(--color-accent-light)] cursor-pointer transition-colors ${opt.playerId === value ? 'bg-[var(--color-accent-light)] font-medium' : ''}`}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-[var(--color-text-primary)]">{opt.nickname}</span>
                    <span className="text-[10px] text-[var(--color-text-muted)]">{opt.username}</span>
                  </div>
                  <div className="text-[10px] text-[var(--color-text-muted)] mt-0.5">{opt.playerId}</div>
                </button>
              ))
            )}
          </div>
        </div>
      )}

      {/* Click outside to close */}
      {open && <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />}
    </div>
  )
}
