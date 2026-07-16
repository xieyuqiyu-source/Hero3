<!-- 仓库扩容弹窗：沿用原站深色铜边弹窗，并展示后端四倍率、四时长实时价格。 -->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{ prices: Record<string, number>; cityGold: number; currentMultiplier?: number; currentEnd?: string; loading?: boolean; submitting?: boolean; message?: string; succeeded?: boolean }>(), { currentMultiplier: 0, currentEnd: '', loading: false, submitting: false, message: '', succeeded: false })
const emit = defineEmits<{ close: []; retry: []; purchase: [multiplier: number, hours: number] }>()
const multipliers = [2, 4, 8, 16]
const durations = [1, 6, 12, 24]
const selectedMultiplier = ref(2)
const selectedHours = ref(1)
const selectedCost = computed(() => props.prices[`${selectedMultiplier.value}x_${selectedHours.value}h`] ?? null)
const cannotAfford = computed(() => selectedCost.value !== null && selectedCost.value > props.cityGold)
const currentEndLabel = computed(() => {
  if (!props.currentMultiplier || !props.currentEnd) return '当前没有容量扩容效果'
  const end = new Date(props.currentEnd)
  const label = Number.isNaN(end.getTime()) ? props.currentEnd : end.toLocaleString('zh-CN', { hour12: false })
  return `当前容量 ×${props.currentMultiplier}，到期时间 ${label}`
})

/** 打开或切换存档时优先选中当前有效倍率，时长回到最小档。 */
watch(() => [props.currentMultiplier, props.currentEnd], () => {
  selectedMultiplier.value = multipliers.includes(props.currentMultiplier) ? props.currentMultiplier : 2
  selectedHours.value = 1
}, { immediate: true })

/** 提交当前价格表中存在且余额充足的扩容组合。 */
function submit() {
  if (props.loading || props.submitting || selectedCost.value === null || cannotAfford.value) return
  emit('purchase', selectedMultiplier.value, selectedHours.value)
}
</script>

<template>
  <div class="capacity-boost-mask" role="presentation" @click.self="emit('close')">
    <section class="capacity-boost-dialog" role="dialog" aria-modal="true" aria-labelledby="capacity-boost-title">
      <header><strong id="capacity-boost-title">城池仓库扩容</strong><button type="button" aria-label="关闭扩容弹窗" @click="emit('close')">×</button></header>
      <div class="capacity-boost-body">
        <p class="capacity-boost-current">{{ currentEndLabel }}</p>
        <p class="capacity-boost-tip">选择容量倍率与持续时间。购买后仓库容量立即生效，同倍率再次购买会续时，更换倍率则按本次选择重新计算。</p>
        <fieldset><legend>容量倍率</legend><div class="capacity-option-grid multiplier"><button v-for="multiplier in multipliers" :key="multiplier" type="button" :class="{ selected: selectedMultiplier === multiplier }" @click="selectedMultiplier = multiplier">×{{ multiplier }}</button></div></fieldset>
        <fieldset><legend>持续时间</legend><div class="capacity-option-grid duration"><button v-for="hours in durations" :key="hours" type="button" :class="{ selected: selectedHours === hours }" @click="selectedHours = hours">{{ hours }} 小时</button></div></fieldset>
        <div v-if="loading" class="capacity-boost-loading">正在读取扩容价格…</div>
        <div v-else-if="selectedCost === null" class="capacity-boost-loading error">价格读取失败，请重试。</div>
        <div v-else class="capacity-boost-price"><span>本次消耗</span><strong>{{ selectedCost.toLocaleString('zh-CN') }} 城金</strong><small>当前城金：{{ cityGold.toLocaleString('zh-CN') }}</small></div>
        <p v-if="message" class="capacity-boost-message" :class="{ success: succeeded, error: !succeeded }" role="status">{{ message }}</p>
      </div>
      <footer><button v-if="!loading && selectedCost === null" type="button" @click="emit('retry')">重新读取</button><button type="button" :disabled="loading || submitting || selectedCost === null || cannotAfford" @click="submit">{{ submitting ? '扩容中…' : cannotAfford ? '城金不足' : '确认扩容' }}</button><button type="button" @click="emit('close')">关闭</button></footer>
    </section>
  </div>
</template>
