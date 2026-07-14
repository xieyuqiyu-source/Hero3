/** Vue 应用使用的真实军情状态服务单例。 */
import { reactive } from 'vue'
import { playerGameState } from '../game'
import { gameApi } from '../session'
import { createIntelligenceService, type IntelligenceStore } from './stateService'

const state = reactive<IntelligenceStore>({ phase: 'idle', playerId: null, activeTab: 'all', page: 1, pageSize: 8, reports: [], total: 0, error: '', detail: null, detailLoading: false, detailError: '', deleting: false, actionMessage: '' })

export const intelligenceState = createIntelligenceService(gameApi, state, (result, playerId) => {
  const game = playerGameState.state
  if (game.playerId !== playerId || !game.data) return
  Object.assign(game.data, { unreadMessageCount: result.unreadMessageCount, serverTime: result.serverTime })
  game.receivedAt = Date.now()
})
