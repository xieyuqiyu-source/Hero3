/* 公告状态仓库，独立加载公告列表和未读数量。 */

import { create } from 'zustand'
import { gameApi } from '@/api/game'
import type { Announcement } from '@/types/game'

interface AnnouncementStore {
  announcements: Announcement[]
  unread: number
  loading: boolean
  error: string
  loadedPlayerId: string
  loadAnnouncements: (playerId: string) => Promise<void>
  markRead: (playerId: string, announcementId: string) => Promise<Announcement | null>
  clear: () => void
}

export const useAnnouncementStore = create<AnnouncementStore>((set, get) => ({
  announcements: [],
  unread: 0,
  loading: false,
  error: '',
  loadedPlayerId: '',

  // loadAnnouncements 拉取当前玩家公告，不依赖完整游戏状态接口。
  loadAnnouncements: async (playerId: string) => {
    if (!playerId) {
      get().clear()
      return
    }
    set({ loading: true, error: '' })
    try {
      const result = await gameApi.listAnnouncements(playerId)
      set({
        announcements: result.announcements,
        unread: result.unread,
        loadedPlayerId: playerId,
        loading: false,
        error: '',
      })
    } catch (error) {
      set({
        announcements: [],
        unread: 0,
        loadedPlayerId: playerId,
        loading: false,
        error: error instanceof Error ? error.message : '公告加载失败',
      })
    }
  },

  // markRead 标记公告已读，并同步本地列表和未读数。
  markRead: async (playerId: string, announcementId: string) => {
    if (!playerId || !announcementId) return null
    const announcement = await gameApi.markAnnouncementRead(playerId, announcementId)
    set((state) => {
      const previous = state.announcements.find((item) => item.id === announcementId)
      const becameRead = previous && !previous.read
      return {
        announcements: state.announcements.map((item) => (
          item.id === announcement.id ? { ...item, ...announcement, read: true } : item
        )),
        unread: becameRead ? Math.max(0, state.unread - 1) : state.unread,
      }
    })
    return announcement
  },

  // clear 清空公告缓存。
  clear: () => set({ announcements: [], unread: 0, loading: false, error: '', loadedPlayerId: '' }),
}))
