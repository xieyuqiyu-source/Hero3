import { type FC } from 'react'
import { useGameStore } from '@/store/gameStore'

const ReportsPage: FC = () => {
  const reports = useGameStore((store) => store.state?.recentBattleReports ?? [])

  return (
    <div className="page-placeholder">
      <h2>战报</h2>
      <p>最近战报：{reports.length} 条</p>
    </div>
  )
}

export default ReportsPage
