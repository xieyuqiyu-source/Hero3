<!-- 将 V0.3 真实游戏状态装配进已验收的 997px 公共外壳。 -->
<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import type { AccountInfo, BuildingBalanceConfig, PlayerSummary, UnitsConfig } from '../api/types'
import { shellNavigation } from '../data/shellNavigation'
import type { GameStateStore } from '../game/stateService'
import type { WorldMapMarchAction } from '../game/types'
import type { WorldMapStateStore } from '../worldMap/stateService'
import { toRecruitmentCategories, toRecruitmentQueues } from '../game/recruitmentAdapter'
import { toCityGameViewModel } from '../game/viewModel'
import { useServerClock } from '../game/useServerClock'
import ChatDock from './ChatDock.vue'
import GameHeader from './GameHeader.vue'
import MainStage from './MainStage.vue'
import MapStage from './MapStage.vue'
import RecruitmentStage from './RecruitmentStage.vue'
import ResourceBar from './ResourceBar.vue'
import SessionStatus from './SessionStatus.vue'
import SidePanel from './SidePanel.vue'

const props = withDefaults(defineProps<{ account: AccountInfo; players?: PlayerSummary[]; currentPlayerId?: string; game: GameStateStore; worldMap: WorldMapStateStore; buildingConfigs: Record<string, BuildingBalanceConfig>; unitsConfig: UnitsConfig; cityGoldPerSecond: number }>(), { players: () => [], currentPlayerId: '' })
const emit = defineEmits<{ logout: []; selectPlayer: [playerId: string]; retry: []; refreshMilitary: []; refreshMap: []; refreshMarches: []; navigateMap: [x: number, y: number]; returnMapHome: []; dispatchMarch: [action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]]; upgrade: [buildingId: string]; fillResources: []; instantBuilding: [buildingId: string]; instantAllBuildings: []; recruit: [unitId: string, amount: number]; instantRecruit: [queueId: string] }>()
const model = computed(() => props.game.data ? toCityGameViewModel(props.game.data, props.account.gold) : null)
const recruitmentCategories = computed(() => toRecruitmentCategories(model.value?.player.faction ?? 'wei', props.unitsConfig, props.game.data?.army ?? []))
const recruitmentQueues = computed(() => toRecruitmentQueues(props.game.data?.recruitQueues ?? [], recruitmentCategories.value))
const marchUnits = computed(() => recruitmentCategories.value.flatMap((category) => category.units))
const serverTime = computed(() => model.value?.serverTime ?? '')
const { formatted } = useServerClock(serverTime, toRef(props.game, 'receivedAt'))
const headerAccount = computed(() => ({ serverTime: formatted.value, currencies: [`账号金币: ${props.account.gold.toLocaleString('zh-CN')}`, `城金: ${(model.value?.cityGold ?? 0).toLocaleString('zh-CN')}`], quickLinks: [] }))
const activePrimaryIndex = ref(1)

/** 在已实现的城池、军事征兵和真实地图页之间切换。 */
function selectPrimary(index: number) {
  if (index === 0 || index === 1 || index === 2) activePrimaryIndex.value = index
}

/** 按当前一级页面刷新对应真实状态。 */
function refreshCurrentPage() {
  if (activePrimaryIndex.value === 2) emit('refreshMap')
  else emit('retry')
}

/** 转交地图组件发出的二维中心坐标。 */
function navigateMap(x: number, y: number) { emit('navigateMap', x, y) }

/** 转交地图战争命令及其真实目标、兵力和武将。 */
function dispatchMarch(action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]) { emit('dispatchMarch', action, targetPlayerId, troops, generalIds) }

/** 将征兵组件的兵种与数量原样转交状态服务。 */
function submitRecruit(unitId: string, amount: number) { emit('recruit', unitId, amount) }

/** 初次整页失败走完整重载，已有数据时只刷新军事局部状态。 */
function retryRecruitment() {
  if (props.game.data) emit('refreshMilitary')
  else emit('retry')
}
</script>

