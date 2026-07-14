<!-- 右侧常驻栏使用真实玩家、资源、军队、队列和未读数量。 -->
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import type { PlayerSummary } from '../api/types'
import type { CityGameViewModel, OutgoingMarchViewModel } from '../game/types'
import { formatRemaining, useServerClock } from '../game/useServerClock'

const props = withDefaults(defineProps<{ model: CityGameViewModel; receivedAt: number | null; players?: PlayerSummary[]; currentPlayerId?: string; cityGoldPerSecond?: number; completingBuildingId?: string | null; completingAllBuildings?: boolean; instantMessage?: string; outgoingMarches?: OutgoingMarchViewModel[]; outgoingMarchesLoading?: boolean; outgoingMarchesError?: string; operatingMarchId?: string | null; operatingMarchAction?: 'accelerate' | 'recall' | null; marchOperationMessage?: string; marchOperationSucceeded?: boolean }>(), { players: () => [], currentPlayerId: '', cityGoldPerSecond: 120, completingBuildingId: null, completingAllBuildings: false, instantMessage: '', outgoingMarches: () => [], outgoingMarchesLoading: false, outgoingMarchesError: '', operatingMarchId: null, operatingMarchAction: null, marchOperationMessage: '', marchOperationSucceeded: false })
const emit = defineEmits<{ selectPlayer: [playerId: string]; instantBuilding: [buildingId: string]; instantAllBuildings: []; refreshMarches: []; accelerateMarch: [marchId: string]; recallMarch: [marchId: string]; openIntelligence: [] }>()
const citySelector = ref<HTMLElement | null>(null)
const cityMenuOpen = ref(false)
const openMarchId = ref<string | null>(null)
const refreshedMarchIds = new Set<string>()
const topShortcuts = [
  { image: 'url_jq1.gif', label: '军情', countKey: 'message', action: 'intelligence' }, { image: 'url_xh1.gif', label: '信函', countKey: 'mail', action: '' },
  { image: 'url_cz.gif', label: '充值', countKey: '' }, { image: 'url_zh.gif', label: '账户', countKey: '' },
]
const bottomShortcuts = [
  { image: 'url_rw.gif', label: '任务' }, { image: 'url_bz.gif', label: '帮助' },
  { image: 'url_bbs.gif', label: '论坛' }, { image: 'url_out.gif', label: '退出' },
]
const { remainingSeconds } = useServerClock(toRef(props.model, 'serverTime'), toRef(props, 'receivedAt'))
const instantBusy = computed(() => Boolean(props.completingBuildingId || props.completingAllBuildings))

/** 任一出征或增援到期时只触发一次后端刷新结算。 */
watch(() => props.outgoingMarches.map((march) => `${march.id}:${remainingSeconds(march.endsAt)}`).join('|'), () => {
  const due = props.outgoingMarches.find((march) => remainingSeconds(march.endsAt) <= 0 && !refreshedMarchIds.has(march.id))
  if (!due) return
  refreshedMarchIds.add(due.id)
  emit('refreshMarches')
})

/** 展开指定行军队列并自动收起其它队列。 */
function toggleMarch(marchId: string) { openMarchId.value = openMarchId.value === marchId ? null : marchId }

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

/** 返回单支行军的状态标题，不再按类型汇总。 */
function marchStatusLabel(march: OutgoingMarchViewModel) {
  if (march.status === 'returning') return '返回中'
  if (march.reinforcementRole === 'received') return '被增援'
  if (march.kind === 'reinforce') return '增援中'
  return `${march.label}中`
}

/** 为单支行军选择右侧栏状态色。 */
function marchStatusClass(march: OutgoingMarchViewModel) {
  if (march.reinforcementRole === 'received') return 'received-reinforcement'
  if (march.kind === 'reinforce') return 'reinforcement'
  return 'expedition'
}

/** 返回单支行军的完整悬浮提示。 */
function marchLineTitle(march: OutgoingMarchViewModel) { return `${marchStatusLabel(march)}：${march.targetName}，兵力 ${formatNumber(marchTroopCount(march))}` }

