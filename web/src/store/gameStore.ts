import { create } from 'zustand'
import type { GameState } from '@/types/game'

interface GameStore {
  /** 后端返回的权威游戏状态 */
  state: GameState | null
  /** 是否正在加载 */
  loading: boolean
  /** 错误信息 */
  error: string | null

  /** 设置完整游戏状态 */
  setState: (state: GameState) => void
  /** 部分更新（用于动作接口返回后局部刷新） */
  patchState: (patch: Partial<GameState>) => void
  /** 设置 loading */
  setLoading: (loading: boolean) => void
  /** 设置错误 */
  setError: (error: string | null) => void
}

export const useGameStore = create<GameStore>((set) => ({
  state: null,
  loading: false,
  error: null,

  setState: (state) => set({ state, error: null }),
  patchState: (patch) =>
    set((prev) => ({
      state: prev.state ? { ...prev.state, ...patch } : null,
    })),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error, loading: false }),
}))
