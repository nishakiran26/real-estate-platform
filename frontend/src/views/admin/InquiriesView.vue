<script setup>
import { onMounted, ref } from 'vue'
import { adminApi } from '@/services/api'
import { INQUIRY_STATUS_VARIANT } from '@/utils/constants'
import { formatDateTime, formatPrice, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
const status = ref('')
const search = ref('')

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await adminApi.inquiries({
      page,
      per_page: 25,
      status: status.value,
      q: search.value,
    })
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
        <h2>Leads</h2>
        <p>{{ meta?.total || 0 }} inquiry(ies) across all builders</p>
      </div>
    </div>

    <div class="toolbar panel">
      <input
        v-model="search"
        class="input"
        type="search"
        placeholder="Search by buyer name, mobile or project"
        @keyup.enter="load(1)"
      />
      <select v-model="status" class="select" @change="load(1)">
        <option value="">All statuses</option>
        <option value="NEW">New</option>
        <option value="CONTACTED">Contacted</option>
        <option value="SITE_VISIT_SCHEDULED">Site visit scheduled</option>
        <option value="NEGOTIATING">Negotiating</option>
        <option value="CLOSED_WON">Closed - won</option>
        <option value="CLOSED_LOST">Closed - lost</option>
      </select>
      <button class="btn btn-outline" @click="load(1)">Apply</button>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="6" />

    <EmptyState v-else-if="!items.length" icon="✉️" title="No inquiries match this filter" />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Buyer</th>
            <th>Project</th>
            <th>Interest</th>
            <th>Source</th>
            <th>Received</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="lead in items" :key="lead.id">
            <td>
              <strong>{{ lead.name }}</strong>
              <!-- Contact details stay visible to administrators only; the
                   builder sees them for their own leads in their portal. -->
              <div class="tiny muted">{{ lead.mobile }}</div>
            </td>
            <td class="small">{{ lead.property?.name || '-' }}</td>
            <td class="small">
              {{ lead.configuration || 'Any' }}
              <div v-if="lead.budget" class="tiny muted">{{ formatPrice(lead.budget) }}</div>
            </td>
            <td class="tiny">{{ lead.source }}</td>
            <td class="tiny">{{ formatDateTime(lead.created_at) }}</td>
            <td>
              <span class="badge" :class="`badge-${INQUIRY_STATUS_VARIANT[lead.status] || 'neutral'}`">
                {{ humanise(lead.status) }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>

<style scoped>
.toolbar { display: grid; grid-template-columns: 1fr 210px auto; gap: 10px; padding: 14px; margin-bottom: 16px; }

@media (max-width: 700px) {
  .toolbar { grid-template-columns: 1fr; }
}
</style>
