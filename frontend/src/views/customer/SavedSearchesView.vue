<script setup>
import { onMounted, ref } from 'vue'
import { customerApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { formatPrice } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'

const ui = useUiStore()

const items = ref([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await customerApi.rentalAlerts()
    items.value = res.data || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

// Pausing keeps the search. Someone who has tuned a rent band and a locality
// should not have to rebuild it just to stop the notices for a fortnight.
async function toggle(item) {
  try {
    await customerApi.setRentalAlertActive(item.id, !item.is_active)
    item.is_active = !item.is_active
    ui.success(item.is_active ? 'Alerts resumed' : 'Alerts paused')
  } catch (err) {
    ui.error(err.message)
    load()
  }
}

async function remove(item) {
  const confirmed = await ui.confirm({
    title: `Delete "${item.name}"?`,
    message: 'You will stop hearing about flats that match this search.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!confirmed) return
  try {
    await customerApi.deleteRentalAlert(item.id)
    items.value = items.value.filter((row) => row.id !== item.id)
    ui.success('Saved search deleted')
  } catch (err) {
    ui.error(err.message)
  }
}

// The band a renter saved, written the way they entered it.
function budget(item) {
  if (item.min_rent && item.max_rent) {
    return `${formatPrice(item.min_rent)} – ${formatPrice(item.max_rent)}`
  }
  if (item.max_rent) return `Up to ${formatPrice(item.max_rent)}`
  if (item.min_rent) return `From ${formatPrice(item.min_rent)}`
  return 'Any rent'
}

onMounted(load)
</script>

<template>
  <section>
    <div class="head mb-2">
      <div>
        <h1 class="h4 mb-1">Saved searches</h1>
        <p class="small muted mb-0">
          We will tell you when a flat matching one of these is listed.
        </p>
      </div>
      <RouterLink class="btn btn-outline" :to="{ name: 'rent' }">Search rentals</RouterLink>
    </div>

    <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

    <SkeletonCard v-if="loading" variant="row" :count="3" />

    <EmptyState
      v-else-if="!items.length"
      icon="🔔"
      title="No saved searches yet"
      message="Run a rent search you like, then save it. We will watch for matching flats so you do not have to."
    >
      <RouterLink class="btn btn-primary" :to="{ name: 'rent' }">Search rentals</RouterLink>
    </EmptyState>

    <ul v-else class="alert-list">
      <li v-for="item in items" :key="item.id" class="alert-row panel">
        <div class="alert-main">
          <div class="alert-title">
            <RouterLink :to="`/rent?${item.search_query}`" class="name">{{ item.name }}</RouterLink>
            <span v-if="!item.is_active" class="badge badge-muted">Paused</span>
          </div>
          <p class="tiny muted mb-0">
            {{ budget(item) }}
            <span v-if="item.bhk"> · {{ item.bhk }} BHK</span>
            <span v-if="item.match_count">
              · {{ item.match_count }} match(es) so far
            </span>
            <span v-else> · nothing matched yet</span>
          </p>
        </div>
        <div class="alert-actions">
          <button class="btn btn-ghost btn-sm" @click="toggle(item)">
            {{ item.is_active ? 'Pause' : 'Resume' }}
          </button>
          <button class="btn btn-ghost btn-sm danger" @click="remove(item)">Delete</button>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.alert-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 10px;
}

.alert-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  flex-wrap: wrap;
}

.alert-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}

.name {
  font-weight: 700;
  color: var(--ink-800);
  text-decoration: none;
}
.name:hover { color: var(--primary-600); }

.alert-actions {
  display: flex;
  gap: 6px;
}
</style>
