<!-- 登录后的存档列表和当前玩家选择界面。 -->
<script setup lang="ts">
import type { AccountInfo, PlayerSummary } from '../api/types'

defineProps<{ account: AccountInfo; players: PlayerSummary[]; error: string }>()
const emit = defineEmits<{ select: [playerId: string]; logout: [] }>()

const factionNames: Record<string, string> = { wei: '魏国', shu: '蜀国', wu: '吴国' }

/** 将后端时间格式化为简洁的本地时间。 */
function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}
</script>

<template>
  <main class="auth-page">
    <section class="player-card">
      <header><div><h1>选择存档</h1><p>账号：{{ account.username }}</p></div><button type="button" class="text-button" @click="emit('logout')">退出登录</button></header>
      <p v-if="error" class="form-error" role="alert">{{ error }}</p>
      <div v-if="players.length" class="player-list">
        <button v-for="player in players" :key="player.id" type="button" class="player-entry" :disabled="Boolean(player.deleteScheduledAt)" @click="emit('select', player.id)">
          <strong>{{ player.nickname }}</strong><span>{{ factionNames[player.faction] ?? player.faction }}</span>
          <small>建筑等级 {{ player.buildingLevel }} · 总兵力 {{ player.totalArmy }}</small>
          <small>{{ player.deleteScheduledAt ? '该存档正在等待删除' : `最后更新 ${formatTime(player.updatedAt)}` }}</small>
        </button>
      </div>
      <div v-else class="empty-state"><strong>当前账号还没有玩家存档</strong><p>V0.2 暂不提供创建存档功能，请使用现有 Hero3 前端创建后再返回。</p></div>
    </section>
  </main>
</template>
