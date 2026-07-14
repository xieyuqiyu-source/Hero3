<!-- 使用 Hero3 真实兵种、资源与征兵队列的官方风格征兵页面。 -->
<script setup lang="ts">
import { computed, ref, toRef, watch } from 'vue'
import { baseRecruitCost, estimateInstantCost, RECRUIT_QUEUE_LIMIT, recruitmentStatLabels, type RecruitmentCategoryViewModel, type RecruitmentQueueViewModel, type RecruitmentUnitViewModel } from '../game/recruitmentAdapter'
import type { GameStatePhase } from '../game/stateService'
import { formatRemaining, useServerClock } from '../game/useServerClock'

const props = defineProps<{
  subNavigation: string[]
  activeMilitaryIndex: number
  categories: RecruitmentCategoryViewModel[]
  queues: RecruitmentQueueViewModel[]
  resources: Record<string, number>
  cityGold: number
  cityGoldPerSecond: number
  phase: GameStatePhase
  error: string
  serverTime: string
  receivedAt: number | null
  recruitingUnitId: string | null
  completingQueueId: string | null
  militaryRefreshing: boolean
  actionMessage: string
  resultVersion: number
  actionSucceeded: boolean
  actionType: 'recruit' | 'instant' | null
}>()
const emit = defineEmits<{ retry: []; recruit: [unitId: string, amount: number]; instant: [queueId: string]; selectMilitary: [index: number] }>()
const activeCategoryId = ref('infantry')
const selectedUnit = ref<RecruitmentUnitViewModel | null>(null)
const instantQueue = ref<RecruitmentQueueViewModel | null>(null)
const quantity = ref(1)
const inputError = ref('')
const dueQueueIds = new Set<string>()
const { nowMs, remainingSeconds } = useServerClock(toRef(props, 'serverTime'), toRef(props, 'receivedAt'))
const activeCategory = computed(() => props.categories.find((category) => category.id === activeCategoryId.value) ?? props.categories[0])
const unitCost = computed(() => selectedUnit.value ? baseRecruitCost(selectedUnit.value, 1) : {})
const maxRecruitable = computed(() => {
  if (!selectedUnit.value) return 0
  const limits = Object.entries(selectedUnit.value.cost).filter(([, cost]) => cost > 0).map(([key, cost]) => Math.floor((props.resources[key] ?? 0) / cost))
  return Math.max(0, Math.min(100000, ...(limits.length ? limits : [100000])))
})
const actionBusy = computed(() => Boolean(props.recruitingUnitId || props.completingQueueId))
const instantRemaining = computed(() => instantQueue.value ? remainingSeconds(instantQueue.value.endsAt) : 0)
const instantCost = computed(() => estimateInstantCost(instantRemaining.value, props.cityGoldPerSecond))

watch(() => props.categories, (categories) => {
  if (!categories.some((category) => category.id === activeCategoryId.value)) activeCategoryId.value = categories[0]?.id ?? 'infantry'
  if (selectedUnit.value && !categories.some((category) => category.units.some((unit) => unit.id === selectedUnit.value?.id))) selectedUnit.value = null
}, { deep: true })

watch(() => props.resultVersion, () => {
  if (!props.actionSucceeded) return
  if (props.actionType === 'recruit') selectedUnit.value = null
  if (props.actionType === 'instant') instantQueue.value = null
})

watch(nowMs, () => {
  for (const queue of props.queues) {
    if (remainingSeconds(queue.endsAt) === 0 && !dueQueueIds.has(queue.id)) {
      dueQueueIds.add(queue.id)
      emit('retry')
      break
    }
  }
})

/** 切换征兵分类。 */
function selectCategory(categoryId: string) { activeCategoryId.value = categoryId }

/** 只允许切换至已经完成的战争页或当前征兵页。 */
function selectMilitary(index: number) { if (index === 0 || index === 2) emit('selectMilitary', index) }

/** 打开真实征兵确认弹窗。 */
function openRecruitment(unit: RecruitmentUnitViewModel) {
  selectedUnit.value = unit
  quantity.value = 0
  inputError.value = ''
}

