/* 本文件封装公告未读红点查询，供桌面和移动菜单复用。 */
import { useEffect, useState } from 'react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'

export function useAnnouncementUnread() {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [unread, setUnread] = useState(false)

  useEffect(() => {
    let cancelled = false
    const refreshUnread = () => {
      if (!activePlayerId) {
        setUnread(false)
        return
      }
      gameApi.listAnnouncements(activePlayerId, { page: 1, pageSize: 1 }).then((result) => {
        if (!cancelled) setUnread(result.unread)
      }).catch(() => {
        if (!cancelled) setUnread(false)
      })
    }
    if (!activePlayerId) {
      setUnread(false)
      return
    }
    refreshUnread()
    const timer = window.setInterval(refreshUnread, 30000)
    const handleFocus = () => refreshUnread()
    window.addEventListener('focus', handleFocus)
    return () => {
      cancelled = true
      window.clearInterval(timer)
      window.removeEventListener('focus', handleFocus)
    }
  }, [activePlayerId])

  return unread
}
