import { create } from 'zustand'

export type ToastType = 'success' | 'error' | 'info'

export interface ToastItemData {
  id: number
  type: ToastType
  message: string
}

interface ToastStore {
  items: ToastItemData[]
  add: (type: ToastType, message: string) => void
  remove: (id: number) => void
}

let nextId = 0
const DUPLICATE_WINDOW_MS = 1200
const MAX_TOASTS = 3

export const useToastStore = create<ToastStore>((set) => ({
  items: [],
  add: (type, message) => {
    const id = nextId++
    const now = Date.now()
    set((state) => {
      const duplicate = state.items.some((toast) => (
        toast.type === type &&
        toast.message === message &&
        now - toast.id <= DUPLICATE_WINDOW_MS
      ))
      if (duplicate) return state
      return { items: [...state.items, { id: now + id, type, message }].slice(-MAX_TOASTS) }
    })
    setTimeout(() => {
      set((state) => ({ items: state.items.filter((toast) => toast.id !== now + id) }))
    }, 3500)
  },
  remove: (id) => set((state) => ({ items: state.items.filter((toast) => toast.id !== id) })),
}))

export const toast = {
  success: (message: string) => useToastStore.getState().add('success', message),
  error: (message: string) => useToastStore.getState().add('error', message),
  info: (message: string) => useToastStore.getState().add('info', message),
}
