<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  icon: Component
  label: string
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  loading?: boolean
}>()

defineEmits<{ click: [MouseEvent] }>()
</script>

<template>
  <el-tooltip :content="label" placement="top">
    <el-button
      circle
      :icon="icon"
      :type="type"
      size="small"
      :loading="loading"
      :aria-label="label"
      @click="$emit('click', $event)"
    />
  </el-tooltip>
</template>
