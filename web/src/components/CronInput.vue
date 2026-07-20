<script setup lang="ts">
import { computed } from 'vue'
import cronstrue from 'cronstrue'
import 'cronstrue/locales/zh_TW'

const model = defineModel<string>({ required: true })

const presets = [
  { label: '每分鐘', value: '* * * * *' },
  { label: '每小時整點', value: '0 * * * *' },
  { label: '每天 00:00', value: '0 0 * * *' },
  { label: '每週一 00:00', value: '0 0 * * 1' },
  { label: '每月1號 00:00', value: '0 0 1 * *' },
]

const readable = computed(() => {
  if (!model.value.trim()) return ''
  try {
    return cronstrue.toString(model.value, { locale: 'zh_TW' })
  } catch {
    return '格式看起來不像合法的 cron 表達式（分 時 日 月 週）'
  }
})
</script>

<template>
  <div class="cron-input">
    <div class="cron-presets">
      <el-button
        v-for="preset in presets"
        :key="preset.value"
        size="small"
        @click="model = preset.value"
      >
        {{ preset.label }}
      </el-button>
    </div>
    <el-input v-model="model" placeholder="分 時 日 月 週，例如 0 0 * * *" />
    <div class="cron-readable">{{ readable }}</div>
  </div>
</template>

<style scoped>
.cron-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cron-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.cron-readable {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
