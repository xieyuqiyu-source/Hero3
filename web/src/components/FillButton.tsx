import { useState, type FC } from 'react'
import { Warehouse } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import { useConfirmPreferenceStore } from '@/store/confirmPreferenceStore'
import { toast } from '@/components/ui'
import ConfirmCityGoldModal from './ConfirmCityGoldModal'

const FillButton: FC = () => {
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)
  const resources = useGameStore((s) => s.state?.resources)
  const skipConfirmations = useConfirmPreferenceStore((s) => s.skipConfirmations)

  // 计算需要补充的总资源量和城金花费
  const totalNeeded = (() => {
    if (!resources) return 0
    let total = 0
    for (const res of Object.keys(resources.capacity ?? {})) {
      const current = resources.items?.[res] ?? 0
      const cap = resources.capacity?.[res] ?? 0
      if (current < cap) total += cap - current
    }
    return total
  })()

  const cost = Math.max(1, Math.ceil(totalNeeded / 3000))
  const isFull = totalNeeded === 0

  const handleConfirm = async () => {
    if (!activePlayerId || loading) return
    setLoading(true)
    try {
      const result = await gameApi.fillResourcesPaid(activePlayerId)
      setState(result.state)
      toast.success(`爆仓完成，消耗 ${result.cost} 城金`)
      setConfirmOpen(false)
    } catch (e: any) {
      const msg = e?.message || '操作失败'
      if (msg.includes('insufficient')) {
        toast.error('城金不足')
      } else {
        toast.error(msg)
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={() => {
          if (isFull) return
          if (skipConfirmations) handleConfirm()
          else setConfirmOpen(true)
        }}
        disabled={isFull}
        className={`
          flex items-center gap-1 px-2 py-1 rounded-lg
          text-[10px] font-bold cursor-pointer
          transition-all duration-200
          ${isFull
            ? 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] border border-[var(--color-border)] opacity-50 cursor-not-allowed'
            : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] border border-[var(--color-border)] hover:text-emerald-500 hover:border-emerald-500/40'
          }
        `}
        title={isFull ? '资源已满' : `一键爆仓（${cost} 城金）`}
      >
        <Warehouse size={11} />
        爆仓
      </button>

      <ConfirmCityGoldModal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title="一键爆仓"
        description="将所有资源填满到仓库上限"
        cost={cost}
        loading={loading}
      />
    </>
  )
}

export default FillButton
