<!-- 官方地图页：以真实世界坐标渲染玩家城池、黄巾营地和只读目标能力。 -->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { GeneralAssignmentState, GeneralState, WorldMapMarchAction } from '../game/types'
import type { RecruitmentUnitViewModel } from '../game/recruitmentAdapter'
import { createWorldMapGrid, type WorldMapCell } from '../worldMap/gridAdapter'
import type { WorldMapPhase } from '../worldMap/stateService'
import type { WorldMapTarget, WorldMapViewResponse } from '../worldMap/types'
import MarchCommandDialog from './MarchCommandDialog.vue'

const props = defineProps<{ phase: WorldMapPhase; data: WorldMapViewResponse | null; error: string; overviewPhase: WorldMapPhase; overview: WorldMapViewResponse | null; overviewError: string; sourceName: string; units: RecruitmentUnitViewModel[]; generals: GeneralState[]; assignments: GeneralAssignmentState[]; dispatching: boolean; marchMessage: string; marchSucceeded: boolean; marchResultVersion: number }>()
const emit = defineEmits<{ retry: []; navigate: [x: number, y: number]; home: []; dispatch: [action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]] }>()
const selectedKey = ref('')
const queryX = ref(0)
const queryY = ref(0)
const savedCoordinate = ref('')
const zoomLevels = [.75, 1, 1.25, 1.5]
const zoom = ref(1)
const miniMapOpen = ref(true)
const commandTarget = ref<WorldMapTarget | null>(null)
const commandAction = ref<WorldMapMarchAction>('attack')
const commandStartVersion = ref(0)
const mapTabs = ['地图', 'NPC据点', '副本', '万象幻境']
const grid = computed(() => props.data ? createWorldMapGrid(props.data, zoom.value) : [])
const targetByKey = computed(() => new Map((props.data?.targets ?? []).map((target) => [`${target.x}:${target.y}`, target])))
const selectedTarget = computed(() => targetByKey.value.get(selectedKey.value) ?? null)
const selectedCoordinate = computed(() => {
  const [x, y] = selectedKey.value.split(':').map(Number)
  return Number.isFinite(x) && Number.isFinite(y) ? { x, y } : null
})
const mapZoomStyle = computed(() => ({ '--map-zoom': String(zoom.value) }))
const miniTargets = computed(() => props.overview?.targets ?? [])
const commandMessage = computed(() => props.marchResultVersion > commandStartVersion.value ? props.marchMessage : '')
const commandSucceeded = computed(() => props.marchResultVersion > commandStartVersion.value && props.marchSucceeded)
const miniViewportStyle = computed(() => {
  if (!props.data) return {}
  const columns = Math.max(...grid.value.map((row) => row.length), 1)
  const rows = Math.max(grid.value.length, 1)
  const width = Math.max(5, columns / props.data.width * 100)
  const height = Math.max(5, rows / props.data.height * 100)
  return { width: `${width}%`, height: `${height}%`, left: `${Math.max(0, Math.min(100 - width, props.data.centerX / (props.data.width - 1) * 100 - width / 2))}%`, top: `${Math.max(0, Math.min(100 - height, props.data.centerY / (props.data.height - 1) * 100 - height / 2))}%` }
})

/** 新视野到达时默认选中自己的真实城池，并同步坐标输入框。 */
watch(() => props.data, (view) => {
  if (!view) return
  queryX.value = view.centerX
  queryY.value = view.centerY
  if (!targetByKey.value.has(selectedKey.value)) selectedKey.value = `${view.self.x}:${view.self.y}`
}, { immediate: true })

/** 选中地图地块并由右侧官方信息框展示目标。 */
function selectCell(cell: WorldMapCell) { selectedKey.value = cell.key }

/** 校验坐标后交给状态服务读取新的真实视野。 */
function searchCoordinate() {
  if (!props.data) return
  emit('navigate', Math.max(0, Math.min(props.data.width - 1, Math.trunc(Number(queryX.value)))), Math.max(0, Math.min(props.data.height - 1, Math.trunc(Number(queryY.value)))))
}

/** 按官方八方向罗盘移动地图中心。 */
function move(dx: number, dy: number) {
  if (!props.data) return
  emit('navigate', props.data.centerX + dx, props.data.centerY + dy)
}

