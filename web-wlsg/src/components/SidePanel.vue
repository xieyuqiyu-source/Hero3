<!-- 右侧常驻栏使用真实玩家、资源、军队、队列和未读数量。 -->
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import type { PlayerSummary } from '../api/types'
import type { CityGameViewModel, OutgoingMarchViewModel } from '../game/types'
import { formatRemaining, useServerClock } from '../game/useServerClock'

const props = withDefaults(defineProps<{ model: CityGameViewModel; receivedAt: number | null; players?: PlayerSummary[]; currentPlayerId?: string; cityGoldPerSecond?: number; completingBuildingId?: string | null; completingAllBuildings?: boolean; instantMessage?: string; outgoingMarches?: OutgoingMarchViewModel[]; outgoingMarchesLoading?: boolean; outgoingMarchesError?: string }>(), { players: () => [], currentPlayerId: '', cityGoldPerSecond: 120, completingBuildingId: null, completingAllBuildings: false, instantMessage: '', outgoingMarches: () => [], outgoingMarchesLoading: false, outgoingMarchesError: '' })
const emit = defineEmits<{ selectPlayer: [playerId: string]; instantBuilding: [buildingId: string]; instantAllBuildings: []; refreshMarches: [] }>()
const citySelector = ref<HTMLElement | null>(null)
const cityMenuOpen = ref(false)
const marchPanelOpen = ref(false)
const refreshedMarchIds = new Set<string>()
const topShortcuts = [
  { image: 'url_jq1.gif', label: '军情', countKey: 'message' }, { image: 'url_xh1.gif', label: '信函', countKey: 'mail' },
  { image: 'url_cz.gif', label: '充值', countKey: '' }, { image: 'url_zh.gif', label: '账户', countKey: '' },
]
const bottomShortcuts = [
  { image: 'url_rw.gif', label: '任务' }, { image: 'url_bz.gif', label: '帮助' },
  { image: 'url_bbs.gif', label: '论坛' }, { image: 'url_out.gif', label: '退出' },
]
const { remainingSeconds } = useServerClock(toRef(props.model, 'serverTime'), toRef(props, 'receivedAt'))
const instantBusy = computed(() => Boolean(props.completingBuildingId || props.completingAllBuildings))
const nextMarch = computed(() => props.outgoingMarches[0] ?? null)
const nextMarchSeconds = computed(() => nextMarch.value ? remainingSeconds(nextMarch.value.endsAt) : 0)

/** 最近行军到期时只触发一次后端刷新结算。 */
watch(nextMarchSeconds, (seconds) => {
  const march = nextMarch.value
  if (!march || seconds > 0 || refreshedMarchIds.has(march.id)) return
  refreshedMarchIds.add(march.id)
  emit('refreshMarches')
})

/** 点击右侧城池栏时切换存档下拉菜单。 */
function toggleCityMenu() { cityMenuOpen.value = !cityMenuOpen.value }

/** 选择有效存档并关闭下拉菜单。 */
function selectCity(playerId: string) {
  if (playerId === props.currentPlayerId) {
    cityMenuOpen.value = false
    return
  }
  cityMenuOpen.value = false
  emit('selectPlayer', playerId)
}

/** 点击菜单外部时收起城池列表。 */
function closeCityMenuOutside(event: MouseEvent) {
  if (!citySelector.value?.contains(event.target as Node)) cityMenuOpen.value = false
}

onMounted(() => document.addEventListener('click', closeCityMenuOutside))
onBeforeUnmount(() => document.removeEventListener('click', closeCityMenuOutside))

/** 返回快捷入口对应的真实未读数量。 */
function shortcutCount(key: string) {
  if (key === 'message') return props.model.unreadMessageCount
  if (key === 'mail') return props.model.unreadMailCount
  return 0
}

/** 格式化资源和军队真实数值。 */
function formatNumber(value: number) { return Number(value || 0).toLocaleString('zh-CN') }

