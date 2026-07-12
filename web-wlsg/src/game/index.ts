/** Vue 应用使用的当前玩家状态服务单例。 */
import { reactive } from 'vue'
import { gameApi } from '../session'
import { createGameStateService, type GameStateStore } from './stateService'

const state = reactive<GameStateStore>({ phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', upgradingBuildingId: null, actionMessage: '', fillingResources: false, resourceActionMessage: '', resourceActionSucceeded: false, recruitingUnitId: null, completingRecruitQueueId: null, militaryRefreshing: false, recruitActionMessage: '', recruitResultVersion: 0, recruitActionSucceeded: false, recruitActionType: null })
export const playerGameState = createGameStateService(gameApi, state)
