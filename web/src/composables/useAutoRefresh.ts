import { onBeforeUnmount, ref, watch } from 'vue'

const STORAGE_KEY = 'auto-refresh-enabled'

export const autoRefreshEnabled = ref(localStorage.getItem(STORAGE_KEY) !== 'false')

watch(autoRefreshEnabled, (enabled) => {
  localStorage.setItem(STORAGE_KEY, String(enabled))
})

// 靜默刷新：呼叫傳入的 loadFn，不動呼叫端自己的 loading ref，避免自動刷新時整頁閃爍
export function useAutoRefresh(loadFn: () => unknown, intervalMs: number) {
  let timer: ReturnType<typeof setInterval> | undefined

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = undefined
    }
  }

  function start() {
    stop()
    if (!autoRefreshEnabled.value) return
    timer = setInterval(loadFn, intervalMs)
  }

  start()
  watch(autoRefreshEnabled, start)
  onBeforeUnmount(stop)
}
