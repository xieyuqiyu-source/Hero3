<!-- 轮回绝境副本页：展示真实层级、十八波攻防、随机加成与出兵结算。 -->
<script setup lang="ts">
import { computed, ref, toRef, watch } from 'vue'
import type { GeneralAssignmentState, GeneralState } from '../game/types'
import type { RecruitmentUnitViewModel } from '../game/recruitmentAdapter'
import { formatRemaining, useServerClock } from '../game/useServerClock'
import { activeWave, type DungeonStateStore } from '../dungeon/stateService'
import { dungeonGeneralBonusText, preferredDungeonGeneralId } from '../dungeon/generalPresentation'
import type { DungeonReward } from '../dungeon/types'

const props = defineProps<{ dungeon: DungeonStateStore; playerId: string; accountGold: number; units: RecruitmentUnitViewModel[]; generals: GeneralState[]; assignments: GeneralAssignmentState[]; serverTime: string; receivedAt: number | null }>()
const emit = defineEmits<{ load: []; start: [level: number]; fight: [waveId: string, troops: Record<string, number>, generalIds: string[]]; resetBonus: [waveId: string]; settle: []; exit: [] }>()
const selectedLevel = ref(1)
const troopAmounts = ref<Record<string, number>>({})
const selectedGeneralId = ref('')
const availableLevels = computed(() => (props.dungeon.config?.levels ?? []).filter((level) => level.enabled))
const wave = computed(() => activeWave(props.dungeon.run))
const availableGenerals = computed(() => {
  const busy = new Set(props.assignments.filter((assignment) => assignment.id !== 'main' && assignment.slot !== 'main').map((assignment) => assignment.generalId))
  return props.generals.filter((general) => !busy.has(general.id))
})
const selectedGeneral = computed(() => availableGenerals.value.find((general) => general.id === selectedGeneralId.value) ?? null)
const selectedGeneralBonus = computed(() => dungeonGeneralBonusText(selectedGeneral.value))
/** 返回最近一波随军将领名称，避免战斗结算后选择器切换到下一波主将。 */
const lastExpGeneralName = computed(() => props.generals.find((general) => general.id === props.dungeon.lastGeneralId)?.name ?? '随军将领')
/** 只在本波确实升级时展示升级前后的等级。 */
const lastExpLevelText = computed(() => {
  const before = props.dungeon.lastGeneralLevelBefore
  const after = props.dungeon.lastGeneralLevelAfter
  if (before === null || after === null || before === after) return ''
  return ` · 等级 ${before} → ${after}`
})
const selectedTotal = computed(() => Object.values(troopAmounts.value).reduce((sum, amount) => sum + Math.max(0, Number(amount) || 0), 0))
const { remainingSeconds } = useServerClock(toRef(props, 'serverTime'), toRef(props, 'receivedAt'))
const expiresIn = computed(() => props.dungeon.run ? Math.max(0, remainingSeconds(props.dungeon.run.expiresAt)) : 0)
const enemyTotal = computed(() => Object.values(wave.value?.enemyRemaining ?? {}).reduce((sum, amount) => sum + amount, 0))
const progress = computed(() => props.dungeon.run ? Math.max(0, Math.min(100, (props.dungeon.run.currentWave - 1) / 18 * 100)) : 0)

/** 配置到达后选中第一个开放难度。 */
watch(availableLevels, (levels) => {
  if (levels.length && !levels.some((level) => level.level === selectedLevel.value)) selectedLevel.value = levels[0].level
}, { immediate: true })

/** 每波开始时清除上波兵力，并默认携带当前可用主将。 */
watch(() => wave.value?.id, () => {
  troopAmounts.value = {}
  selectedGeneralId.value = preferredDungeonGeneralId(availableGenerals.value, props.assignments)
}, { immediate: true })

/** 武将占用变化后确保默认选择仍然真实可用。 */
watch(availableGenerals, (generals) => {
  if (!generals.some((general) => general.id === selectedGeneralId.value)) selectedGeneralId.value = preferredDungeonGeneralId(generals, props.assignments)
}, { immediate: true })

/** 把输入限制在当前真实可用兵力范围内。 */
function normalizeAmount(unit: RecruitmentUnitViewModel) {
  troopAmounts.value[unit.id] = Math.max(0, Math.min(unit.owned, Math.trunc(Number(troopAmounts.value[unit.id]) || 0)))
}

