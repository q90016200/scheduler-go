<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const typeMap: Record<string, string> = {
  enabled: 'success',
  success: 'success',
  disabled: 'info',
  failed: 'danger',
}

const labelMap: Record<string, string> = {
  enabled: '啟用',
  disabled: '停用',
  success: '成功',
  failed: '失敗',
}

const tagType = computed(() => typeMap[props.status] || 'info')
const label = computed(() => labelMap[props.status] ?? props.status)
</script>

<template>
  <el-tag :type="tagType as any">{{ label }}</el-tag>
</template>
