import { create } from 'zustand'

const STORAGE_KEY = 'hero3_skip_confirmations'

interface ConfirmPreferenceStore {
  skipConfirmations: boolean
  setSkipConfirmations: (value: boolean) => void
}

function getStoredSkipConfirmations(): boolean {
  return localStorage.getItem(STORAGE_KEY) === 'true'
}

export const useConfirmPreferenceStore = create<ConfirmPreferenceStore>((set) => ({
  skipConfirmations: getStoredSkipConfirmations(),
  setSkipConfirmations: (value) => {
    if (value) localStorage.setItem(STORAGE_KEY, 'true')
    else {
      localStorage.removeItem(STORAGE_KEY)
      sessionStorage.removeItem('npc_warn_dismissed')
    }
    set({ skipConfirmations: value })
  },
}))
