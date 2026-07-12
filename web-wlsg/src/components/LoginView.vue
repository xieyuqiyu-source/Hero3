<!-- V0.2 简洁登录界面，不保存或回显密码。 -->
<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ submitting: boolean; error: string }>()
const emit = defineEmits<{ login: [username: string, password: string]; retry: [] }>()
const username = ref('')
const password = ref('')

/** 提交账号密码并立刻清空组件内密码。 */
function submit() {
  if (props.submitting || !username.value.trim() || !password.value) return
  const currentPassword = password.value
  password.value = ''
  emit('login', username.value.trim(), currentPassword)
}
</script>

<template>
  <main class="auth-page">
    <section class="auth-card">
      <img src="/assets/official/images/logo.jpg" alt="武林三国" class="auth-logo" />
      <h1>登录 Hero3</h1>
      <p class="auth-subtitle">使用现有 Hero3 账号进入游戏</p>
      <form @submit.prevent="submit">
        <label>账号<input v-model="username" name="username" autocomplete="username" required /></label>
        <label>密码<input v-model="password" name="password" type="password" autocomplete="current-password" required /></label>
        <p v-if="error" class="form-error" role="alert">{{ error }}</p>
        <button type="submit" :disabled="submitting || !username.trim() || !password">{{ submitting ? '登录中…' : '登录' }}</button>
      </form>
      <p class="auth-hint">当前版本不提供注册或创建存档功能</p>
    </section>
  </main>
</template>