/** 只有本人仍在去程的队列显示加速与召回操作。 */
function marchCanOperate(march: OutgoingMarchViewModel) { return march.status === 'marching' && march.reinforcementRole !== 'received' }

/** 返回加速图标的真实费用、次数和后端限制提示。 */
function accelerateMarchTitle(march: OutgoingMarchViewModel) {
  if (props.operatingMarchId === march.id && props.operatingMarchAction === 'accelerate') return '正在加速行军…'
  const times = Math.max(0, march.acceleratedTimes ?? 0)
  return times >= 2 ? '该队列已达到最多 2 次加速限制' : `加速行军：消耗 10 城金，剩余时间减半（已加速 ${times}/2 次）`
}

/** 返回召回图标的后端状态说明。 */
function recallMarchTitle(march: OutgoingMarchViewModel) {
  if (props.operatingMarchId === march.id && props.operatingMarchAction === 'recall') return '正在召回队伍…'
  return march.kind === 'reinforce' ? '召回增援并进入返程' : '召回队伍并进入返程；PVP 行军仅出发后 2 分钟内允许召回'
}
</script>

<template>
  <aside class="side-panel" aria-label="当前玩家真实状态">
    <div class="right-top">
      <nav class="side-shortcuts top-shortcuts" aria-label="右侧快捷菜单">
        <button v-for="item in topShortcuts" :key="item.label" type="button" :title="item.label" @click="item.action === 'intelligence' && emit('openIntelligence')"><img :src="`/assets/official/images/${item.image}`" :alt="item.label" /><b v-if="shortcutCount(item.countKey)">{{ shortcutCount(item.countKey) }}</b></button>
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
    <section v-if="outgoingMarches.length || outgoingMarchesLoading || outgoingMarchesError || marchOperationMessage" class="march-status-section" aria-label="行军状态">
      <div v-for="march in outgoingMarches" :key="march.id" class="march-status-block" :class="marchStatusClass(march)">
        <div class="march-status-line" :title="marchLineTitle(march)">
          <strong>{{ marchStatusLabel(march) }}</strong><span>{{ march.targetName }}</span><b>{{ marchCountdown(march) }}</b>
          <div class="march-line-controls">
            <button v-if="marchCanOperate(march)" type="button" class="march-action-icon accelerate" :disabled="Boolean(operatingMarchId) || (march.acceleratedTimes ?? 0) >= 2" :title="accelerateMarchTitle(march)" :aria-label="`加速${march.label}队列`" @click.stop="emit('accelerateMarch', march.id)"><img src="/assets/official/images/speed_up.gif" alt="" /></button>
            <button v-if="marchCanOperate(march)" type="button" class="march-action-icon recall" :disabled="Boolean(operatingMarchId)" :title="recallMarchTitle(march)" :aria-label="`召回${march.label}队列`" @click.stop="emit('recallMarch', march.id)"><span aria-hidden="true">↩</span></button>
            <button type="button" class="march-detail-toggle" @click="toggleMarch(march.id)">{{ openMarchId === march.id ? '收起' : '查看' }}</button>
          </div>
        </div>
        <div v-if="openMarchId === march.id" class="march-status-list">
          <article><header><strong>{{ march.status === 'returning' ? `返回中 · ${march.label}` : marchStatusLabel(march) }}</strong><b>{{ marchCountdown(march) }}</b></header><p>目标：{{ march.targetName }}</p><p>兵力：{{ formatNumber(marchTroopCount(march)) }}</p></article>
        </div>
      </div>
      <div v-if="!outgoingMarches.length && outgoingMarchesLoading" class="march-status-line loading"><span>正在读取行军状态…</span></div>
      <div v-else-if="!outgoingMarches.length && outgoingMarchesError" class="march-status-line error"><span>{{ outgoingMarchesError }}</span><button type="button" @click="emit('refreshMarches')">重试</button></div>
      <p v-if="marchOperationMessage" class="march-operation-message" :class="{ error: !marchOperationSucceeded }" role="status">{{ marchOperationMessage }}</p>
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
