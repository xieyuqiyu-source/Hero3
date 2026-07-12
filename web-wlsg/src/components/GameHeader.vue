<!-- 使用官网公开图片资源复刻顶部品牌、状态和一级导航。 -->
<script setup lang="ts">
defineProps<{
  account: { serverTime: string; currencies: string[]; quickLinks: string[] }
  navigation: string[]
  activeIndex: number
}>()
const emit = defineEmits<{ select: [index: number] }>()
</script>

<template>
  <header class="game-header">
    <img class="official-logo" src="/assets/official/images/logo.jpg" alt="武林三国" />
    <div class="account-area">
      <div class="account-line">
        <span class="server-time">系统时间: {{ account.serverTime }}</span>
        <span v-for="currency in account.currencies" :key="currency">{{ currency }}</span>
        <span v-for="(link, index) in account.quickLinks" :key="link" class="quick-link">
          <img :src="`/assets/official/images/${['card.gif', 'xsyd.gif', 'kfzx.gif'][index]}`" alt="" />{{ link }}
        </span>
      </div>
      <nav class="primary-navigation" aria-label="一级导航">
        <button v-for="(item, index) in navigation" :key="item" type="button" :class="{ active: index === activeIndex }" :title="item" @click="emit('select', index)">
          <img :src="`/assets/official/images/url_${['a', 'b', 'c', 'd', 'e', 'g'][index]}_${index === activeIndex ? '2' : '1'}.gif`" :alt="item" />
        </button>
      </nav>
    </div>
    <div class="country-mark" aria-hidden="true"></div>
  </header>
</template>
