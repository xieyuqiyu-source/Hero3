/** Vue 应用使用的万象幻境状态服务单例。 */
import { reactive } from 'vue'
import { gameApi } from '../session'
import { playerGameState } from '../game'
import { createMirageStateService, type MirageStateStore } from './stateService'

const state = reactive<MirageStateStore>({ phase: 'idle', playerId: null, summary: null, error: '', operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0, gamblingResult: null, slotResult: null })
export const mirageState = createMirageStateService(gameApi, state, playerGameState.state)