/** 关闭征兵确认弹窗。 */
function closeRecruitment() { if (!props.recruitingUnitId) selectedUnit.value = null }

/** 按官网控件将征兵数量增减一名。 */
function changeQuantity(delta: number) { quantity.value = Math.min(maxRecruitable.value, Math.max(0, Math.floor(Number(quantity.value) || 0) + delta)) }

/** 按官网最大值按钮填入当前资源可承担数量。 */
function fillMaximum() { quantity.value = maxRecruitable.value }

/** 聚焦数量输入时全选当前值，复刻官网直接覆盖输入的操作。 */
function selectQuantityInput(event: FocusEvent) { (event.target as HTMLInputElement).select() }

/** 校验数量后提交既有征兵接口。 */
function confirmRecruitment() {
  if (!selectedUnit.value) return
  const value = Number(quantity.value)
  if (!Number.isInteger(value) || value < 1 || value > 100000) {
    inputError.value = '征兵数量必须是 1 至 100000 的整数'
    return
  }
  inputError.value = ''
  emit('recruit', selectedUnit.value.id, value)
}

/** 打开立即完成二次确认。 */
function openInstant(queue: RecruitmentQueueViewModel) { instantQueue.value = queue }

/** 提交立即完成现有队列接口。 */
function confirmInstant() { if (instantQueue.value) emit('instant', instantQueue.value.id) }

/** 格式化兵力、资源与城金数字。 */
function formatNumber(value: number) { return Number(value ?? 0).toLocaleString('zh-CN') }

/** 按官网弹窗固定两位小时格式显示单兵训练时间。 */
function formatOfficialDuration(total: number) {
  const seconds = Math.max(0, Math.floor(total))
  return [Math.floor(seconds / 3600), Math.floor(seconds % 3600 / 60), seconds % 60].map((value) => String(value).padStart(2, '0')).join(':')
}

/** 按官网队列表格格式显示完成时间。 */
function formatCompletionTime(value: string) {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return '--'
  const parts = [date.getFullYear(), date.getMonth() + 1, date.getDate(), date.getHours(), date.getMinutes(), date.getSeconds()].map((part) => String(part).padStart(2, '0'))
  return `${parts[0]}-${parts[1]}-${parts[2]} ${parts[3]}:${parts[4]}:${parts[5]}`
}

/** 返回官方兵种图片地址。 */
function unitImage(code: number | null) { return code ? `/assets/official/recruitment/${code}.gif` : '' }
</script>

