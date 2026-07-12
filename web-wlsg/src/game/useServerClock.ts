/** 基于服务端时间和响应到达时间推进显示时钟与倒计时。 */
import { computed, onBeforeUnmount, ref, watch, type Ref } from 'vue'

/** 创建不依赖本机绝对时间、只使用本机经过时长的服务端时钟。 */
export function useServerClock(serverTime: Ref<string>, receivedAt: Ref<number | null>) {
  const elapsed = ref(0)
  let timer: number | undefined

  /** 重置并启动一秒一次的经过时长计数。 */
  function restart() {
    elapsed.value = receivedAt.value ? Math.max(0, Date.now() - receivedAt.value) : 0
    if (timer !== undefined) window.clearInterval(timer)
    timer = window.setInterval(() => { elapsed.value = receivedAt.value ? Math.max(0, Date.now() - receivedAt.value) : 0 }, 1000)
  }

  watch([serverTime, receivedAt], restart, { immediate: true })
  onBeforeUnmount(() => { if (timer !== undefined) window.clearInterval(timer) })

  const nowMs = computed(() => {
    const baseline = new Date(serverTime.value).getTime()
    return Number.isFinite(baseline) ? baseline + elapsed.value : 0
  })
  const formatted = computed(() => nowMs.value ? new Date(nowMs.value).toLocaleString('zh-CN', { hour12: false }) : '--')

  /** 计算指定结束时间相对服务端时钟的剩余秒数。 */
  function remainingSeconds(endsAt: string | null) {
    if (!endsAt || !nowMs.value) return 0
    const end = new Date(endsAt).getTime()
    return Number.isFinite(end) ? Math.max(0, Math.ceil((end - nowMs.value) / 1000)) : 0
  }

  return { formatted, nowMs, remainingSeconds }
}

/** 将剩余秒数格式化为稳定的时分秒。 */
export function formatRemaining(seconds: number) {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const rest = seconds % 60
  return `${hours}:${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`
}
