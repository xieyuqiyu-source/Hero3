<!-- 使用后端真实数量、容量和产量渲染城池资源条及一键爆仓入口。 -->
<script setup lang="ts">
import { computed } from 'vue'
import type { ResourceViewModel } from '../game/types'
const props = withDefaults(defineProps<{ resources: ResourceViewModel[]; cityGold?: number; filling?: boolean; actionMessage?: string; actionSucceeded?: boolean }>(), { cityGold: 0, filling: false, actionMessage: '', actionSucceeded: false })
const emit = defineEmits<{ fill: [] }>()
const resourceGap = computed(() => props.resources.reduce((total, resource) => total + Math.max(0, resource.capacity - resource.amount), 0))
const estimatedCost = computed(() => resourceGap.value ? Math.max(1, Math.ceil(resourceGap.value / 3000)) : 0)
const cannotAfford = computed(() => estimatedCost.value > props.cityGold)
const fillDisabled = computed(() => props.filling || resourceGap.value === 0 || cannotAfford.value)
const fillTitle = computed(() => {
  if (props.filling) return '正在补满本城资源'
  if (props.actionMessage) return props.actionMessage
  if (!resourceGap.value) return '本城资源已经满仓'
  if (cannotAfford.value) return `预计需要 ${estimatedCost.value} 城金，当前城金不足`
  return `一键补满全部资源，预计消耗 ${estimatedCost.value} 城金，最终以后端结算为准`
})
const fillLabel = computed(() => props.filling ? '爆仓中…' : props.actionMessage ? (props.actionSucceeded ? '爆仓成功' : '爆仓失败') : '一键爆仓')

/** 使用中文千分位格式化真实数值。 */
function formatNumber(value: number) { return Number(value || 0).toLocaleString('zh-CN') }
</script>

<template>
  <section class="resource-bar" aria-label="本城真实资源">
    <strong>本城资源:</strong>
    <div v-for="resource in resources" :key="resource.key" class="resource-item" :title="`${resource.name}：每小时 ${formatNumber(resource.productionPerHour)}`">
      <img :src="`/assets/official/images/${resource.icon}`" alt="" />
      <span>{{ formatNumber(resource.amount) }}/{{ formatNumber(resource.capacity) }}</span>
    </div>
    <button type="button" class="resource-fill-button" :class="{ success: actionSucceeded && actionMessage, error: !actionSucceeded && actionMessage }" :disabled="fillDisabled" :title="fillTitle" @click="emit('fill')">{{ fillLabel }}</button>
    <span v-if="actionMessage" class="resource-fill-feedback" role="status">{{ actionMessage }}</span>
  </section>
</template>
