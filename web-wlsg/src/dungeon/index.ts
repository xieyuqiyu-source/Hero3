/** Vue 应用使用的轮回绝境状态服务单例。 */
import { reactive } from 'vue'
import { gameApi } from '../session'
import { playerGameState } from '../game'
import { createDungeonStateService, type DungeonStateStore } from './stateService'

const state = reactive<DungeonStateStore>({ phase: 'idle', playerId: null, config: null, run: null, error: '', operating: false, actionMessage: '', actionSucceeded: false, resultVersion: 0, lastReportId: '', lastGeneralId: '', lastGeneralExpGained: 0, lastGeneralLevelBefore: null, lastGeneralLevelAfter: null })
export const dungeonState = createDungeonStateService(gameApi, state, playerGameState.state)
