<!-- 官方战报详情：动态展示双方、增援、十兵种、损失、收益与情报可见性。 -->
<script setup lang="ts">
import { computed } from 'vue'
import type { BattleReportState } from '../game/types'
import { formatGeneralProgress, toOfficialBattleReport } from '../intelligence/reportAdapter'

const props = defineProps<{ report: BattleReportState }>()
const emit = defineEmits<{ close: [] }>()
const model = computed(() => toOfficialBattleReport(props.report))

/** 统一格式化战报中的兵力与资源数值。 */
function formatNumber(value: number) { return Math.max(0, value || 0).toLocaleString('zh-CN') }
</script>

<template>
  <article class="official-battle-report">
    <header class="official-report-title">{{ model.title }}</header>
    <template v-for="(side, index) in model.sides" :key="side.key">
      <div v-if="index === 1" class="official-report-battle-divider"><b>交 战</b></div>
      <section class="official-report-side" :class="side.role">
        <h3><i></i>{{ side.roleLabel }} - {{ side.name }} <span v-if="index === 0">交战时间：{{ model.occurredAt }}</span></h3>
        <div class="official-report-identity">
          <img class="official-report-faction" :src="side.factionIcon" :alt="side.faction" />
          <div class="official-report-general-center">
            <div class="official-report-general-fields">
              <div class="official-report-general"><b>将领名称: {{ side.general?.name || '-' }}<template v-if="side.general?.level"> (Lv {{ side.general.level }})</template></b></div>
              <div><b>官职: -</b></div>
              <div><b>军衔: -</b></div>
            </div>
            <p class="official-report-general-exp">获得经验：<b>{{ side.general ? formatGeneralProgress(side.generalExp, side.generalLevelBefore, side.generalLevelAfter) : '-' }}</b></p>
          </div>
          <div class="official-report-result" :class="side.result"><img v-if="side.result === 'defeat'" src="/assets/official/report/zz_sb.gif" alt="败" /><b v-else>{{ side.resultLabel }}</b></div>
        </div>
        <p class="official-report-traits"><b>将领特性：</b><span v-if="side.traits.length" class="official-report-trait-list"><span v-for="trait in side.traits" :key="trait.key" class="official-report-trait"><strong>{{ trait.name }}</strong><em v-if="trait.phase">{{ trait.phase }}</em><span v-if="trait.detailText">{{ trait.detailText }}</span></span></span><template v-else>-</template></p>
        <table class="official-report-units">
          <tbody>
            <tr><th>兵种</th><td v-for="unit in side.units" :key="unit.key" :title="unit.name"><img v-if="unit.icon" :src="unit.icon" :alt="unit.name" /><span v-else>?</span></td></tr>
            <tr><th>出动</th><td v-for="unit in side.units" :key="unit.key">{{ formatNumber(unit.dispatched) }}</td></tr>
            <tr><th>阵亡</th><td v-for="unit in side.units" :key="unit.key">{{ formatNumber(unit.lost) }}</td></tr>
          </tbody>
        </table>
        <dl v-if="side.role === 'attacker'" class="official-report-summary-rows">
          <div><dt>掠夺资源</dt><dd>{{ model.resourceText }}</dd></div>
          <div><dt>战损反馈</dt><dd>{{ model.feedbackText }}</dd></div>
          <div><dt>宝物掉落</dt><dd><template v-if="model.dropItems.length"><span v-for="drop in model.dropItems" :key="drop.key" class="official-report-drop-item">{{ drop.name }} {{ formatNumber(drop.amount) }} 个</span></template><template v-else>无</template></dd></div>
          <div><dt>获得城金</dt><dd><span class="official-report-city-gold">{{ formatNumber(model.cityGold) }}</span></dd></div>
          <div v-if="model.wallText"><dt>城墙信息</dt><dd>{{ model.wallText }}</dd></div>
          <div v-if="model.visibilityReason"><dt>防守方信息</dt><dd>{{ model.visibilityReason }}</dd></div>
        </dl>
      </section>
    </template>
    <button class="official-report-close" type="button" @click="emit('close')">关 闭</button>
  </article>
</template>
