<!-- 右侧常驻栏使用真实玩家、资源、军队、队列和未读数量。 -->
<script setup lang="ts">
import { toRef } from 'vue'
import type { CityGameViewModel } from '../game/types'
import { formatRemaining, useServerClock } from '../game/useServerClock'

const props = defineProps<{ model: CityGameViewModel; receivedAt: number | null }>()
const topShortcuts = [
  { image: 'url_jq1.gif', label: '军情', countKey: 'message' }, { image: 'url_xh1.gif', label: '信函', countKey: 'mail' },
  { image: 'url_cz.gif', label: '充值', countKey: '' }, { image: 'url_zh.gif', label: '账户', countKey: '' },
]
const bottomShortcuts = [
  { image: 'url_rw.gif', label: '任务' }, { image: 'url_bz.gif', label: '帮助' },
  { image: 'url_bbs.gif', label: '论坛' }, { image: 'url_out.gif', label: '退出' },
]
const { remainingSeconds } = useServerClock(toRef(props.model, 'serverTime'), toRef(props, 'receivedAt'))

/** 返回快捷入口对应的真实未读数量。 */
function shortcutCount(key: string) {
  if (key === 'message') return props.model.unreadMessageCount
  if (key === 'mail') return props.model.unreadMailCount
  return 0
}

/** 格式化资源和军队真实数值。 */
function formatNumber(value: number) { return Number(value || 0).toLocaleString('zh-CN') }

/** 格式化建造队列的服务端基准剩余时间。 */
function queueText(endsAt: string) {
  const seconds = remainingSeconds(endsAt)
  return seconds > 0 ? formatRemaining(seconds) : '结算中'
}
</script>

<template>
  <aside class="side-panel" aria-label="当前玩家真实状态">
    <div class="right-top"></div>
    <nav class="side-shortcuts top-shortcuts" aria-label="右侧快捷菜单">
      <button v-for="item in topShortcuts" :key="item.label" type="button"><img :src="`/assets/official/images/${item.image}`" :alt="item.label" /><b v-if="shortcutCount(item.countKey)">{{ shortcutCount(item.countKey) }}</b></button>
    </nav>
    <div class="city-name"><span>{{ model.player.nickname }}</span><button type="button" aria-label="当前阵营">{{ model.player.factionName.slice(0, 1) }}</button></div>
    <section class="notice-box real-notice"><h2>当前状态</h2><p>账号金币：{{ formatNumber(model.accountGold) }}</p><p>城金：{{ formatNumber(model.cityGold) }}</p><p>坐标、VIP 与推广信息尚未接入</p></section>
    <section class="side-section resource-section">
      <h2>城池生产力</h2>
      <div class="resource-row" v-for="resource in model.resources" :key="resource.key">
        <img :src="`/assets/official/images/${resource.icon}`" alt="" /><strong>{{ formatNumber(resource.productionPerHour) }} 每小时</strong>
        <button type="button" disabled title="V0.3 只读展示；购买加成需另行授权"><img src="/assets/official/images/g_cf_2_2.gif" alt="加成" /></button>
        <button type="button" disabled title="查看加成（后续写操作版本接入）"><img src="/assets/official/images/add_0.gif" alt="详情" /></button>
      </div>
    </section>
    <section class="side-section army-section">
      <h2>本城直属军队 <span>{{ model.general ? `武将 Lv.${model.general.level}` : '无驻城武将' }}</span></h2>
      <div v-if="model.general" class="general-row"><img :src="`/assets/official/images/${model.general.icon}`" alt="" />{{ model.general.name }}（等级{{ model.general.level }}）</div>
      <div v-if="model.army.length" class="troop-grid real-troop-grid"><div v-for="unit in model.army" :key="unit.key" class="troop-item" :title="unit.name" tabindex="0" :aria-label="`${unit.name}，数量 ${formatNumber(unit.amount)}`"><img :src="`/assets/official/images/${unit.icon}`" :alt="unit.name" /><span>{{ formatNumber(unit.amount) }}</span><span class="troop-name-tooltip" role="tooltip">{{ unit.name }}</span></div></div>
      <div v-else class="side-empty">当前没有直属军队</div>
    </section>
    <section class="side-section build-section">
      <h2>城池建造</h2>
      <div v-for="queue in model.buildQueues" :key="queue.id" class="build-row"><span>↑</span>{{ queue.name }}({{ queue.level }}) {{ queueText(queue.endsAt) }}</div>
      <div v-if="!model.buildQueues.length" class="side-empty">当前没有建造队列</div>
    </section>
    <div class="right-bottom"></div>
    <nav class="side-shortcuts bottom-shortcuts" aria-label="右侧底部菜单"><button v-for="item in bottomShortcuts" :key="item.label" type="button"><img :src="`/assets/official/images/${item.image}`" :alt="item.label" /></button></nav>
  </aside>
</template>
