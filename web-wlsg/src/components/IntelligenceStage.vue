<!-- 军情页保持官网列表结构并接入真实战报、分页、已读、详情和删除接口。 -->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { intelligenceState } from '../intelligence'
import { intelligenceTabs, intelligenceTotalPages } from '../intelligence/adapter'
import type { IntelligenceTabKey } from '../game/types'
import OfficialBattleReport from './OfficialBattleReport.vue'

const state = intelligenceState.state
const selectedIds = ref<string[]>([])
const deleteConfirmOpen = ref(false)
const totalPages = computed(() => intelligenceTotalPages(state.total, state.pageSize))
const allPageSelected = computed(() => state.reports.length > 0 && state.reports.every((report) => selectedIds.value.includes(report.id)))

/** 切换真实军情分类并清除上一分类的选择。 */
function selectTab(tab: IntelligenceTabKey) {
  selectedIds.value = []
  deleteConfirmOpen.value = false
  void intelligenceState.selectTab(tab)
}

/** 切换当前真实页全部军情的选中状态。 */
function toggleSelectAll() {
  const pageIds = state.reports.map((report) => report.id)
  selectedIds.value = allPageSelected.value ? [] : pageIds
}

/** 打开删除确认；未选择时由状态服务给出统一提示。 */
function requestDelete() {
  if (!selectedIds.value.length) {
    void intelligenceState.deleteReports([])
    return
  }
  deleteConfirmOpen.value = true
}

/** 确认后串行删除所选真实军情并重新读取权威列表。 */
async function confirmDelete() {
  const ids = [...selectedIds.value]
  deleteConfirmOpen.value = false
  await intelligenceState.deleteReports(ids)
  selectedIds.value = []
}

/** 打开后端完整战报并按现有接口标记未读记录。 */
function openReport(reportId: string) { void intelligenceState.openReport(reportId) }

/** 切换真实后端分页并清除当前页选择。 */
function selectPage(page: number) {
  selectedIds.value = []
  void intelligenceState.selectPage(page)
}

/** 标签或分页切换完成后移除已不在当前页的勾选项。 */
watch(() => state.reports.map((report) => report.id).join(','), () => {
  const visible = new Set(state.reports.map((report) => report.id))
  selectedIds.value = selectedIds.value.filter((id) => visible.has(id))
})
</script>

<template>
  <section class="intelligence-stage main-stage" aria-label="武林三国风格军情页">
    <div class="panel-top"><i></i><span></span><b></b></div>
    <div class="panel-center intelligence-panel-center">
      <nav class="stage-toolbar intelligence-tabs" aria-label="军情分类">
        <button v-for="tab in intelligenceTabs" :key="tab.key" type="button" :class="{ active: state.activeTab === tab.key }" :disabled="state.phase === 'loading' || state.deleting" @click="selectTab(tab.key)">{{ tab.label }}</button>
      </nav>
      <div class="intelligence-content">
        <template v-if="state.detail || state.detailLoading || state.detailError">
          <header class="intelligence-list-header intelligence-detail-header">
            <strong><span></span>情报详情</strong>
            <button type="button" @click="intelligenceState.closeReport">返回情报列表</button>
          </header>
          <div v-if="state.detailLoading" class="intelligence-state">正在读取战报详情…</div>
          <div v-else-if="state.detailError" class="intelligence-state error"><p>{{ state.detailError }}</p><button type="button" @click="intelligenceState.closeReport">返回列表</button></div>
          <OfficialBattleReport v-else-if="state.detail" :report="state.detail" @close="intelligenceState.closeReport" />
        </template>
        <template v-else>
          <header class="intelligence-list-header">
            <strong><span></span>情报列表</strong>
            <p v-if="state.activeTab === 'all'">友情提示：“最新”只显示未读军情，阅读后将归入对应分类。</p>
            <p v-else>友情提示：当前分类只显示已读军情，系统仅保留后端可见期限内的记录。</p>
          </header>
          <div class="intelligence-table-frame">
            <table class="intelligence-table">
              <thead><tr><th>选 择</th><th>情报类型</th><th>主 题</th><th>日 期</th></tr></thead>
              <tbody>
                <tr v-for="report in state.reports" :key="report.id" :class="{ unread: !report.read }">
                  <td><input v-model="selectedIds" type="checkbox" :value="report.id" :disabled="state.deleting" :aria-label="`选择军情：${report.title}`" /></td>
                  <td>{{ report.typeLabel }}</td>
                  <td><button type="button" :title="report.title" :disabled="state.detailLoading || state.deleting" @click="openReport(report.id)">{{ report.title }}</button></td>
                  <td>{{ report.createdAt }}</td>
                </tr>
                <tr v-if="state.phase === 'loading' && !state.reports.length" class="intelligence-empty"><td colspan="4">正在读取真实军情…</td></tr>
                <tr v-else-if="state.phase === 'error'" class="intelligence-empty error"><td colspan="4">{{ state.error }} <button type="button" @click="intelligenceState.refresh">重试</button></td></tr>
                <tr v-else-if="!state.reports.length" class="intelligence-empty"><td colspan="4">{{ state.activeTab === 'all' ? '当前没有未读军情' : '当前分类暂无军情' }}</td></tr>
              </tbody>
            </table>
          </div>
          <div class="intelligence-actions">
            <div><button type="button" :disabled="state.phase === 'loading' || state.deleting || !state.reports.length" @click="toggleSelectAll">{{ allPageSelected ? '取消' : '全选' }}</button><button type="button" :disabled="state.phase === 'loading' || state.deleting" @click="requestDelete">{{ state.deleting ? '删除中' : '删除' }}</button></div>
            <nav aria-label="军情分页"><span>{{ state.total }}</span><button type="button" :disabled="state.phase === 'loading' || state.page === 1" @click="selectPage(state.page - 1)">‹</button><b>{{ state.page }}/{{ totalPages }}</b><button type="button" :disabled="state.phase === 'loading' || state.page === totalPages" @click="selectPage(state.page + 1)">›</button></nav>
          </div>
          <p v-if="state.actionMessage" class="intelligence-action-message" role="status">{{ state.actionMessage }}</p>
        </template>
      </div>
    </div>
    <div class="panel-bottom"><i></i><span></span><b></b></div>
    <p class="left-footer">抵制不良游戏 拒绝盗版游戏 注意自我保护 谨防受骗上当 适度游戏益脑 沉迷游戏伤身 合理安排时间 享受健康生活</p>
    <div v-if="deleteConfirmOpen" class="intelligence-confirm-mask" role="presentation">
      <section class="intelligence-confirm" role="dialog" aria-modal="true" aria-labelledby="intelligence-delete-title">
        <header id="intelligence-delete-title">删除军情</header>
        <p>确定删除已选择的 {{ selectedIds.length }} 条军情吗？删除后无法恢复。</p>
        <footer><button type="button" @click="confirmDelete">确定删除</button><button type="button" @click="deleteConfirmOpen = false">取消</button></footer>
      </section>
    </div>
  </section>
</template>
