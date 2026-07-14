/** Vue 应用使用的当前玩家状态服务单例。 */
import { reactive } from 'vue'
import { gameApi } from '../session'
import { createGameStateService, type GameStateStore } from './stateService'

const state = reactive<GameStateStore>({ phase: 'idle', playerId: null, data: null, receivedAt: null, error: '', upgradingBuildingId: null, actionMessage: '', completingBuildingId: null, completingAllBuildings: false, buildingInstantMessage: '', buildingInstantSucceeded: false, fillingResources: false, resourceActionMessage: '', resourceActionSucceeded: false, recruitingUnitId: null, completingRecruitQueueId: null, militaryRefreshing: false, recruitActionMessage: '', recruitResultVersion: 0, recruitActionSucceeded: false, recruitActionType: null, dispatchingMarch: false, marchActionMessage: '', marchActionSucceeded: false, marchResultVersion: 0, outgoingMarches: [], outgoingMarchesLoading: false, outgoingMarchesError: '', operatingMarchId: null, operatingMarchAction: null, marchOperationMessage: '', marchOperationSucceeded: false })
export const playerGameState = createGameStateService(gameApi, state)