/** 仅在当前页面内记录最近收藏坐标，不伪造后端持久化。 */
function collectCoordinate() {
  if (selectedCoordinate.value) savedCoordinate.value = `${selectedCoordinate.value.x} | ${selectedCoordinate.value.y}`
}

/** 在四档比例之间缩放，并保持官方 100% 为默认档。 */
function changeZoom(step: number) {
  const index = zoomLevels.indexOf(zoom.value)
  zoom.value = zoomLevels[Math.max(0, Math.min(zoomLevels.length - 1, index + step))]
}

/** 将鼠标滚轮方向转换为一档缩放，阻止页面跟随滚动。 */
function zoomWithWheel(event: WheelEvent) { changeZoom(event.deltaY < 0 ? 1 : -1) }

/** 点击小地图空白处后换算为 100×100 世界坐标。 */
function navigateFromMiniMap(event: MouseEvent) {
  if (!props.overview) return
  const box = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = Math.round((event.clientX - box.left) / box.width * (props.overview.width - 1))
  const y = Math.round((event.clientY - box.top) / box.height * (props.overview.height - 1))
  emit('navigate', x, y)
}

/** 点击小地图目标时直接居中，并同步主地图选择坐标。 */
function navigateToMiniTarget(target: WorldMapTarget) {
  selectedKey.value = `${target.x}:${target.y}`
  emit('navigate', target.x, target.y)
}

/** 计算真实目标在世界小地图中的百分比位置与关系色。 */
function miniTargetStyle(target: WorldMapTarget) {
  if (!props.overview) return {}
  return { left: `${target.x / Math.max(1, props.overview.width - 1) * 100}%`, top: `${target.y / Math.max(1, props.overview.height - 1) * 100}%` }
}

/** 返回目标动作禁用原因，能力值完全以后端响应为准。 */
function actionTitle(target: WorldMapTarget | null, action: 'attack' | 'plunder' | 'scout' | 'reinforce') {
  if (!target) return '请先选择玩家城池'
  const enabled = { attack: target.canAttack, plunder: target.canPlunder, scout: target.canScout, reinforce: target.canReinforce }[action]
  const reason = { attack: target.attackReason, plunder: target.plunderReason, scout: target.scoutReason, reinforce: target.reinforceReason }[action]
  return enabled ? `当前目标可${{ attack: '攻击', plunder: '掠夺', scout: '侦查', reinforce: '增援' }[action]}` : (reason || target.reason || '当前目标不可执行该操作')
}

/** 为能力允许的玩家城池打开官方战争命令弹窗。 */
function openCommand(action: WorldMapMarchAction) {
  const target = selectedTarget.value
  const allowed = target && { attack: target.canAttack, plunder: target.canPlunder, scout: target.canScout, reinforce: target.canReinforce }[action]
  if (!target?.playerId || !allowed) return
  commandAction.value = action
  commandStartVersion.value = props.marchResultVersion
  commandTarget.value = target
}

/** 将弹窗输入原样转交统一玩家状态服务。 */
function dispatchCommand(action: WorldMapMarchAction, targetPlayerId: string, troops: Record<string, number>, generalIds: string[]) { emit('dispatch', action, targetPlayerId, troops, generalIds) }
</script>

