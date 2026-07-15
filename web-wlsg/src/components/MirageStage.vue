<!-- 万象幻境页：接入六合博戏、天机轮转和真实奖励库存兑换。 -->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { RecruitmentUnitViewModel } from '../game/recruitmentAdapter'
import type { MirageStateStore } from '../mirage/stateService'
import type { MirageGameType, MirageRecord } from '../mirage/types'

const props = defineProps<{ mirage: MirageStateStore; units: RecruitmentUnitViewModel[] }>()
const emit = defineEmits<{ load: []; gamble: [unitType: string, amount: number, betId: string, exactNumber: number]; spin: [unitType: string, amount: number]; redeem: [recordId: string, amount: number]; redeemAll: [gameType: MirageGameType] }>()
const activeGame = ref<MirageGameType>('slot')
const selectedUnit = ref('')
const betAmount = ref(1000)
const betId = ref('big')
const exactNumber = ref(10)
const combatUnits = computed(() => props.units.filter((unit) => unit.dispatchable && unit.owned > 0))
const selectedOwned = computed(() => combatUnits.value.find((unit) => unit.id === selectedUnit.value)?.owned ?? 0)
const records = computed(() => (props.mirage.summary?.records ?? []).filter((record) => record.gameType === activeGame.value))
const inventory = computed(() => records.value.filter((record) => record.remainingAmount > 0))
const totalInventory = computed(() => inventory.value.reduce((sum, record) => sum + record.remainingAmount, 0))
const slotGrid = computed(() => props.mirage.slotResult?.grid?.length ? props.mirage.slotResult.grid : [['bronze_charm', 'jade_seal', 'tiger_tally'], ['silver_charm', 'gold_charm', 'scatter'], ['bonus', 'bronze_charm', 'jade_seal']])

/** 兵力变化时保持一个有效的默认押注兵种。 */
watch(combatUnits, (units) => {
  if (!units.some((unit) => unit.id === selectedUnit.value)) selectedUnit.value = units[0]?.id ?? ''
}, { immediate: true })

/** 把押注数量限制到后端要求和当前真实兵力范围。 */
function normalizeBet() {
  const minimum = activeGame.value === 'slot' ? 1000 : 1
  betAmount.value = Math.max(minimum, Math.min(selectedOwned.value, Math.trunc(Number(betAmount.value) || minimum)))
}

/** 提交当前子玩法的后端权威结算。 */
function play() {
  normalizeBet()
  exactNumber.value = Math.max(3, Math.min(18, Math.trunc(Number(exactNumber.value) || 10)))
  if (!selectedUnit.value || betAmount.value <= 0 || betAmount.value > selectedOwned.value) return
  if (activeGame.value === 'slot') emit('spin', selectedUnit.value, betAmount.value)
  else emit('gamble', selectedUnit.value, betAmount.value, betId.value, betId.value === 'exact' ? exactNumber.value : 0)
}

/** 返回天机图案的传统符号和名称。 */
function symbolMeta(symbol: string) {
  return ({ bronze_charm: ['铜', '青铜符'], silver_charm: ['银', '白银符'], gold_charm: ['金', '赤金符'], tiger_tally: ['虎', '虎符'], jade_seal: ['玺', '玉玺'], scatter: ['令', '天机令'], bonus: ['匣', '宝匣'] } as Record<string, [string, string]>)[symbol] ?? ['玄', symbol]
}

/** 返回稀有度中文名。 */
function rarityLabel(rarity: string) { return ({ common: '普通', rare: '稀有', epic: '珍奇', legendary: '传说' } as Record<string, string>)[rarity] ?? rarity }

/** 紧凑显示记录时间。 */
function timeLabel(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false }) }

/** 兑换当前记录的全部剩余库存。 */
function redeemRecord(record: MirageRecord) { if (record.remainingAmount > 0) emit('redeem', record.id, record.remainingAmount) }
</script>

