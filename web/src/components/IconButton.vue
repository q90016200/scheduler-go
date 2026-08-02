<script setup lang="ts">
import type { Component } from 'vue'

withDefaults(
  defineProps<{
    icon: Component
    label: string
    type?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
    size?: 'small' | 'default' | 'large'
    loading?: boolean
    disabled?: boolean
  }>(),
  { size: 'small' },
)

defineEmits<{ click: [MouseEvent] }>()
</script>

<template>
  <el-tooltip :content="label" placement="top">
    <el-button
      circle
      :icon="icon"
      :type="type"
      :size="size"
      :loading="loading"
      :disabled="disabled"
      :aria-label="label"
      @click="$emit('click', $event)"
    />
  </el-tooltip>
</template>