<template>
  <div class="page-shell">
    <main class="game-canvas" :aria-label="activePrimaryIndex === 2 ? '武林三国风格世界地图页' : activePrimaryIndex === 1 ? '武林三国风格军事征兵页' : '武林三国风格当前城池资源页'">
      <GameHeader :account="headerAccount" :navigation="shellNavigation.primary" :active-index="activePrimaryIndex" @select="selectPrimary" />
      <ResourceBar :resources="model?.resources ?? []" :city-gold="model?.cityGold ?? 0" :filling="game.fillingResources" :action-message="game.resourceActionMessage" :action-succeeded="game.resourceActionSucceeded" @fill="emit('fillResources')" />
      <SessionStatus :username="account.username" :nickname="model?.player.nickname ?? '加载中'" @logout="emit('logout')" />
      <button type="button" class="game-refresh" :disabled="activePrimaryIndex === 2 ? worldMap.phase === 'loading' : game.phase === 'loading'" @click="refreshCurrentPage">{{ (activePrimaryIndex === 2 ? worldMap.phase : game.phase) === 'loading' ? '读取中' : '刷新' }}</button>
      <div class="game-body">
        <RecruitmentStage v-if="activePrimaryIndex === 1" :sub-navigation="shellNavigation.militarySecondary" :categories="recruitmentCategories" :queues="recruitmentQueues" :resources="game.data?.resources.items ?? {}" :city-gold="game.data?.cityGold ?? 0" :city-gold-per-second="cityGoldPerSecond" :phase="game.phase" :error="game.error" :server-time="game.data?.serverTime ?? ''" :received-at="game.receivedAt" :recruiting-unit-id="game.recruitingUnitId" :completing-queue-id="game.completingRecruitQueueId" :military-refreshing="game.militaryRefreshing" :action-message="game.recruitActionMessage" :result-version="game.recruitResultVersion" :action-succeeded="game.recruitActionSucceeded" :action-type="game.recruitActionType" @retry="retryRecruitment" @recruit="submitRecruit" @instant="emit('instantRecruit', $event)" />
        <MapStage v-else-if="activePrimaryIndex === 2" :phase="worldMap.phase" :data="worldMap.data" :error="worldMap.error" :overview-phase="worldMap.overviewPhase" :overview="worldMap.overview" :overview-error="worldMap.overviewError" :source-name="model?.player.nickname ?? ''" :units="marchUnits" :generals="game.data?.generals ?? (game.data?.general ? [game.data.general] : [])" :assignments="game.data?.generalAssignments ?? []" :dispatching="game.dispatchingMarch" :march-message="game.marchActionMessage" :march-succeeded="game.marchActionSucceeded" :march-result-version="game.marchResultVersion" @retry="emit('refreshMap')" @navigate="navigateMap" @home="emit('returnMapHome')" @dispatch="dispatchMarch" />
        <MainStage v-else :sub-navigation="shellNavigation.secondary" :buildings="model?.resourceBuildings ?? []" :building-configs="buildingConfigs" :resources="game.data?.resources.items ?? {}" :phase="game.phase" :error="game.error" :action-message="game.actionMessage" :upgrading-building-id="game.upgradingBuildingId" @retry="emit('retry')" @upgrade="emit('upgrade', $event)" />
        <SidePanel v-if="model" :model="model" :received-at="game.receivedAt" :players="players" :current-player-id="currentPlayerId" :city-gold-per-second="cityGoldPerSecond" :completing-building-id="game.completingBuildingId" :completing-all-buildings="game.completingAllBuildings" :instant-message="game.buildingInstantMessage" :outgoing-marches="game.outgoingMarches" :outgoing-marches-loading="game.outgoingMarchesLoading" :outgoing-marches-error="game.outgoingMarchesError" @select-player="emit('selectPlayer', $event)" @instant-building="emit('instantBuilding', $event)" @instant-all-buildings="emit('instantAllBuildings')" @refresh-marches="emit('refreshMarches')" />
        <aside v-else class="side-panel loading-side" aria-label="玩家状态加载中"><div class="right-top"></div><p>{{ game.phase === 'error' ? '真实状态加载失败' : '正在加载真实状态…' }}</p></aside>
      </div>
    </main>
    <ChatDock />
  </div>
</template>
