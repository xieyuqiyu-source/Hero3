import type { GameState } from '@/types'

interface PlayerStatePanelProps {
  gameState: GameState | null
}

export default function PlayerStatePanel({ gameState }: PlayerStatePanelProps) {
  const resources = gameState?.resources
  const totalArmy = gameState?.army.reduce((sum, unit) => sum + unit.amount, 0) ?? 0

  return (
    <div className="table-like">
      <div className="table-row">
        <span>{gameState?.player.nickname ?? '--'}</span>
        <strong>兵力 {totalArmy.toLocaleString()}</strong>
        <small>{gameState?.player.faction ?? '未同步'}</small>
      </div>
      <div className="resource-readout">
        <span>木 {resources?.wood.toLocaleString() ?? '--'}</span>
        <span>石 {resources?.stone.toLocaleString() ?? '--'}</span>
        <span>铁 {resources?.iron.toLocaleString() ?? '--'}</span>
        <span>粮 {resources?.food.toLocaleString() ?? '--'}</span>
      </div>
      <div className="state-summary">
        <span>建筑 {gameState?.buildings.length ?? 0}</span>
        <span>地图目标 {gameState?.mapTargets.length ?? 0}</span>
        <span>战报 {gameState?.recentBattleReports.length ?? 0}</span>
        <span>未读信函 {gameState?.unreadMessageCount ?? 0}</span>
      </div>
    </div>
  )
}
