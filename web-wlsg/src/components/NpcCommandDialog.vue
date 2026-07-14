<!-- NPC 战争命令弹窗：复用地图官方弹窗结构并提交即时进攻、掠夺和侦查。 -->
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { npcFactionLabel, npcTierLabel } from '../data/npcDirectory'
import type { RecruitmentUnitViewModel } from '../game/recruitmentAdapter'
import type { GeneralAssignmentState, GeneralState, NpcCityState, NpcCommandAction } from '../game/types'

const props = defineProps<{ city: NpcCityState; sourceName: string; units: RecruitmentUnitViewModel[]; generals: GeneralState[]; assignments: GeneralAssignmentState[]; submitting: boolean; message: string; succeeded: boolean; resultVersion: number }>()
const emit = defineEmits<{ close: []; submit: [action: NpcCommandAction, npcId: string, troops: Record<string, number>, generalIds: string[]] }>()
const action = ref<NpcCommandAction>('attack')
const amounts = reactive<Record<string, number>>({})
const selectedGeneralId = ref('')
const inputError = ref('')
const actions: Array<{ key: NpcCommandAction; label: string }> = [{ key: 'attack', label: '攻击' }, { key: 'plunder', label: '掠夺' }, { key: 'scout', label: '侦察' }]
const scoutUnit = computed(() => props.units.find((unit) => ['zhanYingTanMa', 'flyingKite', 'secretAgent'].includes(unit.id)) ?? null)
const hasTroops = computed(() => action.value === 'scout' ? (scoutUnit.value?.owned ?? 0) > 0 : props.units.some((unit) => unit.dispatchable && normalizedAmount(unit) > 0))

/** 切换目标时清空上一次未提交的兵力和武将选择。 */
watch(() => props.city.id, () => {
  action.value = 'attack'
  props.units.forEach((unit) => { amounts[unit.id] = 0 })
  selectedGeneralId.value = ''
  inputError.value = ''
}, { immediate: true })

/** 操作成功后清空已结算的输入，保留结果消息。 */
watch(() => props.resultVersion, () => {
  if (!props.succeeded) return
  props.units.forEach((unit) => { amounts[unit.id] = 0 })
  selectedGeneralId.value = ''
})

/** 判断武将是否被除主将槽外的真实任务占用。 */
function generalBusy(generalId: string) { return props.assignments.some((assignment) => assignment.generalId === generalId && assignment.id !== 'main' && assignment.slot !== 'main') }

/** 将兵力输入限制为当前后端返回的可用整数范围。 */
function normalizedAmount(unit: RecruitmentUnitViewModel) {
  if (!unit.dispatchable) return 0
  const value = Math.trunc(Number(amounts[unit.id] ?? 0))
  return Number.isFinite(value) ? Math.max(0, Math.min(unit.owned, value)) : 0
}

/** 失焦时修正非法或超量输入。 */
function normalizeInput(unit: RecruitmentUnitViewModel) { amounts[unit.id] = normalizedAmount(unit) }

/** 点击现有数量时填入该兵种全部可用兵力。 */
function fillUnit(unit: RecruitmentUnitViewModel) { if (action.value !== 'scout' && unit.dispatchable) amounts[unit.id] = unit.owned }

/** 切换 NPC 即时命令并清除旧校验信息。 */
function selectAction(next: NpcCommandAction) {
  if (props.submitting) return
  action.value = next
  inputError.value = ''
  if (next === 'scout') selectedGeneralId.value = ''
}

/** 校验输入后提交当前 NPC 的即时结算命令。 */
function submitCommand() {
  if (props.submitting) return
  if (!hasTroops.value) {
    inputError.value = action.value === 'scout' ? '本城没有可派出的侦察兵' : '请至少选择一个出征兵种'
    return
  }
  const troops = props.units.reduce<Record<string, number>>((result, unit) => {
    const amount = normalizedAmount(unit)
    if (amount > 0) result[unit.id] = amount
    return result
  }, {})
  emit('submit', action.value, props.city.id, troops, selectedGeneralId.value ? [selectedGeneralId.value] : [])
}
</script>

