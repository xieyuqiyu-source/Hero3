/**
 * 路由守卫：没有活跃玩家存档时跳转到登录页
 */
import { type FC } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { useGameStore } from '@/store/gameStore'

const RequirePlayer: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)

  if (!activePlayerId) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}

export default RequirePlayer
