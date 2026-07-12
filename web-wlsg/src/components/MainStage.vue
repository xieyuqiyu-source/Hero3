<!-- 官网资源建筑页框架中的真实地块网格与加载错误状态。 -->
<script setup lang="ts">
import { computed } from 'vue'
import type { BuildingBalanceConfig } from '../api/types'
import type { GameStatePhase } from '../game/stateService'
import type { ResourceBuildingViewModel } from '../game/types'

const props = defineProps<{ subNavigation: string[]; buildings: ResourceBuildingViewModel[]; buildingConfigs: Record<string, BuildingBalanceConfig>; resources: Record<string, number>; phase: GameStatePhase; error: string; actionMessage: string; upgradingBuildingId: string | null }>()
const emit = defineEmits<{ retry: []; upgrade: [buildingId: string] }>()

/** 生成建筑状态文本，倒计时统一交给右侧建造队列展示。 */
function statusText(building: ResourceBuildingViewModel) {
  return building.endsAt ? '升级中' : building.status
}

const hasBuildings = computed(() => props.buildings.length > 0)

const resourceLabels: Record<string, string> = { wood: '木材', stone: '泥土', iron: '铁矿', food: '粮食' }
const buildingDescriptions: Record<string, string> = {
  wood_camp: '伐木场是生产木材的地方，等级越高，生产的木材越多。',
  stone_quarry: '泥土场是生产泥土的地方，等级越高，生产的泥土越多。',
  iron_mine: '铁矿场是生产铁矿的地方，等级越高，生产的铁矿越多。',
  farm: '农田是生产粮食的地方，等级越高，生产的粮食越多。',
}

/** 返回建筑对应的后台平衡配置。 */
function configFor(building: ResourceBuildingViewModel) { return props.buildingConfigs[building.buildingType] }

/** 返回当前等级产量。 */
function currentProduction(building: ResourceBuildingViewModel) { return configFor(building)?.productionByLevel?.[building.level] ?? 0 }

/** 返回下一级产量，最高级时返回空值。 */
function nextProduction(building: ResourceBuildingViewModel) { return configFor(building)?.productionByLevel?.[building.level + 1] }

/** 返回下一级的四项资源消耗。 */
function upgradeCosts(building: ResourceBuildingViewModel) {
  const cost = configFor(building)?.upgradeCostByLevel?.[building.level] ?? {}
  return ['wood', 'stone', 'iron', 'food'].map((key) => ({ key, name: resourceLabels[key], amount: cost[key] ?? 0, insufficient: (props.resources[key] ?? 0) < (cost[key] ?? 0) }))
}

/** 将后台升级秒数格式化为官网风格时间。 */
function upgradeDuration(building: ResourceBuildingViewModel) {
  const total = configFor(building)?.upgradeSecondsByLevel?.[building.level] ?? 0
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor(total % 3600 / 60)
  const seconds = total % 60
  return [hours, minutes, seconds].map((value) => String(value).padStart(2, '0')).join(':')
}

/** 格式化升级资源数量。 */
function formatNumber(value: number) { return Number(value || 0).toLocaleString('zh-CN') }

/** 判断建筑是否存在后续等级且不处于升级中。 */
function hasUpgradeLevel(building: ResourceBuildingViewModel) {
  const config = configFor(building)
  return !building.endsAt && Boolean(config?.upgradeCostByLevel?.[building.level] || config?.goldUpgradeCostByLevel?.[building.level] !== undefined)
}

/** 判断当前四项资源是否足以支付下一级消耗。 */
function hasSufficientResources(building: ResourceBuildingViewModel) {
  return upgradeCosts(building).every((cost) => !cost.insufficient)
}

/** 判断页面是否允许选择该建筑的建造按钮。 */
function canStartUpgrade(building: ResourceBuildingViewModel) {
  return hasUpgradeLevel(building) && hasSufficientResources(building) && !props.upgradingBuildingId
}

/** 返回禁用建造按钮的明确原因。 */
function upgradeButtonTitle(building: ResourceBuildingViewModel) {
  if (building.endsAt) return '该建筑正在升级'
  if (!hasUpgradeLevel(building)) return '已达当前配置最高级'
  const missing = upgradeCosts(building).filter((cost) => cost.insufficient).map((cost) => cost.name)
  if (missing.length) return `${missing.join('、')}不足，暂不可建造`
  if (props.upgradingBuildingId) return '正在提交其他建筑的建造请求'
  return '建造下一级'
}
</script>