/** 一键填入当前城池全部可用战斗兵力。 */
function fillTroops() {
  const next: Record<string, number> = {}
  for (const unit of props.units.filter((item) => item.dispatchable && item.owned > 0)) {
    next[unit.id] = unit.owned
  }
  troopAmounts.value = next
}

/** 提交当前波兵力和至多一名可用武将。 */
function submitWave() {
  if (!wave.value) return
  emit('fight', wave.value.id, { ...troopAmounts.value }, selectedGeneralId.value ? [selectedGeneralId.value] : [])
}

/** 将奖励类型转换为中文紧凑文本。 */
function rewardText(rewards: DungeonReward[]) {
  return rewards.length ? rewards.map((reward) => `${rewardName(reward)}×${reward.amount.toLocaleString('zh-CN')}`).join('、') : '暂无'
}

/** 提供常见副本奖励的中文名称并保留未知 ID。 */
function rewardName(reward: DungeonReward) {
  return ({ general_exp_small: '小型武将经验', general_exp_medium: '中型武将经验', general_exp_large: '大型武将经验', token_reincarnation: '轮回令' } as Record<string, string>)[reward.id] ?? reward.id
}

/** 返回副本和波次状态中文名称。 */
function statusLabel(status: string) { return ({ running: '进行中', completed: '已通关', failed: '挑战失败', expired: '已经超时', rewarded: '奖励已结算', active: '当前波', cleared: '已通过', locked: '未开启' } as Record<string, string>)[status] ?? status }
</script>

