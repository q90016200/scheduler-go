<script setup lang="ts">
// 任務類型（http / rpc）與 method 分色顯示；色票在 style.css 的 --kind-* / --m-*
defineProps<{ type: string; method: string }>()
</script>

<template>
  <span class="kind">
    <span class="kind-type" :class="`t-${type}`">{{ type }}</span>
    <span v-if="type === 'http'" class="kind-method" :class="`m-${method.toLowerCase()}`">{{ method }}</span>
    <span v-else class="kind-rpc" :title="method">{{ method }}</span>
  </span>
</template>

<style scoped>
.kind {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font: 500 11px/1 var(--mono);
}
.kind-type {
  --c: var(--text-muted);
  padding: 3px 6px;
  border-radius: 4px;
  color: var(--c);
  background: color-mix(in oklch, var(--c) 14%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in oklch, var(--c) 30%, transparent);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.t-http {
  --c: var(--kind-http);
}
.t-rpc {
  --c: var(--kind-rpc);
}
.kind-method {
  font-weight: 700;
  color: var(--text-muted);
}
.m-get {
  color: var(--m-get);
}
.m-post {
  color: var(--m-post);
}
.m-put {
  color: var(--m-put);
}
.m-patch {
  color: var(--m-patch);
}
.m-delete {
  color: var(--m-delete);
}
.kind-rpc {
  color: var(--kind-rpc);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
