import { type FC } from 'react'
import { useGameStore } from '@/store/gameStore'

const MilitaryPage: FC = () => {
  const army = useGameStore((store) => store.state?.army ?? [])
  const recruitQueues = useGameStore((store) => store.state?.recruitQueues ?? [])

  return (
    <div className="page-placeholder">
      <h2>军事</h2>
      <p>当前兵力：{army.reduce((sum, unit) => sum + unit.amount, 0).toLocaleString()}</p>
      <p>征兵队列：{recruitQueues.length} 个</p>
    </div>
  )
}

export default MilitaryPage