<template>
  <section class="main-stage">
    <div class="panel-top"><i></i><span></span><b></b></div>
    <div class="panel-center">
      <nav class="secondary-navigation" aria-label="二级导航"><button v-for="(item, index) in subNavigation" :key="item" type="button" :class="{ active: index === 1 }">{{ item }}</button></nav>
      <div class="stage-toolbar"><button type="button" class="active">资源建筑</button><button type="button">军事建筑</button><button type="button">拓建新城</button><button type="button">迁都</button></div>
      <div class="stage-content real-stage-content">
        <div class="official-description">资源建筑会持续生产木材、泥土、铁矿和粮食。悬浮可查看真实产量和升级消耗，点击“建”可启动建造。</div>
        <p v-if="actionMessage" class="building-action-message" aria-live="polite">{{ actionMessage }}</p>
        <div v-if="phase === 'loading'" class="game-state-message"><span class="loading-mark"></span><strong>正在加载当前城池…</strong></div>
        <div v-else-if="phase === 'error'" class="game-state-message error"><strong>城池数据加载失败</strong><p>{{ error }}</p><button type="button" @click="emit('retry')">重新加载</button></div>
        <div v-else-if="phase === 'ready' && !hasBuildings" class="game-state-message"><strong>当前存档暂无资源地块</strong><p>接口没有返回可展示的资源地块或资源建筑。</p></div>
        <div v-else-if="phase === 'ready'" class="resource-building-grid">
          <article v-for="building in buildings" :key="building.slotId" class="resource-building-card" :class="{ fallback: building.isFallback, unavailable: hasUpgradeLevel(building) && !hasSufficientResources(building) }" tabindex="0">
            <div class="building-picture">
              <img v-if="building.image" :src="`/assets/official/images/${building.image}`" :alt="building.buildingName" />
              <span v-else>暂无图片</span>
              <div class="building-actions"><b>{{ building.level }}</b><i title="拆除尚未接入">拆</i><button type="button" :disabled="!canStartUpgrade(building)" :title="upgradeButtonTitle(building)" @click.stop="emit('upgrade', building.id)">{{ upgradingBuildingId === building.id ? '…' : '建' }}</button></div>
            </div>
            <strong>{{ building.buildingName }}</strong><small>{{ building.resourceName }} · {{ statusText(building) }}</small>
            <section v-if="configFor(building)" class="building-tooltip" role="tooltip">
              <h3>{{ building.buildingName }}（等级 {{ building.level }}）</h3>
              <p>{{ buildingDescriptions[building.buildingType] ?? `${building.buildingName}的等级越高，建筑效果越强。` }}</p>
              <strong v-if="configFor(building)?.productionByLevel">当前等级产量：{{ currentProduction(building) }} /小时</strong>
              <strong v-if="nextProduction(building) !== undefined">高一等级产量：{{ nextProduction(building) }} /小时</strong>
              <h4 v-if="hasUpgradeLevel(building)">提升至等级 {{ building.level + 1 }}:</h4>
              <div v-if="hasUpgradeLevel(building)" class="tooltip-costs"><span v-for="cost in upgradeCosts(building)" :key="cost.key" :class="{ insufficient: cost.insufficient }">{{ cost.name }}: {{ formatNumber(cost.amount) }}</span><span>时间: {{ upgradeDuration(building) }}</span></div>
              <p v-if="hasUpgradeLevel(building) && !hasSufficientResources(building)" class="building-unavailable-reason">资源不足，当前不可建造</p>
              <h4 v-if="!hasUpgradeLevel(building)">{{ building.endsAt ? '当前正在升级' : '已达当前配置最高级' }}</h4>
            </section>
          </article>
        </div>
      </div>
    </div>
    <div class="panel-bottom"><i></i><span></span><b></b></div>
    <div class="left-footer">抵制不良游戏　拒绝盗版游戏　注意自我保护　谨防受骗上当　适度游戏益脑　沉迷游戏伤身</div>
  </section>
</template>
