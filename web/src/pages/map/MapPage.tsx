import { type FC } from 'react'
import { useGameStore } from '@/store/gameStore'

const MapPage: FC = () => {
  const targets = useGameStore((store) => store.state?.mapTargets ?? [])

  return (
    <div className="page-placeholder">
      <h2>地图</h2>
      <p>可攻击目标：{targets.length} 个</p>
      {targets[0] && <p>最近目标：{targets[0].type} Lv.{targets[0].level}</p>}
    </div>
  )
}

export default MapPage
