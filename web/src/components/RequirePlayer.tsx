/**
 * 路由守卫：未登录账号或没有活跃玩家存档时跳转到登录/阵营选择页
 */
import { type FC } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { useGameStore } from '@/store/gameStore'
import { useAccountStore } from '@/store/accountStore'

const RequirePlayer: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const account = useAccountStore((s) => s.account)

  // 未登录账号 → 回登录页（选择阵营之前先登录）
  // 已登录但还没活跃玩家 → 同样回登录页（去选阵营创建存档）
  if (!account || !activePlayerId) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}

export default RequirePlayer
