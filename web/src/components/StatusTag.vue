<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const typeMap: Record<string, string> = {
  active: 'success',
  success: 'success',
  inactive: 'info',
  failed: 'danger',
}

const tagType = computed(() => typeMap[props.status] || 'info')
</script>

<template>
  <el-tag :type="tagType as any">{{ status }}</el-tag>
</template>
