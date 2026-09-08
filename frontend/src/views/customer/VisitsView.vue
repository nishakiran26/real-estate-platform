<script setup>
import { onMounted, ref } from 'vue'
import { customerApi } from '@/services/api'
import { formatDate, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')

// Status colour carries the outcome at a glance, so the list is scannable
// without reading every row.
const TONE = {
  REQUESTED: 'info',
  CONFIRMED: 'success',
  RESCHEDULED: 'warning',
  COMPLETED: 'neutral',
  CANCELLED: 'danger',
}

async function load(page = 1) {
  loading.value = true
  try {
    const res = await customerApi.visits({ page, per_page: 20 })
    items.value = res.data || []
    meta.value = res.meta || null
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(() => load())
</script>

<template>
  <div>
    <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

    <div class="section-head">
      <div>
        <h2>My visits</h2>
        <p>Viewings you have asked for, and what the owner said</p>
      </div>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="4" />

    <EmptyState
      v-else-if="!items.length"
      icon="📅"
      title="No visits requested yet"
      message="When you ask to view a property, the request and the owner's reply appear here."
    >
      <RouterLink class="btn btn-primary" :to="{ name: 'rent' }">Browse rentals</RouterLink>
    </EmptyState>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Property</th>
            <th>You asked for</th>
            <th>Confirmed for</th>
            <th>Status</th>
            <th>Owner's note</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="visit in items" :key="visit.id">
            <td>
              <RouterLink
                v-if="visit.property"
                :to="{ name: 'property-detail', params: { slug: visit.property.slug } }"
              >
                <strong>{{ visit.property.name }}</strong>
              </RouterLink>
              <span v-else class="muted">Listing removed</span>
            </td>
            <td class="small">
              {{ formatDate(visit.requested_at) }}
              <div class="tiny muted">{{ humanise(visit.time_slot) }}</div>
            </td>
            <td class="small">
              <template v-if="visit.scheduled_at">{{ formatDate(visit.scheduled_at) }}</template>
              <span v-else class="muted">Awaiting the owner</span>
            </td>
            <td>
              <span class="badge" :class="`badge-${TONE[visit.status] || 'neutral'}`">
                {{ humanise(visit.status) }}
              </span>
            </td>
            <td class="tiny muted">{{ visit.remarks || '—' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>
