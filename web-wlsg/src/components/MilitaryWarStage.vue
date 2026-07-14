<!-- 军事战争页：用现有军事、世界地图和增援接口驱动官方“调兵遣将”结构。 -->
<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { militaryWarContent, type MilitaryWarOrderContent } from '../data/militaryWarContent'
import type { GeneralAssignmentState, GeneralState, ReinforcementListItem, WorldMapMarchAction } from '../game/types'
import type { RecruitmentUnitViewModel } from '../game/recruitmentAdapter'
import { homeGeneral, militaryWarFoodPerHour, toMilitaryWarReinforcements, toMilitaryWarUnits, type MilitaryWarUnitCatalogItem } from '../game/militaryWarAdapter'
import { formatRemaining, useServerClock } from '../game/useServerClock'
import type { WorldMapTarget } from '../worldMap/types'
import MarchCommandDialog from './MarchCommandDialog.vue'

const props = defineProps<{ subNavigation: string[]; activeMilitaryIndex: number; sourceName: string; faction: string; units: RecruitmentUnitViewModel[]; unitCatalog: MilitaryWarUnitCatalogItem[]; general: GeneralState | null; generals: GeneralState[]; assignments: GeneralAssignmentState[]; sentReinforcements: ReinforcementListItem[]; receivedReinforcements: ReinforcementListItem[]; targets: WorldMapTarget[]; phase: string; error: string; militaryRefreshing: boolean; marchesLoading: boolean; marchesError: string; serverTime: string; receivedAt: number | null; dispatching: boolean; marchMessage: string; marchSucceeded: boolean; marchResultVersion: number }>()
const emit = defineEmits<{ selectMilitary: [index: number]; refresh: []; refreshMarches: []; dispatch: [action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]] }>()
const commandOpen = ref(false)
const commandAction = ref<WorldMapMarchAction>('attack')
const commandStartVersion = ref(0)
const guardUnits = computed(() => toMilitaryWarUnits(props.units))
const guardGeneral = computed(() => homeGeneral(props.general, props.assignments))
const incomingRows = computed(() => toMilitaryWarReinforcements(props.receivedReinforcements, 'incoming', props.unitCatalog))
const outgoingRows = computed(() => toMilitaryWarReinforcements(props.sentReinforcements, 'outgoing', props.unitCatalog))
const commandTargets = computed(() => props.targets.filter((target) => target.targetType === 'player_city' && target.relation !== 'self' && Boolean(target.playerId)))
const commandMessage = computed(() => props.marchResultVersion > commandStartVersion.value ? props.marchMessage : '')
const commandSucceeded = computed(() => props.marchResultVersion > commandStartVersion.value && props.marchSucceeded)
const generalIcon = computed(() => ({ wei: 'general_tag_1.gif', shu: 'general_tag_2.gif', wu: 'general_tag_3.gif' }[props.faction] ?? 'general_tag_1.gif'))
const { remainingSeconds } = useServerClock(toRef(props, 'serverTime'), toRef(props, 'receivedAt'))

/** 只允许在已经实现的战争与征兵页面之间切换。 */
function selectMilitary(index: number) { if (index === 0 || index === 2) emit('selectMilitary', index) }

/** 打开复用的真实战争命令弹窗，目标能力以后端世界地图响应为准。 */
function openOrder(order: MilitaryWarOrderContent) {
  commandAction.value = order.id
  commandStartVersion.value = props.marchResultVersion
  commandOpen.value = true
}

/** 将战争弹窗中的真实目标、兵力和武将原样交给状态服务。 */
function dispatchCommand(action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]) { emit('dispatch', action, targetPlayerId, troops, generalIds) }

/** 同时刷新军事局部视图与增援列表，避免两个区域时间不同步。 */
function refreshWarData() { emit('refresh'); emit('refreshMarches') }

/** 使用中文千分位格式化真实兵力和耗粮。 */
function formatNumber(value: number) { return Number(value || 0).toLocaleString('zh-CN') }

