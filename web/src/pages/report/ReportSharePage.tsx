import { useState, useEffect, type FC } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Skull } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useConfigStore } from '@/store/configStore'
import BattleReportDetail from '@/pages/news/components/BattleReportDetail'
import type { BattleReport } from '@/types/game'

const ReportSharePage: FC = () => {
  const { reportId } = useParams<{ reportId: string }>()
  const navigate = useNavigate()
  const [report, setReport] = useState<BattleReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const loadBootstrap = useConfigStore((s) => s.loadBootstrap)

  useEffect(() => {
    loadBootstrap()
  }, [loadBootstrap])

  useEffect(() => {
    if (!reportId) return
    setLoading(true)
    gameApi.getReport(reportId)
      .then(setReport)
      .catch(() => setError('战报不存在或已过期'))
      .finally(() => setLoading(false))
  }, [reportId])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)]">
        <div className="animate-pulse text-[var(--color-text-muted)] text-sm">加载战报中...</div>
      </div>
    )
  }

  if (error || !report) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-[var(--color-bg)] gap-4">
        <Skull size={48} className="text-red-400/50" />
        <p className="text-[var(--color-text-secondary)] text-sm">{error || '战报不存在'}</p>
        <button onClick={() => navigate('/')} className="text-xs text-[var(--color-accent)] hover:underline cursor-pointer">返回游戏</button>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[var(--color-bg)] px-4 sm:px-[16.67%] py-6">
      <div className="w-full" style={{ zoom: 1.4 }}>
        <BattleReportDetail report={report} onBack={() => navigate('/')} />
      </div>
    </div>
  )
}

export default ReportSharePage