/** 格式化建造队列的服务端基准剩余时间。 */
function queueText(endsAt: string) {
  const seconds = remainingSeconds(endsAt)
  return seconds > 0 ? formatRemaining(seconds) : '结算中'
}

/** 按后端城金折抵秒数估算单条建造队列加速消耗。 */
function queueInstantCost(endsAt: string) { return Math.max(1, Math.ceil(remainingSeconds(endsAt) / Math.max(1, props.cityGoldPerSecond))) }

/** 汇总当前全部建造队列的预计加速城金。 */
function allQueueInstantCost() { return props.model.buildQueues.reduce((total, queue) => total + queueInstantCost(queue.endsAt), 0) }

/** 返回一键完成的悬浮说明。 */
function instantAllTitle() {
  if (props.instantMessage) return props.instantMessage
  if (!props.model.buildQueues.length) return '当前没有建造队列'
  const cost = allQueueInstantCost()
  if (cost > props.model.cityGold) return `预计需要 ${cost} 城金，当前城金不足`
  return `点击立即完成全部 ${props.model.buildQueues.length} 条队列，预计消耗 ${cost} 城金，最终以后端结算为准`
}

/** 返回单条队列锤子图标的悬浮说明。 */
function instantQueueTitle(endsAt: string) {
  const cost = queueInstantCost(endsAt)
  return cost > props.model.cityGold ? `预计需要 ${cost} 城金，当前城金不足` : `点击立即完成该队列，预计消耗 ${cost} 城金，最终以后端结算为准`
}

/** 格式化活动行军倒计时，未知时间交由后端刷新。 */
function marchCountdown(march: OutgoingMarchViewModel) {
  const seconds = remainingSeconds(march.endsAt)
  return seconds > 0 ? formatRemaining(seconds) : '结算中'
}

/** 汇总一支行军包含的真实兵力数量。 */
function marchTroopCount(march: OutgoingMarchViewModel) { return Object.values(march.troops).reduce((total, amount) => total + amount, 0) }
</script>

