<!-- NPC 与据点列表：复用官网排行列表结构，并按 Hero3 四层级同页展示规则改版。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { npcFactionLabel, npcRecoveryLabel, npcTierLabel, npcTierMeta, npcTierOrder, npcTraitSummary, sortNpcCities } from '../data/npcDirectory'
import type { NpcCityState } from '../game/types'
import type { NpcPhase } from '../npc/stateService'
import type { WorldMapTarget } from '../worldMap/types'

const props = defineProps<{ mode: 'npc' | 'stronghold'; targets: WorldMapTarget[]; cities: NpcCityState[]; phase: NpcPhase; error: string; refreshing: boolean; actionMessage: string; actionSucceeded: boolean; lastRefreshedAt: string; refreshCost: number; accountGold: number }>()
const emit = defineEmits<{ retry: []; refresh: []; select: [city: NpcCityState] }>()
const cities = computed(() => sortNpcCities(props.cities))
const strongholds = computed(() => props.targets.filter((target) => target.targetType === 'yellow_turban'))

/** 将据点后端状态转换为玩家可读中文，不直接暴露内部枚举。 */
function strongholdStatus(target: WorldMapTarget): string {
  if (target.status === 'active') return '可挑战'
  if (target.status === 'defeated') return '已击破'
  if (target.status === 'unavailable') return '暂不可战'
  if (target.status === 'available') return '可挑战'
  return target.status ? '状态未知' : '未知'
}

/** 将服务端刷新时间转换为当前浏览器的简短显示。 */
function refreshedAtLabel(value: string): string {
  const time = new Date(value)
  return Number.isNaN(time.getTime()) ? '-' : time.toLocaleString('zh-CN', { hour12: false })
}
</script>

<template>
  <section class="npc-directory" :aria-label="mode === 'npc' ? 'NPC 城池列表' : '据点列表'">
    <div v-if="mode === 'npc'" class="npc-tier-legend" aria-label="NPC 规模颜色说明">
      <span v-for="tier in npcTierOrder" :key="tier" :class="`tier-${tier}`" :title="npcTierMeta[tier].description"><i></i>{{ npcTierLabel(tier) }}</span>
      <em>四种规模同页展示</em>
    </div>
    <div class="npc-list-title"><i></i><strong>{{ mode === 'npc' ? 'npc列表' : '据点列表' }}</strong><template v-if="mode === 'npc'"><small>上次刷新：{{ refreshedAtLabel(lastRefreshedAt) }}</small><button type="button" :disabled="refreshing || phase === 'loading' || accountGold < refreshCost" :title="accountGold < refreshCost ? `账号金币不足 ${refreshCost}` : `消耗 ${refreshCost} 账号金币刷新全部 NPC 城池`" @click="emit('refresh')">{{ refreshing ? '刷新中…' : `刷新城池（${refreshCost}金币）` }}</button></template></div>
    <div v-if="mode === 'npc' && phase === 'loading'" class="npc-directory-state"><span class="loading-mark"></span><strong>正在读取真实 NPC 城池…</strong></div>
    <div v-else-if="mode === 'npc' && phase === 'error'" class="npc-directory-state error"><strong>NPC 城池加载失败</strong><p>{{ error }}</p><button type="button" @click="emit('retry')">重新读取</button></div>
    <table v-else-if="mode === 'npc' && cities.length" class="npc-directory-table">
      <thead><tr><th>城池名称</th><th>规模</th><th>阵营</th><th>恢复特性</th><th>城池词条</th></tr></thead>
      <tbody>
        <tr v-for="item in cities" :key="item.id" :class="`tier-${item.tier}`">
          <td><button type="button" :title="`向${item.name}下达战争命令`" @click="emit('select', item)">{{ item.name }}</button></td>
          <td><b>{{ npcTierLabel(item.tier) }}</b></td>
          <td>{{ npcFactionLabel(item.faction) }}</td>
          <td>{{ npcRecoveryLabel(item.recoveryProfile) }}</td>
          <td :title="npcTraitSummary(item)">{{ npcTraitSummary(item) }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else-if="mode === 'npc'" class="npc-directory-state"><strong>当前没有 NPC 城池</strong><p>可以尝试重新读取；付费刷新会消耗 {{ refreshCost }} 账号金币。</p><button type="button" @click="emit('retry')">重新读取</button></div>
    <table v-else class="npc-directory-table stronghold-table">
      <thead><tr><th>据点名称</th><th>坐标</th><th>等级</th><th>方向</th><th>状态</th></tr></thead>
      <tbody>
        <tr v-for="target in strongholds" :key="target.targetId">
          <td><button type="button" :title="`${target.name}（${target.x}|${target.y}）`">{{ target.name }}</button></td>
          <td>{{ target.x }}|{{ target.y }}</td><td>{{ target.level }}</td><td>{{ target.direction || '-' }}</td><td>{{ strongholdStatus(target) }}</td>
        </tr>
        <tr v-if="!strongholds.length" class="npc-directory-empty"><td colspan="5">当前地图视野内暂无据点</td></tr>
      </tbody>
    </table>
    <p v-if="mode === 'npc' && actionMessage" class="npc-directory-feedback" :class="{ success: actionSucceeded, error: !actionSucceeded }">{{ actionMessage }}</p>
    <p class="npc-directory-note">{{ mode === 'npc' ? `共 ${cities.length} 座城池；不同颜色代表小型、中型、大型、超大 NPC，点击城池下达命令。` : '据点数据来自当前世界地图视野，切换地图区域后同步变化。' }}</p>
  </section>
</template>
