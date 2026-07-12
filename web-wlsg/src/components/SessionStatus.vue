<!-- 公共外壳中克制展示的当前账号、玩家与退出入口。 -->
<script setup lang="ts">
defineProps<{ username: string; nickname: string }>()
const emit = defineEmits<{ logout: [] }>()
</script>

<template>
  <div class="session-status"><span :title="`账号：${username}`">{{ nickname }}</span><button type="button" @click="emit('logout')">退出</button></div>
</template>
