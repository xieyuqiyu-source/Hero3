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

export const useToastStore = create<ToastStore>((set) => ({
  items: [],
  add: (type, message) => {
    const id = nextId++
    set((state) => ({ items: [...state.items, { id, type, message }] }))
    setTimeout(() => {
      set((state) => ({ items: state.items.filter((toast) => toast.id !== id) }))
    }, 3500)
  },
  remove: (id) => set((state) => ({ items: state.items.filter((toast) => toast.id !== id) })),
}))

export const toast = {
  success: (message: string) => useToastStore.getState().add('success', message),
  error: (message: string) => useToastStore.getState().add('error', message),
  info: (message: string) => useToastStore.getState().add('info', message),
}
