<!-- 官方战争命令弹窗：将攻击、掠夺、侦查和增援输入转换为真实 Hero3 行军请求。 -->
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { GeneralAssignmentState, GeneralState, WorldMapMarchAction } from '../game/types'
import type { RecruitmentUnitViewModel } from '../game/recruitmentAdapter'
import type { WorldMapTarget } from '../worldMap/types'

const props = withDefaults(defineProps<{ target?: WorldMapTarget | null; targetOptions?: WorldMapTarget[]; sourceName: string; units: RecruitmentUnitViewModel[]; generals: GeneralState[]; assignments: GeneralAssignmentState[]; initialAction: WorldMapMarchAction; submitting: boolean; message: string; succeeded: boolean; resultVersion: number }>(), { target: null, targetOptions: () => [] })
const emit = defineEmits<{ close: []; submit: [action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]] }>()
const action = ref<WorldMapMarchAction>(props.initialAction)
const amounts = reactive<Record<string, number>>({})
const selectedGeneralId = ref('')
const selectedTargetId = ref('')
const inputError = ref('')
const actions: Array<{ key: WorldMapMarchAction; label: string }> = [{ key: 'attack', label: '攻击' }, { key: 'plunder', label: '掠夺' }, { key: 'scout', label: '侦察' }, { key: 'reinforce', label: '增援' }]
const currentTarget = computed(() => props.target ?? props.targetOptions.find((target) => target.targetId === selectedTargetId.value) ?? null)
const actionAllowed = computed(() => ({ attack: Boolean(currentTarget.value?.canAttack), plunder: Boolean(currentTarget.value?.canPlunder), scout: Boolean(currentTarget.value?.canScout), reinforce: Boolean(currentTarget.value?.canReinforce) }))
const actionSelectable = computed(() => props.targetOptions.length ? {
  attack: props.targetOptions.some((target) => target.canAttack),
  plunder: props.targetOptions.some((target) => target.canPlunder),
  scout: props.targetOptions.some((target) => target.canScout),
  reinforce: props.targetOptions.some((target) => target.canReinforce),
} : actionAllowed.value)
const availableTargets = computed(() => props.targetOptions.filter((target) => ({ attack: target.canAttack, plunder: target.canPlunder, scout: target.canScout, reinforce: target.canReinforce })[action.value]))
const scoutUnit = computed(() => props.units.find((unit) => ['zhanYingTanMa', 'flyingKite', 'secretAgent'].includes(unit.id)) ?? null)
const hasTroops = computed(() => action.value === 'scout' ? (scoutUnit.value?.owned ?? 0) > 0 : props.units.some((unit) => unit.dispatchable && normalizedAmount(unit) > 0))

/** 目标或初始动作变化时清空上一次尚未提交的表单。 */
watch(() => [props.target?.targetId ?? '', props.initialAction, props.targetOptions.map((target) => target.targetId).join('|')] as const, () => {
  action.value = props.initialAction
  selectedTargetId.value = props.target?.targetId ?? availableTargets.value[0]?.targetId ?? ''
  props.units.forEach((unit) => { amounts[unit.id] = 0 })
  selectedGeneralId.value = ''
  inputError.value = ''
}, { immediate: true })

/** 后端成功后清空已经派出的输入，保留成功消息供用户确认。 */
watch(() => props.resultVersion, () => {
  if (!props.succeeded) return
  props.units.forEach((unit) => { amounts[unit.id] = 0 })
  selectedGeneralId.value = ''
})

/** 判断武将是否被除主将槽外的真实任务占用。 */
function generalBusy(generalId: string) {
  return props.assignments.some((assignment) => assignment.generalId === generalId && assignment.id !== 'main' && assignment.slot !== 'main')
}

/** 显示包含被动特性加成后的武力，便于出征前核对将领实际数值。 */
function generalForce(general: GeneralState) { return general.effectiveStats?.force ?? general.stats?.force ?? 0 }

/** 将输入安全限制为当前后端兵力范围内整数。 */
function normalizedAmount(unit: RecruitmentUnitViewModel) {
  if (!unit.dispatchable) return 0
  const value = Math.trunc(Number(amounts[unit.id] ?? 0))
  return Number.isFinite(value) ? Math.max(0, Math.min(unit.owned, value)) : 0
}

/** 失焦时把非法或超量输入修正到合法范围。 */
function normalizeInput(unit: RecruitmentUnitViewModel) { amounts[unit.id] = normalizedAmount(unit) }

/** 点击官方现有数量时填入该兵种全部可用兵力。 */
function fillUnit(unit: RecruitmentUnitViewModel) { if (action.value !== 'scout' && unit.dispatchable) amounts[unit.id] = unit.owned }

/** 切换动作时只允许后端目标能力明确开放的命令。 */
function selectAction(next: WorldMapMarchAction) {
  if (!actionSelectable.value[next] || props.submitting) return
  action.value = next
  if (!availableTargets.value.some((target) => target.targetId === selectedTargetId.value)) selectedTargetId.value = availableTargets.value[0]?.targetId ?? ''
  inputError.value = ''
}

