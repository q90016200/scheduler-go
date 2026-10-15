<script setup lang="ts">
// 任務名稱一律帶短 ID（xid 末 6 碼，同一時段建立的任務在這段最不容易撞），避免同名任務看錯；
// 完整 ID 在 hover 的 title 上
import { shortId } from '../utils/id'
defineProps<{ id: string; name?: string | null; group?: string | null; duplicate?: boolean }>()
</script>

<template>
  <div class="task-name">
    <div class="line">
      <router-link :to="`/tasks/${id}`" class="name">{{ name || '(已刪除的任務)' }}</router-link>
      <span class="short-id" :title="id">{{ shortId(id) }}</span>
      <el-tooltip v-if="duplicate" content="這一頁有其他同名任務，請用 # 後的 ID 或群組分辨" placement="top">
        <el-tag size="small" type="warning" effect="plain" round>同名</el-tag>
      </el-tooltip>
    </div>
    <div v-if="group" class="group">{{ group }}</div>
  </div>
</template>

<style scoped>
.task-name {
  display: grid;
  gap: 1px;
  min-width: 0;
}
.line {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
}
.name {
  color: var(--text);
  font-weight: 500;
  text-decoration: none;
}
.name:hover,
.name:focus-visible {
  color: var(--accent);
  text-decoration: underline;
}
.short-id {
  font: 11px var(--mono);
  color: var(--text-muted);
}
.group {
  font-size: 12px;
  color: var(--text-muted);
}
</style>