<template>
  <section class="main-stage map-stage" aria-label="世界地图页">
    <div class="panel-top"><i></i><span></span><b></b></div>
    <div class="panel-center map-panel-center">
      <nav class="secondary-navigation" aria-label="地图二级导航">
        <button v-for="(tab, index) in mapTabs" :key="tab" type="button" :class="{ active: index === 0 }" :disabled="index !== 0">{{ tab }}</button>
      </nav>
      <div v-if="phase === 'loading' && !data" class="map-state-message"><span class="loading-mark"></span><strong>正在读取真实世界地图…</strong></div>
      <div v-else-if="phase === 'error' && !data" class="map-state-message error"><strong>世界地图加载失败</strong><p>{{ error }}</p><button type="button" @click="emit('retry')">重新读取</button></div>
      <div v-else-if="data" class="official-map-list">
        <div class="official-map-left">
          <ul class="official-map-legend" aria-label="地图关系图例">
            <li><img src="/assets/official/map/map_a4.gif" alt="" />自 己</li>
            <li><img src="/assets/official/map/map_a2.gif" alt="" />本 盟</li>
            <li><img src="/assets/official/map/map_a9.gif" alt="" />同 盟</li>
            <li><img src="/assets/official/map/map_a3.gif" alt="" />敌 人</li>
            <li class="yellow-turban-legend"><i></i>黄巾营地</li>
          </ul>
          <div class="official-map-box-top"></div>
          <div class="official-map-box-left"></div>
          <div class="official-map-viewport" :class="{ refreshing: phase === 'loading' }" :style="mapZoomStyle" aria-label="真实世界地图格" @wheel.prevent="zoomWithWheel">
            <div v-for="(row, rowIndex) in grid" :key="rowIndex" class="official-map-row" :class="rowIndex % 2 === 0 ? 'second' : 'first'" :style="{ top: `${(-54 + rowIndex * 21) * zoom}px` }">
              <button v-for="cell in row" :key="cell.key" type="button" class="official-map-cell" :class="{ selected: selectedKey === cell.key, target: cell.target, yellow: cell.target?.targetType === 'yellow_turban' }" :title="cell.target ? `${cell.target.name}（${cell.x}|${cell.y}）` : `荒野（${cell.x}|${cell.y}）`" @click="selectCell(cell)">
                <img :src="cell.image" :alt="cell.target?.name ?? `荒野 ${cell.x}|${cell.y}`" />
                <i class="official-map-hitbox" aria-hidden="true"></i>
                <span v-if="cell.target?.targetType === 'yellow_turban'">黄巾</span>
              </button>
            </div>
            <section v-if="miniMapOpen" class="world-mini-map" aria-label="世界小地图">
              <header><b>世界小地图</b><button type="button" title="收起小地图" @click="miniMapOpen = false">×</button></header>
              <button class="world-mini-map-canvas" type="button" title="点击移动主地图" @click="navigateFromMiniMap">
                <i class="mini-map-viewport" :style="miniViewportStyle"></i>
                <span v-for="target in miniTargets" :key="target.targetId" class="mini-map-target" :class="[target.relation, { yellow: target.targetType === 'yellow_turban' }]" :style="miniTargetStyle(target)" :title="`${target.name}（${target.x}|${target.y}）`" @click.stop="navigateToMiniTarget(target)"></span>
                <em v-if="overviewPhase === 'loading'">读取中…</em><em v-else-if="overviewPhase === 'error'" :title="overviewError">加载失败</em>
              </button>
            </section>
          </div>
          <div class="official-map-box-right"><ol><li v-for="value in 18" :key="value" :class="{ major: value % 4 === 1 }">{{ value % 4 === 1 ? Math.max(0, data.centerY - 10 + value) : '' }}</li></ol></div>
          <div class="official-map-box-bottom"></div>
          <form class="official-map-search" @submit.prevent="searchCoordinate">
            <label>X <input v-model.number="queryX" type="number" :min="0" :max="data.width - 1" /></label>
            <label>Y <input v-model.number="queryY" type="number" :min="0" :max="data.height - 1" /></label>
            <button type="submit"><img src="/assets/official/map/map_cz.gif" alt="查询" /></button>
            <button type="button" @click="emit('home')"><img src="/assets/official/map/fhbc.gif" alt="返回本城" /></button>
            <button type="button" disabled title="当前版本未开放迁移城池"><img src="/assets/official/map/bqcc.gif" alt="迁移城池" /></button>
            <button type="button" @click="collectCoordinate"><img src="/assets/official/map/map_collect.gif" alt="收藏坐标" /></button>
            <div class="map-zoom-controls" aria-label="地图缩放">
              <button type="button" :disabled="zoom === zoomLevels[0]" title="缩小地图" @click="changeZoom(-1)">−</button><b>{{ Math.round(zoom * 100) }}%</b><button type="button" :disabled="zoom === zoomLevels[zoomLevels.length - 1]" title="放大地图" @click="changeZoom(1)">＋</button>
              <button v-if="!miniMapOpen" type="button" class="mini-map-toggle" title="展开小地图" @click="miniMapOpen = true">图</button>
            </div>
            <span v-if="savedCoordinate && !miniMapOpen">已收藏 {{ savedCoordinate }}</span>
          </form>
        </div>
        <aside class="official-map-right" aria-label="地图目标信息">
          <div class="official-map-compass">
            <img src="/assets/official/map/map_bagua.gif" alt="八方向地图罗盘" />
            <button class="nw" type="button" title="向左上移动" @click="move(-7, -10)"></button><button class="n" type="button" title="向上移动" @click="move(0, -10)"></button><button class="ne" type="button" title="向右上移动" @click="move(7, -10)"></button>
            <button class="w" type="button" title="向左移动" @click="move(-7, 0)"></button><button class="center" type="button" title="刷新当前中心" @click="move(0, 0)"></button><button class="e" type="button" title="向右移动" @click="move(7, 0)"></button>
            <button class="sw" type="button" title="向左下移动" @click="move(-7, 10)"></button><button class="s" type="button" title="向下移动" @click="move(0, 10)"></button><button class="se" type="button" title="向右下移动" @click="move(7, 10)"></button>
          </div>
          <div class="official-map-city-card">
            <ul>
              <li>{{ selectedCoordinate ? `${selectedCoordinate.x} | ${selectedCoordinate.y}` : '- | -' }}</li>
              <li :title="selectedTarget?.name">{{ selectedTarget?.name ?? '荒野' }}</li>
              <li>{{ selectedTarget?.targetType === 'yellow_turban' ? '黄巾军' : (selectedTarget?.name ?? '-') }}</li>
              <li>{{ selectedTarget ? `城池等级 ${selectedTarget.level}` : '-' }}</li>
              <li>{{ selectedTarget?.relation === 'self' ? '自己' : selectedTarget?.relation === 'ally' ? '本盟' : '-' }}</li>
            </ul>
            <div class="official-map-city-actions"><button type="button" :disabled="!selectedTarget"><img src="/assets/official/map/map_ck.gif" alt="查看" /></button><button type="button" :disabled="!selectedTarget?.playerId"><img src="/assets/official/map/map_yj.gif" alt="邮件" /></button></div>
          </div>
          <div class="official-map-war-actions">
            <div>
              <button type="button" :disabled="!selectedTarget?.canAttack" :title="actionTitle(selectedTarget, 'attack')" @click="openCommand('attack')"><img src="/assets/official/map/map_gj.gif" alt="攻击" /></button>
              <button type="button" :disabled="!selectedTarget?.canPlunder" :title="actionTitle(selectedTarget, 'plunder')" @click="openCommand('plunder')"><img src="/assets/official/map/map_ld.gif" alt="掠夺" /></button>
              <button type="button" :disabled="!selectedTarget?.canScout" :title="actionTitle(selectedTarget, 'scout')" @click="openCommand('scout')"><img src="/assets/official/map/map_zc.gif" alt="侦查" /></button>
            </div>
            <div><button type="button" :disabled="!selectedTarget?.canReinforce" :title="actionTitle(selectedTarget, 'reinforce')" @click="openCommand('reinforce')"><img src="/assets/official/map/map_zy.gif" alt="增援" /></button><button type="button" disabled title="当前版本未开放拓荒新城"><img src="/assets/official/map/map_tjxc.gif" alt="拓荒新城" /></button></div>
          </div>
          <p v-if="selectedTarget?.reason" class="official-map-reason" :title="selectedTarget.reason">{{ selectedTarget.reason }}</p>
        </aside>
      </div>
    </div>
    <div class="panel-bottom"><i></i><span></span><b></b></div>
    <div class="left-footer">抵制不良游戏　拒绝盗版游戏　注意自我保护　谨防受骗上当　适度游戏益脑　沉迷游戏伤身</div>
    <MarchCommandDialog v-if="commandTarget" :target="commandTarget" :source-name="sourceName" :units="units" :generals="generals" :assignments="assignments" :initial-action="commandAction" :submitting="dispatching" :message="commandMessage" :succeeded="commandSucceeded" :result-version="marchResultVersion" @close="commandTarget = null" @submit="dispatchCommand" />
  </section>
</template>