<template>
  <aside class="side-panel" aria-label="当前玩家真实状态">
    <div class="right-top">
      <nav class="side-shortcuts top-shortcuts" aria-label="右侧快捷菜单">
        <button v-for="item in topShortcuts" :key="item.label" type="button"><img :src="`/assets/official/images/${item.image}`" :alt="item.label" /><b v-if="shortcutCount(item.countKey)">{{ shortcutCount(item.countKey) }}</b></button>
      </nav>
    </div>
    <div ref="citySelector" class="city-name city-selector">
      <button type="button" class="city-selector-current" aria-haspopup="listbox" :aria-expanded="cityMenuOpen" @click.stop="toggleCityMenu">{{ model.player.nickname }}</button>
      <button type="button" class="city-selector-arrow" aria-label="切换存档" :aria-expanded="cityMenuOpen" @click.stop="toggleCityMenu">↓</button>
      <div v-if="cityMenuOpen" class="city-dropdown" role="listbox" aria-label="账号存档列表">
        <button v-for="player in players" :key="player.id" type="button" role="option" :aria-selected="player.id === currentPlayerId" :class="{ active: player.id === currentPlayerId }" :disabled="Boolean(player.deleteScheduledAt)" :title="player.deleteScheduledAt ? '该存档正在等待删除' : `${player.nickname} · ${player.faction}`" @click.stop="selectCity(player.id)">{{ player.nickname }}</button>
      </div>
    </div>
    <section class="notice-box real-notice"><h2>当前状态</h2><p>账号金币：{{ formatNumber(model.accountGold) }}</p><p>城金：{{ formatNumber(model.cityGold) }}</p><p>坐标、VIP 与推广信息尚未接入</p></section>
    <section v-if="outgoingMarches.length || outgoingMarchesLoading || outgoingMarchesError" class="march-status-section" aria-label="出征状态">
      <div v-if="outgoingMarches.length" class="march-status-line"><strong>出征中</strong><span>×{{ outgoingMarches.length }}</span><b>{{ nextMarch ? marchCountdown(nextMarch) : '' }}</b><button type="button" @click="marchPanelOpen = !marchPanelOpen">{{ marchPanelOpen ? '收起' : '查看' }}</button></div>
      <div v-else-if="outgoingMarchesLoading" class="march-status-line loading"><span>正在读取出征状态…</span></div>
      <div v-else class="march-status-line error"><span>{{ outgoingMarchesError }}</span><button type="button" @click="emit('refreshMarches')">重试</button></div>
      <div v-if="marchPanelOpen && outgoingMarches.length" class="march-status-list">
        <article v-for="march in outgoingMarches" :key="march.id"><header><strong>{{ march.status === 'returning' ? '返回中' : '出征中' }} · {{ march.label }}</strong><b>{{ marchCountdown(march) }}</b></header><p>目标：{{ march.targetName }}</p><p>兵力：{{ formatNumber(marchTroopCount(march)) }}</p></article>
      </div>
    </section>
    <section class="side-section resource-section">
      <h2>城池生产力</h2>
      <div class="resource-row" v-for="resource in model.resources" :key="resource.key">
        <img :src="`/assets/official/images/${resource.icon}`" alt="" /><strong>{{ formatNumber(resource.productionPerHour) }} 每小时</strong>
        <button type="button" disabled title="V0.3 只读展示；购买加成需另行授权"><img src="/assets/official/images/g_cf_2_2.gif" alt="加成" /></button>
        <button type="button" disabled title="查看加成（后续写操作版本接入）"><img src="/assets/official/images/add_0.gif" alt="详情" /></button>
      </div>
    </section>
    <section class="side-section army-section">
      <h2>本城直属军队 <span>{{ model.general ? `武将 Lv.${model.general.level}` : '无驻城武将' }}</span></h2>
      <div v-if="model.general" class="general-row"><img :src="`/assets/official/images/${model.general.icon}`" alt="" />{{ model.general.name }}（等级{{ model.general.level }}）</div>
      <div v-if="model.army.length" class="troop-grid real-troop-grid"><div v-for="unit in model.army" :key="unit.key" class="troop-item" :title="unit.name" tabindex="0" :aria-label="`${unit.name}，数量 ${formatNumber(unit.amount)}`"><img :src="`/assets/official/images/${unit.icon}`" :alt="unit.name" /><span>{{ formatNumber(unit.amount) }}</span><span class="troop-name-tooltip" role="tooltip">{{ unit.name }}</span></div></div>
      <div v-else class="side-empty">当前没有直属军队</div>
    </section>
    <section class="side-section build-section">
      <h2>城池建造 <button type="button" class="instant-all-buildings" :disabled="instantBusy || !model.buildQueues.length || allQueueInstantCost() > model.cityGold" :title="instantAllTitle()" @click="emit('instantAllBuildings')">{{ completingAllBuildings ? '完成中…' : '一键完成' }}</button></h2>
      <div v-for="queue in model.buildQueues" :key="queue.id" class="build-row"><span>↑</span><b>{{ queue.name }}({{ queue.level }}) {{ queueText(queue.endsAt) }}</b><button type="button" class="instant-building" :disabled="instantBusy || queueInstantCost(queue.endsAt) > model.cityGold" :title="instantQueueTitle(queue.endsAt)" :aria-label="`立即完成${queue.name}建造队列`" @click="emit('instantBuilding', queue.id)"><img src="/assets/official/images/speed_up.gif" alt="" /></button></div>
      <div v-if="!model.buildQueues.length" class="side-empty">当前没有建造队列</div>
      <span v-if="instantMessage" class="side-action-status" role="status">{{ instantMessage }}</span>
    </section>
    <div class="right-bottom"></div>
    <nav class="side-shortcuts bottom-shortcuts" aria-label="右侧底部菜单"><button v-for="item in bottomShortcuts" :key="item.label" type="button"><img :src="`/assets/official/images/${item.image}`" :alt="item.label" /></button></nav>
  </aside>
</template>