<template>
  <div class="march-command-mask" role="presentation" @click.self="emit('close')">
    <section class="march-command-dialog" role="dialog" aria-modal="true" aria-label="NPC 战争命令操作">
      <header class="march-command-title"><span>战争命令操作</span></header>
      <div class="march-command-body">
        <div class="march-command-modes npc-command-modes">
          <label v-for="item in actions" :key="item.key"><input type="radio" name="npc-action" :checked="action === item.key" :disabled="submitting" @change="selectAction(item.key)" />{{ item.label }}</label>
          <span>NPC 战斗即时结算，不产生行军时间</span>
        </div>
        <div class="march-command-destination npc-command-destination">
          <label>目的地：<input :value="city.name" type="text" readonly /></label>
          <span>规模：<input :value="npcTierLabel(city.tier)" type="text" readonly /></span>
          <span>阵营：<input :value="npcFactionLabel(city.faction)" type="text" readonly /></span>
          <label>派兵城池：<select disabled><option>{{ sourceName }}</option></select></label>
        </div>
        <div class="march-command-general">
          <img src="/assets/official/map/command/top_xz.gif" alt="武将" />
          <span v-if="!generals.length">当前没有可出征武将</span>
          <label v-for="general in generals" :key="general.id" :class="{ busy: generalBusy(general.id) }"><input v-model="selectedGeneralId" type="radio" name="npc-general" :value="general.id" :disabled="submitting || action === 'scout' || generalBusy(general.id)" />{{ general.name }}(Lv {{ general.level }})</label>
          <label v-if="generals.length"><input v-model="selectedGeneralId" type="radio" name="npc-general" value="" :disabled="submitting || action === 'scout'" />不出动</label>
        </div>
        <div v-if="action === 'scout'" class="march-scout-note">
          <img v-if="scoutUnit?.officialCode" :src="`/assets/official/images/${scoutUnit.officialCode}.gif`" :alt="scoutUnit.name" />
          <strong>{{ scoutUnit?.name ?? '侦察兵' }}</strong><span>后端将自动派出全部可用侦察兵（{{ scoutUnit?.owned ?? 0 }}）并立即结算</span>
        </div>
        <div v-else class="march-unit-grid">
          <label v-for="unit in units" :key="unit.id" :title="unit.name">
            <img v-if="unit.officialCode" :src="`/assets/official/images/${unit.officialCode}.gif`" :alt="unit.name" /><span v-else class="march-unit-fallback">兵</span>
            <input v-model.number="amounts[unit.id]" type="number" min="0" :max="unit.owned" :disabled="submitting || unit.owned === 0 || !unit.dispatchable" :aria-label="`${unit.name}出征数量`" @blur="normalizeInput(unit)" />
            <button type="button" :disabled="submitting || unit.owned === 0 || !unit.dispatchable" :title="unit.dispatchable ? `全部派出${unit.name}` : `${unit.name}不是战斗单位`" @click="fillUnit(unit)">({{ unit.owned }})</button>
          </label>
        </div>
        <p class="march-command-difference">进攻、掠夺与侦查均由 Hero3 后端立即结算；兵力损失、资源收益、武将经验和军情战报以后端结果为准。</p>
        <p v-if="inputError || message" class="march-command-message" :class="{ success: succeeded && !inputError, error: inputError || (!succeeded && message) }" aria-live="polite">{{ inputError || message }}</p>
        <footer><button type="button" :disabled="submitting || !hasTroops" @click="submitCommand"><img src="/assets/official/map/command/n_qd.gif" alt="确定" /></button><button type="button" :disabled="submitting" @click="emit('close')"><img src="/assets/official/map/command/n_gb.gif" alt="关闭" /></button><span v-if="submitting">即时结算中…</span></footer>
      </div>
    </section>
  </div>
</template>
