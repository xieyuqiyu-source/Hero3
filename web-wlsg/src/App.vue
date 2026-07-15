<!-- V0.2 应用状态入口：启动、登录、选档与已通过的公共外壳。 -->
<script setup lang="ts">
import { onMounted, watch } from 'vue'
import GameShell from './components/GameShell.vue'
import LoginView from './components/LoginView.vue'
import PlayerSelectView from './components/PlayerSelectView.vue'
import { playerGameState } from './game'
import { sessionService } from './session'
import { worldMapState } from './worldMap'
import { intelligenceState } from './intelligence'
import { npcState } from './npc'
import { dungeonState } from './dungeon'
import { mirageState } from './mirage'

const state = sessionService.state

/** 提交登录且由状态服务统一展示可读错误。 */
async function login(username: string, password: string) {
  await sessionService.login(username, password).catch(() => undefined)
}

onMounted(() => sessionService.initialize())

/** 退出时同步清除玩家真实状态，避免上一名玩家数据残留。 */
function logout() {
  playerGameState.clear()
  worldMapState.clear()
  intelligenceState.clear()
  npcState.clear()
  dungeonState.clear()
  mirageState.clear()
  sessionService.logout()
}

/** 扣费刷新 NPC 后同步页头的账户金币权威余额。 */
async function refreshNpcCities() {
  const accountGold = await npcState.refresh()
  if (accountGold !== null) sessionService.updateAccountGold(accountGold)
}

/** 重置副本加成后同步页头账户金币。 */
async function resetDungeonBonus(waveId: string) {
  const accountGold = await dungeonState.resetBonus(waveId)
  if (accountGold !== null) sessionService.updateAccountGold(accountGold)
}

watch(
  () => [state.phase, state.currentPlayer?.id] as const,
  ([phase, playerId]) => {
    if (phase === 'game' && playerId) {
      void playerGameState.load(playerId)
      void worldMapState.load(playerId)
      void worldMapState.loadOverview(playerId)
    } else {
      playerGameState.clear()
      worldMapState.clear()
      intelligenceState.clear()
      npcState.clear()
      dungeonState.clear()
      mirageState.clear()
    }
  },
)
</script>

<template>
  <main v-if="state.phase === 'loading'" class="auth-page"><section class="status-card"><span class="loading-mark"></span><h1>正在连接 Hero3</h1><p>加载公共配置并检查本地会话…</p></section></main>
  <main v-else-if="state.phase === 'error'" class="auth-page"><section class="status-card"><h1>暂时无法进入游戏</h1><p class="form-error">{{ state.error }}</p><button type="button" @click="sessionService.initialize">重新连接</button></section></main>
  <LoginView v-else-if="state.phase === 'login'" :submitting="state.submitting" :error="state.error" @login="login" />
  <PlayerSelectView v-else-if="state.phase === 'players' && state.account" :account="state.account" :players="state.players" :error="state.error" @select="sessionService.selectPlayer" @logout="sessionService.logout" />
  <GameShell v-else-if="state.phase === 'game' && state.account && state.currentPlayer" :account="state.account" :players="state.players" :current-player-id="state.currentPlayer.id" :game="playerGameState.state" :world-map="worldMapState.state" :npc="npcState.state" :dungeon="dungeonState.state" :mirage="mirageState.state" :building-configs="state.bootstrap?.balance.buildings ?? {}" :units-config="state.bootstrap?.units ?? {}" :city-gold-per-second="state.bootstrap?.balance.cityGoldPerSecond ?? 120" @select-player="sessionService.selectPlayer" @retry="playerGameState.refresh" @refresh-military="playerGameState.refreshMilitary" @refresh-map="worldMapState.refresh" @load-npc="npcState.load" @refresh-npc="refreshNpcCities" @dispatch-npc="npcState.dispatch" @load-dungeon="dungeonState.load" @start-dungeon="dungeonState.start" @fight-dungeon="dungeonState.fight" @reset-dungeon-bonus="resetDungeonBonus" @settle-dungeon="dungeonState.settle" @load-mirage="mirageState.load" @gamble-mirage="mirageState.gamble" @spin-mirage="mirageState.spin" @redeem-mirage="mirageState.redeem" @redeem-all-mirage="mirageState.redeemAll" @refresh-marches="playerGameState.refreshOutgoingMarches" @accelerate-march="playerGameState.accelerateOutgoingMarch" @recall-march="playerGameState.recallOutgoingMarch" @navigate-map="worldMapState.navigate" @return-map-home="worldMapState.returnHome" @dispatch-march="playerGameState.dispatchWorldMapCommand" @upgrade="playerGameState.upgradeBuilding" @fill-resources="playerGameState.fillResourcesPaid" @load-capacity-boost-prices="playerGameState.loadCapacityBoostPrices" @purchase-capacity-boost="playerGameState.purchaseCapacityBoost" @instant-building="playerGameState.instantCompleteBuilding" @instant-all-buildings="playerGameState.instantCompleteAllBuildings" @recruit="playerGameState.recruit" @instant-recruit="playerGameState.instantCompleteRecruit" @logout="logout" />
</template>
