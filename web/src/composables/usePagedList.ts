import { ref, type Ref } from 'vue'
import type { SortOrder } from '../types'

interface SortState<F extends string> {
  sort_field: F | ''
  sort_order: SortOrder
}

// 共用分頁清單狀態：page/total/pageSize/loading + load(silent) try/finally + reloadFromFirstPage + handleSortChange
export function usePagedList<T, F extends string = string>(
  fetchPage: (page: number, pageSize: number) => Promise<{ items: T[]; total: number }>,
) {
  const items: Ref<T[]> = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = 20
  const loading = ref(false)

  async function load(silent = false) {
    if (!silent) loading.value = true
    try {
      const result = await fetchPage(page.value, pageSize)
      items.value = result.items
      total.value = result.total
    } finally {
      if (!silent) loading.value = false
    }
  }

  function reloadFromFirstPage() {
    page.value = 1
    load()
  }

  function handleSortChange(filters: SortState<F>, { prop, order }: { prop: string; order: 'ascending' | 'descending' | null }) {
    filters.sort_field = order ? (prop as F) : ''
    filters.sort_order = order === 'ascending' ? 'asc' : 'desc'
    reloadFromFirstPage()
  }

  return { items, total, page, pageSize, loading, load, reloadFromFirstPage, handleSortChange }
}