/** 返回增援状态及基于服务端时间的倒计时。 */
function rowStatus(status: string, endsAt: string) {
  if (!endsAt) return status
  const seconds = remainingSeconds(endsAt)
  return `${status} ${seconds > 0 ? formatRemaining(seconds) : '结算中'}`
}

/** 按援军来源阵营显示对应的官方将领标识。 */
function reinforcementGeneralIcon(faction: string) { return ({ wei: 'general_tag_1.gif', shu: 'general_tag_2.gif', wu: 'general_tag_3.gif' }[faction] ?? 'general_tag_1.gif') }
</script>

<template>
  <section class="main-stage military-war-stage" aria-label="军事战争调兵遣将页">
    <div class="panel-top"><i></i><span></span><b></b></div>
    <div class="panel-center military-war-panel-center">
      <nav class="secondary-navigation" aria-label="军事二级导航">
        <button v-for="(item, index) in subNavigation" :key="item" type="button" :class="{ active: index === activeMilitaryIndex }" :disabled="index !== 0 && index !== 2" :title="index === 1 || index === 3 ? '当前页面尚未实现此标签' : ''" @click="selectMilitary(index)">{{ item }}</button>
      </nav>
      <nav class="war-page-tabs" aria-label="战争页标签">
        <button v-for="(tab, index) in militaryWarContent.tabs" :key="tab" type="button" :class="{ active: index === 0 }" :disabled="index !== 0" :title="index === 0 ? '' : '当前页面尚未实现此标签'">{{ tab }}</button>
        <span></span><a href="#" @click.prevent>我的锦囊</a><i></i><a href="#" @click.prevent>购买资源</a><i></i><a href="#" @click.prevent>领取装备</a><i></i><a href="#" @click.prevent>套装帮助</a>
      </nav>
      <div class="military-war-content">
        <div class="war-description">{{ militaryWarContent.description }}<br /><b>{{ militaryWarContent.officialWarning }}</b></div>
        <div v-if="error && phase === 'error'" class="war-data-state error"><span>{{ error }}</span><button type="button" @click="refreshWarData">重新读取</button></div>
        <section class="war-command-panel">
          <header><strong><i></i>战争指令-[鼠标点击下面指令可进行下达]</strong><div><button type="button" disabled title="现有后端未提供本城增援选项设置">本城增援选项</button><img src="/assets/official/military/stop_war_no.gif" alt="" /><button type="button" disabled title="现有后端未提供免战操作">我要免战</button><button type="button" disabled title="现有后端未提供军队攻击强化操作">强化军队攻击力</button></div></header>
          <div class="war-command-grid">
            <button v-for="order in militaryWarContent.orders" :key="order.id" type="button" :disabled="dispatching" :title="order.description" @click="openOrder(order)"><img :src="`/assets/official/military/${order.image}`" :alt="order.label" /></button>
          </div>
          <p class="war-command-preview" aria-live="polite">{{ marchesError || (commandTargets.length ? `已读取 ${commandTargets.length} 个真实玩家城池目标。` : '当前地图视野中没有可下达命令的玩家城池。') }}</p>
        </section>

        <section class="war-army-panel guard-army-panel">
          <header><strong><i></i>本城所属的守城军队</strong><button type="button" :disabled="militaryRefreshing" @click="refreshWarData">{{ militaryRefreshing ? '读取中…' : '刷新军队' }}</button></header>
          <table class="war-guard-table">
            <tbody>
              <tr class="war-general-row"><th><img :src="`/assets/official/images/${generalIcon}`" alt="将领" /></th><td :colspan="Math.max(1, guardUnits.length)">{{ guardGeneral ? `${guardGeneral.name}（等级${guardGeneral.level}）` : '暂无驻城将领' }}</td></tr>
              <tr class="war-unit-icons"><th></th><td v-for="unit in guardUnits" :key="unit.id" :title="unit.name"><img v-if="unit.officialCode" :src="`/assets/official/images/${unit.officialCode}.gif`" :alt="unit.name" /><span v-else class="war-unit-fallback">兵</span></td><td v-if="!guardUnits.length">暂无兵种配置</td></tr>
              <tr class="war-unit-amounts"><th></th><td v-for="unit in guardUnits" :key="unit.id">{{ formatNumber(unit.amount) }}</td><td v-if="!guardUnits.length">0</td></tr>
              <tr class="war-food-row"><th>总耗粮</th><td :colspan="Math.max(1, guardUnits.length)">{{ formatNumber(militaryWarFoodPerHour(guardUnits)) }}/小时</td></tr>
            </tbody>
          </table>
        </section>

        <template v-if="incomingRows.length">
          <section v-for="row in incomingRows" :key="row.id" class="war-army-panel war-reinforcement-detachment">
            <header><strong><i></i>{{ row.playerName }}来增援本城的军队</strong><span>{{ rowStatus(row.status, row.endsAt) }}</span></header>
            <table class="war-guard-table">
              <tbody>
                <tr class="war-general-row"><th><img :src="`/assets/official/images/${reinforcementGeneralIcon(row.faction)}`" alt="将领" /></th><td :colspan="Math.max(1, row.units.length)">{{ row.generalNames }}</td></tr>
                <tr class="war-unit-icons"><th></th><td v-for="unit in row.units" :key="unit.id" :title="unit.name"><img v-if="unit.officialCode" :src="`/assets/official/images/${unit.officialCode}.gif`" :alt="unit.name" /><span v-else class="war-unit-fallback">兵</span></td><td v-if="!row.units.length">暂无兵种配置</td></tr>
                <tr class="war-unit-amounts"><th></th><td v-for="unit in row.units" :key="unit.id">{{ formatNumber(unit.amount) }}</td><td v-if="!row.units.length">0</td></tr>
                <tr class="war-food-row"><th>总耗粮</th><td :colspan="Math.max(1, row.units.length)">{{ formatNumber(row.foodPerHour) }}/小时</td></tr>
              </tbody>
            </table>
          </section>
        </template>
        <section v-else class="war-army-panel war-empty-army-panel">
          <header><strong><i></i>他城来增援本城的军队</strong></header><table><tbody><tr><td>{{ marchesLoading ? '正在读取真实增援…' : '暂无' }}</td></tr></tbody></table>
        </section>
        <section class="war-army-panel war-reinforcement-panel">
          <header><strong><i></i>本城去增援他城的军队</strong></header>
          <table><thead><tr><th>目标城池</th><th>将领</th><th>兵力</th><th>状态</th></tr></thead><tbody><tr v-if="!outgoingRows.length"><td colspan="4">{{ marchesLoading ? '正在读取真实增援…' : '暂无' }}</td></tr><tr v-for="row in outgoingRows" :key="row.id"><td>{{ row.playerName }}</td><td>{{ row.generalNames }}</td><td :title="row.troops">{{ row.troops }}</td><td>{{ rowStatus(row.status, row.endsAt) }}</td></tr></tbody></table>
        </section>
        <section v-for="section in militaryWarContent.unsupportedSections" :key="section.id" class="war-army-panel war-empty-army-panel">
          <header><strong><i></i>{{ section.title }}</strong></header><table><tbody><tr><td>{{ section.message }}</td></tr></tbody></table>
        </section>
      </div>
    </div>
    <div class="panel-bottom"><i></i><span></span><b></b></div>
    <div class="left-footer">抵制不良游戏　拒绝盗版游戏　注意自我保护　谨防受骗上当　适度游戏益脑　沉迷游戏伤身</div>
    <MarchCommandDialog v-if="commandOpen" :target-options="commandTargets" :source-name="sourceName" :units="units" :generals="generals" :assignments="assignments" :initial-action="commandAction" :submitting="dispatching" :message="commandMessage" :succeeded="commandSucceeded" :result-version="marchResultVersion" @close="commandOpen = false" @submit="dispatchCommand" />
  </section>
</template>
