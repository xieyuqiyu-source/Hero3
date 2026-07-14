/** Vue 应用使用的 NPC 页面状态服务单例。 */
import { reactive } from 'vue'
import { playerGameState } from '../game'
import { gameApi } from '../session'
import { createNpcStateService, type NpcStateStore } from './stateService'

const state = reactive<NpcStateStore>({ phase: 'idle', playerId: null, data: null, error: '', refreshing: false, operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0 })
export const npcState = createNpcStateService(gameApi, state, playerGameState.state)