/** 校验表单后提交当前账号已验证存档、目标玩家和兵力。 */
function submitCommand() {
  const target = currentTarget.value
  if (!target?.playerId || props.submitting || !actionAllowed.value[action.value]) return
  if (!hasTroops.value) {
    inputError.value = action.value === 'scout' ? '本城没有可派出的侦察兵' : '请至少选择一个出征兵种'
    return
  }
  const troops = props.units.reduce<Record<string, number>>((result, unit) => {
    const amount = normalizedAmount(unit)
    if (amount > 0) result[unit.id] = amount
    return result
  }, {})
  emit('submit', action.value, target.playerId, troops, selectedGeneralId.value ? [selectedGeneralId.value] : [])
}
</script>

<template>
  <div class="march-command-mask" role="presentation" @click.self="emit('close')">
    <section class="march-command-dialog" role="dialog" aria-modal="true" aria-label="战争命令操作">
      <header class="march-command-title"><span>战争命令操作</span></header>
      <div class="march-command-body">
        <div class="march-command-modes">
          <label v-for="item in actions" :key="item.key" :class="{ disabled: !actionSelectable[item.key] }"><input type="radio" name="march-action" :checked="action === item.key" :disabled="!actionSelectable[item.key] || submitting" @change="selectAction(item.key)" />{{ item.label }}</label>
          <button type="button" disabled title="坐标收藏夹已在地图下方提供">坐标收藏夹</button>
        </div>
        <div class="march-command-destination">
          <label v-if="targetOptions.length">目的地：<select v-model="selectedTargetId" :disabled="submitting"><option v-for="item in availableTargets" :key="item.targetId" :value="item.targetId">{{ item.name }}（{{ item.x }}|{{ item.y }}）</option></select></label>
          <label v-else>目的地：<input :value="currentTarget?.name ?? ''" type="text" readonly /></label>
          <span>或坐标 X：<input :value="currentTarget?.x ?? ''" type="text" readonly /></span><span>Y：<input :value="currentTarget?.y ?? ''" type="text" readonly /></span>
          <label>派兵城池：<select disabled><option>{{ sourceName }}</option></select></label>
        </div>
        <div class="march-command-general">
          <img src="/assets/official/map/command/top_xz.gif" alt="武将" />
          <span v-if="!generals.length">当前没有可出征武将</span>
          <label v-for="general in generals" :key="general.id" :class="{ busy: generalBusy(general.id) }"><input v-model="selectedGeneralId" type="radio" name="march-general" :value="general.id" :disabled="submitting || action === 'scout' || generalBusy(general.id)" />{{ general.name }}(Lv {{ general.level }}，武力 {{ generalForce(general) }})</label>
          <label v-if="generals.length"><input v-model="selectedGeneralId" type="radio" name="march-general" value="" :disabled="submitting || action === 'scout'" />不出动</label>
        </div>
        <div v-if="action === 'scout'" class="march-scout-note">
          <img v-if="scoutUnit?.officialCode" :src="`/assets/official/images/${scoutUnit.officialCode}.gif`" :alt="scoutUnit.name" />
          <strong>{{ scoutUnit?.name ?? '侦察兵' }}</strong><span>后端将自动派出全部可用侦察兵（{{ scoutUnit?.owned ?? 0 }}）</span>
        </div>
        <div v-else class="march-unit-grid">
          <label v-for="unit in units" :key="unit.id" :title="unit.name">
            <img v-if="unit.officialCode" :src="`/assets/official/images/${unit.officialCode}.gif`" :alt="unit.name" /><span v-else class="march-unit-fallback">兵</span>
            <input v-model.number="amounts[unit.id]" type="number" min="0" :max="unit.owned" :disabled="submitting || unit.owned === 0 || !unit.dispatchable" :aria-label="`${unit.name}出征数量`" @blur="normalizeInput(unit)" />
            <button type="button" :disabled="submitting || unit.owned === 0 || !unit.dispatchable" :title="unit.dispatchable ? `全部派出${unit.name}` : `${unit.name}不是战斗单位`" @click="fillUnit(unit)">({{ unit.owned }})</button>
          </label>
        </div>
        <p class="march-command-difference">Hero3 当前接口不支持指定攻城器械的建筑目标；最终行军时间、武将占用和兵力扣除以后端为准。</p>
        <p v-if="targetOptions.length && !availableTargets.length" class="march-command-message error">当前世界地图中没有可执行该命令的玩家城池。</p>
        <p v-if="inputError || message" class="march-command-message" :class="{ success: succeeded && !inputError, error: inputError || (!succeeded && message) }" aria-live="polite">{{ inputError || message }}</p>
        <footer><button type="button" :disabled="submitting || !currentTarget || !actionAllowed[action] || !hasTroops" @click="submitCommand"><img src="/assets/official/map/command/n_qd.gif" alt="确定" /></button><button type="button" :disabled="submitting" @click="emit('close')"><img src="/assets/official/map/command/n_gb.gif" alt="关闭" /></button><span v-if="submitting">提交命令中…</span></footer>
      </div>
    </section>
  </div>
</template>
