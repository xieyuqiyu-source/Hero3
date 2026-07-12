/** Vue 应用使用的世界地图状态服务单例。 */
import { reactive } from 'vue'
import { gameApi } from '../session'
import { createWorldMapStateService, type WorldMapStateStore } from './stateService'

const state = reactive<WorldMapStateStore>({ phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', overviewPhase: 'idle', overview: null, overviewError: '' })
export const worldMapState = createWorldMapStateService(gameApi, state)