<template>
  <section class="main-stage recruitment-stage" aria-label="军事征兵页">
    <div class="panel-top"><i></i><span></span><b></b></div>
    <div class="panel-center recruitment-panel-center">
      <nav class="secondary-navigation" aria-label="军事二级导航"><button v-for="(item, index) in subNavigation" :key="item" type="button" :class="{ active: index === activeMilitaryIndex }" :disabled="index !== 0 && index !== 2" :title="index === 1 || index === 3 ? '当前页面尚未复刻' : ''" @click="selectMilitary(index)">{{ item }}</button></nav>
      <nav class="stage-toolbar recruitment-tabs" aria-label="征兵分类">
        <button v-for="category in categories" :key="category.id" type="button" :class="{ active: category.id === activeCategory?.id }" @click="selectCategory(category.id)">{{ category.label }}</button>
      </nav>
      <div class="stage-content recruitment-content">
        <div v-if="phase === 'loading'" class="recruit-empty"><strong>正在读取真实军事数据…</strong></div>
        <div v-else-if="phase === 'error'" class="recruit-empty"><strong>军事数据加载失败</strong><p>{{ error }}</p><button type="button" @click="emit('retry')">重新读取</button></div>
        <template v-else-if="activeCategory">
          <div class="official-description recruitment-description">{{ activeCategory.description }}<br />(最多 <b>{{ activeCategory.queueLimit }}</b> 条征兵队列，每条最多征募 <b>{{ activeCategory.batchLimit }}</b> 名{{ activeCategory.unitLabel }})</div>
          <section class="official-queue-panel" aria-label="征兵队列">
            <header class="official-queue-title">
              <span v-if="queues.length">当前征兵队列将在 <b>{{ formatOfficialDuration(remainingSeconds(queues[0].endsAt)) }}</b> 内完成</span>
              <span v-else>{{ militaryRefreshing ? '正在刷新征兵队列…' : '当前没有征兵队列' }}</span>
              <strong>[队列数： {{ queues.length }}/{{ RECRUIT_QUEUE_LIMIT }}]</strong>
            </header>
            <div class="official-queue-table-frame">
              <table class="official-queue-table">
                <thead><tr><th>图标</th><th>数量</th><th>兵种名称</th><th>所需时间</th><th>完成时间</th><th>加速征募</th></tr></thead>
                <tbody>
                  <tr v-for="(queue, index) in queues" :key="queue.id">
                    <td><img v-if="queue.officialCode" :src="`/assets/official/images/${queue.officialCode}.gif`" :alt="queue.unitName" /><span v-else class="queue-fallback">兵</span></td>
                    <td>{{ formatNumber(queue.amount) }}</td><td>{{ queue.unitName }}</td>
                    <td>{{ index === 0 ? formatOfficialDuration(remainingSeconds(queue.endsAt)) : '等待队列' }}</td>
                    <td>{{ formatCompletionTime(queue.endsAt) }}</td>
                    <td><button type="button" :disabled="actionBusy || remainingSeconds(queue.endsAt) === 0" @click="openInstant(queue)">激活</button></td>
                  </tr>
                  <tr v-if="!queues.length" class="official-queue-empty"><td colspan="6">当前没有正在征募的兵种</td></tr>
                </tbody>
              </table>
            </div>
          </section>
          <p v-if="actionMessage" class="recruit-preview-message" :class="{ error: !actionSucceeded }" aria-live="polite">{{ actionMessage }} <button v-if="!actionSucceeded" type="button" @click="emit('retry')">重试读取</button></p>
          <div v-if="activeCategory.units.length" class="recruit-unit-grid">
            <article v-for="unit in activeCategory.units" :key="unit.id" class="recruit-unit-card">
              <div class="recruit-portrait" tabindex="0">
                <img v-if="unit.officialCode" :src="unitImage(unit.officialCode)" :alt="unit.name" />
                <div v-else class="recruit-image-fallback"><strong>{{ unit.name }}</strong><span>暂无官方缩略图</span></div>
                <section class="recruit-unit-tooltip" role="tooltip"><h3>{{ unit.name }}</h3><p>{{ unit.description }}</p></section>
              </div>
              <div class="recruit-unit-info">
                <h2>{{ unit.name }}</h2>
                <dl class="recruit-stats"><div v-for="(stat, index) in unit.stats" :key="recruitmentStatLabels[index]"><dt>{{ recruitmentStatLabels[index] }}</dt><dd>{{ stat }}</dd></div></dl>
                <div class="recruit-owned"><span>当前有 <b>{{ formatNumber(unit.owned) }}</b></span>
                  <button type="button" :disabled="actionBusy || queues.length >= RECRUIT_QUEUE_LIMIT" :title="queues.length >= RECRUIT_QUEUE_LIMIT ? '征兵队列已满' : `征募${unit.name}`" @click.stop="openRecruitment(unit)"><img src="/assets/official/recruitment/n_zm.gif" :alt="`征募${unit.name}`" /></button>
                </div>
              </div>
            </article>
          </div>
          <div v-else class="recruit-empty"><strong>当前没有可征募项目</strong><p>{{ activeCategory.description }}</p></div>
        </template>
      </div>
    </div>
    <div class="panel-bottom"><i></i><span></span><b></b></div>
    <div class="left-footer">抵制不良游戏　拒绝盗版游戏　注意自我保护　谨防受骗上当　适度游戏益脑　沉迷游戏伤身</div>

    <div v-if="selectedUnit" class="recruit-modal-mask official-recruit-mask" role="presentation" @click.self="closeRecruitment">
      <section class="official-recruit-modal" role="dialog" aria-modal="true" :aria-label="`征募${selectedUnit.name}`">
        <div class="official-recruit-avatar">
          <img v-if="selectedUnit.officialCode" :src="`/assets/official/recruitment/confirm/${selectedUnit.officialCode}.gif`" :alt="selectedUnit.name" />
          <span v-else>兵</span>
        </div>
        <div class="official-recruit-detail">
          <div class="official-recruit-title"><span>兵种名称:</span><b>{{ selectedUnit.name }}</b> <span>现有数量:</span><b>{{ formatNumber(selectedUnit.owned) }}</b></div>
          <ul class="official-recruit-costs" aria-label="单个兵种基础消耗">
            <li title="木材"><b>{{ formatNumber(unitCost.wood ?? 0) }}</b></li><li title="泥土"><b>{{ formatNumber(unitCost.stone ?? 0) }}</b></li><li title="铁矿"><b>{{ formatNumber(unitCost.iron ?? 0) }}</b></li><li title="粮食"><b>{{ formatNumber(unitCost.food ?? 0) }}</b></li><li title="时间"><b>{{ formatOfficialDuration(selectedUnit.trainSeconds) }}</b></li>
          </ul>
          <div class="official-recruit-controls">
            <button type="button" aria-label="减少一名" :disabled="Boolean(recruitingUnitId)" @click="changeQuantity(-1)"><img src="/assets/official/recruitment/confirm/zb_jt_a.gif" alt="" /></button>
            <input v-model.number="quantity" type="text" inputmode="numeric" aria-label="征募数量" :disabled="Boolean(recruitingUnitId)" @focus="selectQuantityInput" />
            <button type="button" aria-label="增加一名" :disabled="Boolean(recruitingUnitId)" @click="changeQuantity(1)"><img src="/assets/official/recruitment/confirm/zb_jt_b.gif" alt="" /></button>
            <button type="button" aria-label="填入最大数量" :disabled="Boolean(recruitingUnitId)" @click="fillMaximum"><img src="/assets/official/recruitment/confirm/zb_jt_c.gif" alt="" /></button>
            <button type="button" class="official-recruit-limit" :disabled="Boolean(recruitingUnitId)" @click="fillMaximum">(<b>{{ formatNumber(maxRecruitable) }}</b>/{{ formatNumber(maxRecruitable) }})</button>
            <button type="button" aria-label="确认征募" :disabled="Boolean(recruitingUnitId)" @click="confirmRecruitment"><img src="/assets/official/recruitment/confirm/zb_jt_d.gif" alt="征募" /></button>
            <button type="button" aria-label="关闭" :disabled="Boolean(recruitingUnitId)" @click="closeRecruitment"><img src="/assets/official/recruitment/confirm/jz_b_6.gif" alt="关闭" /></button>
            <span v-if="recruitingUnitId">提交中…</span><span v-else-if="inputError" class="official-recruit-error">{{ inputError }}</span>
          </div>
        </div>
      </section>
    </div>

    <div v-if="instantQueue" class="recruit-modal-mask" role="presentation" @click.self="instantQueue = null">
      <section class="recruit-modal instant-modal" role="dialog" aria-modal="true" aria-label="立即完成征兵队列">
        <header><strong>立即完成征兵</strong><button type="button" aria-label="关闭" :disabled="Boolean(completingQueueId)" @click="instantQueue = null">×</button></header>
        <div class="instant-body"><p>{{ instantQueue.unitName }} × {{ formatNumber(instantQueue.amount) }}</p><p>剩余时间：<b>{{ formatRemaining(instantRemaining) }}</b></p><p>预计消耗城金：<b>{{ formatNumber(instantCost) }}</b>（当前 {{ formatNumber(cityGold) }}）</p><small>最终城金消耗以后端返回为准。</small></div>
        <footer><button type="button" :disabled="Boolean(completingQueueId)" @click="confirmInstant">{{ completingQueueId ? '处理中…' : '确认加速' }}</button><button type="button" :disabled="Boolean(completingQueueId)" @click="instantQueue = null">取消</button></footer>
      </section>
    </div>
  </section>
</template>