<template>
  <section class="adventure-stage mirage-stage" aria-label="万象幻境">
    <div class="adventure-banner mirage-banner"><div><small>MYSTIC REALM</small><h2>万象幻境</h2><p>以真实兵力为筹，观骰象、转天机。所有结果由服务器生成，所得兵力先存入幻境宝库，确认兑换后才进入军队或编外驻防。</p></div><div class="mirage-orb"><i></i><b>万象</b></div></div>
    <nav class="mirage-game-tabs"><button type="button" :class="{ active: activeGame === 'slot' }" @click="activeGame = 'slot'">天机轮转</button><button type="button" :class="{ active: activeGame === 'gambling' }" @click="activeGame = 'gambling'">六合博戏</button><span>幻境库存 {{ totalInventory.toLocaleString('zh-CN') }}</span></nav>
    <div v-if="mirage.phase === 'loading' && !mirage.summary" class="adventure-state"><span class="loading-mark"></span><strong>正在凝聚幻境…</strong></div>
    <div v-else-if="mirage.phase === 'error'" class="adventure-state error"><strong>万象幻境读取失败</strong><p>{{ mirage.error }}</p><button type="button" @click="emit('load')">重新读取</button></div>
    <template v-else>
      <section class="adventure-panel mirage-play-panel">
        <header><strong><i></i>{{ activeGame === 'slot' ? '天机轮转 · 五线同观' : '六合博戏 · 三骰定数' }}</strong><span>{{ activeGame === 'slot' ? '最低押注 1,000' : '大小、单双或指定点数' }}</span></header>
        <div v-if="activeGame === 'slot'" class="slot-board">
          <div v-for="(row, rowIndex) in slotGrid" :key="rowIndex" class="slot-row"><div v-for="(symbol, columnIndex) in row" :key="`${rowIndex}-${columnIndex}`" :class="`symbol-${symbol}`" :title="symbolMeta(symbol)[1]"><b>{{ symbolMeta(symbol)[0] }}</b><span>{{ symbolMeta(symbol)[1] }}</span></div></div>
          <p v-if="mirage.slotResult">{{ mirage.slotResult.won ? `本局赢得 ${mirage.slotResult.winAmount.toLocaleString('zh-CN')} ${mirage.slotResult.betUnit}` : '本局未中奖' }}<small v-if="mirage.slotResult.winningLines.length">中奖线 {{ mirage.slotResult.winningLines.length }} 条 · 免费转动 {{ mirage.slotResult.freeSpins.length }} 次</small></p>
        </div>
        <div v-else class="dice-board"><div><b v-for="(dice, index) in mirage.gamblingResult?.diceValues ?? [1, 4, 6]" :key="index">{{ dice }}</b></div><p v-if="mirage.gamblingResult">三骰合计 <strong>{{ mirage.gamblingResult.diceTotal }}</strong> · {{ mirage.gamblingResult.won ? `命中 ${mirage.gamblingResult.betLabel}，赔率 ×${mirage.gamblingResult.multiplier}` : `${mirage.gamblingResult.betLabel}未命中` }}</p><p v-else>三骰入盅，六合定数</p></div>
        <div class="mirage-controls"><label>押注兵种 <select v-model="selectedUnit" :disabled="mirage.operating"><option v-for="unit in combatUnits" :key="unit.id" :value="unit.id">{{ unit.name }}（{{ unit.owned.toLocaleString('zh-CN') }}）</option></select></label><label>押注数量 <input v-model.number="betAmount" type="number" :min="activeGame === 'slot' ? 1000 : 1" :max="selectedOwned" :disabled="mirage.operating" @change="normalizeBet" /></label><template v-if="activeGame === 'gambling'"><label>押注 <select v-model="betId" :disabled="mirage.operating"><option value="big">大（11-17）</option><option value="small">小（4-10）</option><option value="odd">单</option><option value="even">双</option><option value="triple">豹子</option><option value="exact">指定点数</option></select></label><label v-if="betId === 'exact'">点数 <input v-model.number="exactNumber" type="number" min="3" max="18" /></label></template><button type="button" :disabled="mirage.operating || !selectedUnit || betAmount > selectedOwned || betAmount < (activeGame === 'slot' ? 1000 : 1)" @click="play">{{ mirage.operating ? '推演中…' : activeGame === 'slot' ? '转动天机' : '揭开骰盅' }}</button></div>
      </section>

      <section class="adventure-panel mirage-inventory-panel"><header><strong><i></i>幻境宝库</strong><button type="button" :disabled="mirage.operating || !inventory.length" @click="emit('redeemAll', activeGame)">一键兑换本页库存</button></header><table><thead><tr><th>所得</th><th>品质</th><th>原始奖励</th><th>待兑换</th><th>时间</th><th>操作</th></tr></thead><tbody><tr v-if="!records.length"><td colspan="6">当前玩法暂无记录</td></tr><tr v-for="record in records.slice(0, 12)" :key="record.id" :class="`rarity-${record.rarity}`"><td>{{ record.resultName }}</td><td>{{ rarityLabel(record.rarity) }}</td><td>{{ record.rewardUnit || '无' }} {{ record.rewardAmount.toLocaleString('zh-CN') }}</td><td>{{ record.remainingAmount.toLocaleString('zh-CN') }}</td><td>{{ timeLabel(record.createdAt) }}</td><td><button type="button" :disabled="mirage.operating || record.remainingAmount <= 0" @click="redeemRecord(record)">{{ record.remainingAmount > 0 ? '兑换' : '已领取' }}</button></td></tr></tbody></table></section>
    </template>
    <p v-if="mirage.actionMessage" class="adventure-feedback" :class="mirage.actionSucceeded ? 'success' : 'error'">{{ mirage.actionMessage }}</p>
  </section>
</template>
