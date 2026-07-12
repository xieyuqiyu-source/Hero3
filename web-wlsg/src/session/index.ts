/** Vue 应用使用的 V0.2 会话服务单例。 */
import { reactive } from 'vue'
import { createApiClient } from '../api/client'
import { createGameApi } from '../api/gameApi'
import { createSessionService, type SessionService, type SessionState } from './service'
import { createSessionStorage } from './storage'

const storage = createSessionStorage(window.localStorage)
const state = reactive<SessionState>({
  phase: 'loading', account: null, players: [], currentPlayer: null, submitting: false, error: '', bootstrap: null,
})

let session: SessionService
export const apiClient = createApiClient({
  baseUrl: (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, ''),
  getToken: storage.getToken,
  onUnauthorized: () => session.handleUnauthorized(),
})
export const gameApi = createGameApi(apiClient)
session = createSessionService(gameApi, storage, state)

export const sessionService = session