<template>
  <section class="adventure-stage dungeon-stage" aria-label="轮回绝境副本">
    <div class="adventure-banner dungeon-banner">
      <div><small>HERO3 DUNGEON</small><h2>轮回绝境</h2><p>十八波攻防轮转，随机军势加成，阵亡兵力真实扣除。越过的波次越多，最终累计奖励越丰厚。</p></div>
      <dl><div><dt>规则</dt><dd>奇数进攻 · 偶数防守</dd></div><div><dt>时限</dt><dd>开启后 2 小时</dd></div><div><dt>结算</dt><dd>通关、失败或超时</dd></div></dl>
    </div>

    <div v-if="dungeon.phase === 'loading' && !dungeon.config" class="adventure-state"><span class="loading-mark"></span><strong>正在开启轮回之门…</strong></div>
    <div v-else-if="dungeon.phase === 'error'" class="adventure-state error"><strong>轮回绝境读取失败</strong><p>{{ dungeon.error }}</p><button type="button" @click="emit('load')">重新读取</button></div>

    <template v-else-if="!dungeon.run">
      <section class="adventure-panel dungeon-level-panel">
        <header><strong><i></i>选择轮回层级</strong><span>当前金币：{{ accountGold.toLocaleString('zh-CN') }}</span></header>
        <div class="dungeon-level-grid">
          <button v-for="level in availableLevels" :key="level.level" type="button" :class="{ selected: selectedLevel === level.level }" :disabled="dungeon.operating" @click="selectedLevel = level.level">
            <b>第{{ level.level }}境</b><strong>{{ level.name }}</strong><span>敌军基准 {{ level.enemyTroopBase.toLocaleString('zh-CN') }}</span><span>出战兵力 不限</span><em>经验上限 {{ level.rewardExpCap.toLocaleString('zh-CN') }}</em>
          </button>
        </div>
        <p class="adventure-warning">开启副本不会立即扣兵；每次出战会真实结算兵损，已阵亡兵力不会返还。</p>
        <div class="adventure-actions"><button type="button" :disabled="dungeon.operating || !availableLevels.length" @click="emit('start', selectedLevel)">{{ dungeon.operating ? '开启中…' : '开启轮回绝境' }}</button></div>
      </section>
    </template>

    <template v-else>
      <section class="adventure-panel dungeon-progress-panel">
        <header><strong><i></i>{{ dungeon.run.levelName }}</strong><span :class="`status-${dungeon.run.status}`">{{ statusLabel(dungeon.run.status) }} · {{ dungeon.run.status === 'running' ? formatRemaining(expiresIn) : `第 ${dungeon.run.currentWave}/18 波` }}</span></header>
        <div class="dungeon-wave-track"><i :style="{ width: `${progress}%` }"></i><button v-for="item in dungeon.run.waves" :key="item.id" type="button" disabled :class="[item.status, item.waveType]" :title="`第 ${item.waveIndex} 波 · ${item.waveType === 'attack' ? '进攻' : '防守'} · ${statusLabel(item.status)}`">{{ item.waveIndex }}</button></div>
      </section>

      <section v-if="wave && dungeon.run.status === 'running'" class="adventure-panel dungeon-wave-panel">
        <header><strong><i></i>第 {{ wave.waveIndex }} 波 · {{ wave.waveType === 'attack' ? '破阵进攻' : '据城迎敌' }}</strong><span>敌军 {{ enemyTotal.toLocaleString('zh-CN') }} / 出战兵力不限</span></header>
        <div class="dungeon-wave-summary">
          <div><h3>敌军阵势</h3><p><b>{{ ({ wei: '魏', shu: '蜀', wu: '吴' } as Record<string, string>)[wave.enemyFaction] ?? wave.enemyFaction }}</b><span v-for="(amount, unit) in wave.enemyRemaining" :key="unit">{{ unit }} {{ amount.toLocaleString('zh-CN') }}</span></p></div>
          <div><h3>天命加成</h3><p class="ally">我方：{{ wave.allyBonus.label }}</p><p class="enemy">敌方：{{ wave.enemyBonus.label }}</p><button type="button" :disabled="dungeon.operating || accountGold < (dungeon.config?.bonusResetGoldCost ?? 0)" @click="emit('resetBonus', wave.id)">换签（{{ dungeon.config?.bonusResetGoldCost ?? 0 }} 金币）</button></div>
          <div><h3>本波奖励</h3><p>{{ rewardText(wave.rewardPreview) }}</p></div>
        </div>
        <table class="dungeon-troop-table"><thead><tr><th>兵种</th><th>可用</th><th>本波出战</th></tr></thead><tbody><tr v-for="unit in units.filter((item) => item.dispatchable)" :key="unit.id"><td><img v-if="unit.officialCode" :src="`/assets/official/images/${unit.officialCode}.gif`" :alt="unit.name" />{{ unit.name }}</td><td>{{ unit.owned.toLocaleString('zh-CN') }}</td><td><input v-model.number="troopAmounts[unit.id]" type="number" min="0" :max="unit.owned" :disabled="dungeon.operating || unit.owned <= 0" @change="normalizeAmount(unit)" /></td></tr></tbody></table>
        <div class="dungeon-command-row"><button type="button" :disabled="dungeon.operating" @click="fillTroops">全军出战</button><label>随军武将 <select v-model="selectedGeneralId" :disabled="dungeon.operating"><option value="">不出动</option><option v-for="general in availableGenerals" :key="general.id" :value="general.id">{{ general.name }}（等级{{ general.level }}）</option></select></label><b>已选 {{ selectedTotal.toLocaleString('zh-CN') }}</b><button class="primary" type="button" :disabled="dungeon.operating || selectedTotal <= 0" @click="submitWave">{{ dungeon.operating ? '结算中…' : wave.waveType === 'attack' ? '发起进攻' : '整军迎敌' }}</button></div>
        <p class="dungeon-general-bonus" :class="{ inactive: !selectedGeneral }"><strong>{{ selectedGeneral ? `${selectedGeneral.name}随军加成` : '未携将' }}</strong><span>{{ selectedGeneralBonus }}</span></p>
      </section>

      <section class="adventure-panel dungeon-reward-panel"><header><strong><i></i>本轮累计奖励</strong></header><p>{{ rewardText(dungeon.run.pendingRewards) }}</p><p v-if="dungeon.lastGeneralId" class="dungeon-general-exp"><strong>本波将领经验</strong><span>{{ lastExpGeneralName }} <b>+{{ dungeon.lastGeneralExpGained.toLocaleString('zh-CN') }}</b>{{ lastExpLevelText }}</span></p><div v-if="dungeon.run.status !== 'running' && !dungeon.run.rewardGrantedAt" class="adventure-actions"><button type="button" :disabled="dungeon.operating" @click="emit('settle')">结算并领取</button></div><div v-else-if="dungeon.run.rewardGrantedAt" class="adventure-actions"><span class="success">奖励已发放</span><button type="button" :disabled="dungeon.operating" @click="emit('exit')">退出副本</button></div></section>
    </template>
    <p v-if="dungeon.actionMessage" class="adventure-feedback" :class="dungeon.actionSucceeded ? 'success' : 'error'">{{ dungeon.actionMessage }}<span v-if="dungeon.lastReportId"> · 战报 {{ dungeon.lastReportId.slice(-8) }}</span></p>
  </section>
</template>
