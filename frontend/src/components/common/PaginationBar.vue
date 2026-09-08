<script setup>
import { computed } from 'vue'

const props = defineProps({
  meta: { type: Object, default: null },
})
const emit = defineEmits(['change'])

const page = computed(() => props.meta?.page || 1)
const totalPages = computed(() => props.meta?.total_pages || 1)

/**
 * A compact window around the current page with ellipses, so a 40-page result
 * set does not render 40 buttons.
 */
const pages = computed(() => {
  const total = totalPages.value
  const current = page.value
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)

  const items = [1]
  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)

  if (start > 2) items.push('...')
  for (let i = start; i <= end; i++) items.push(i)
  if (end < total - 1) items.push('...')
  items.push(total)
  return items
})

function go(target) {
  if (target === '...' || target === page.value || target < 1 || target > totalPages.value) return
  emit('change', target)
}
</script>

<template>
  <nav v-if="meta && totalPages > 1" class="pagination" aria-label="Pagination">
    <button class="btn btn-outline btn-sm" :disabled="page <= 1" @click="go(page - 1)">
      Previous
    </button>

    <div class="pages">
      <button
        v-for="(item, index) in pages"
        :key="`${item}-${index}`"
        class="page-btn"
        :class="{ active: item === page, gap: item === '...' }"
        :disabled="item === '...'"
        :aria-current="item === page ? 'page' : undefined"
        @click="go(item)"
      >
        {{ item }}
      </button>
    </div>

    <button class="btn btn-outline btn-sm" :disabled="page >= totalPages" @click="go(page + 1)">
      Next
    </button>
  </nav>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-top: 28px;
  flex-wrap: wrap;
}

.pages { display: flex; gap: 4px; }

.page-btn {
  min-width: 34px;
  height: 34px;
  padding: 0 8px;
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-sm);
  background: var(--white);
  color: var(--ink-700);
  font-size: 0.86rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition);
}
.page-btn:hover:not(:disabled):not(.active) { border-color: var(--brand-500); color: var(--brand-700); }
.page-btn.active { background: var(--brand-700); border-color: var(--brand-700); color: var(--white); }
.page-btn.gap { border: 0; background: none; cursor: default; }
</style>
