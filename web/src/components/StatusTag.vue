<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const statusMap: Record<string, [string, string]> = {
  enabled: ['success', '啟用'],
  disabled: ['info', '停用'],
  success: ['success', '成功'],
  failed: ['danger', '失敗'],
}

const tagType = computed(() => statusMap[props.status]?.[0] || 'info')
const label = computed(() => statusMap[props.status]?.[1] ?? props.status)
</script>

<template>
  <el-tag :type="tagType as any">{{ label }}</el-tag>
</template>
