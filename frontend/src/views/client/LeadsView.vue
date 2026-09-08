<script setup>
import { onMounted, ref } from 'vue'
import { clientApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { INQUIRY_STATUS_VARIANT } from '@/utils/constants'
import { formatDateTime, formatPrice, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const ui = useUiStore()

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
const status = ref('')
const search = ref('')
const expanded = ref(null)

const STATUSES = [
  'NEW', 'CONTACTED', 'SITE_VISIT_SCHEDULED', 'NEGOTIATING', 'CLOSED_WON', 'CLOSED_LOST',
]

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await clientApi.inquiries({
      page,
      per_page: 20,
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

async function updateStatus(lead, next) {
  try {
    await clientApi.updateInquiryStatus(lead.id, { status: next, remarks: '' })
    lead.status = next
    ui.success('Lead updated')
  } catch (err) {
    ui.error(err.message)
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
        <p>{{ meta?.total || 0 }} inquiry(ies) on your projects</p>
      </div>
    </div>

    <div class="toolbar panel">
      <input
        v-model="search"
        class="input"
        type="search"
        placeholder="Search by name, mobile or project"
        @keyup.enter="load(1)"
      />
      <select v-model="status" class="select" @change="load(1)">
        <option value="">All statuses</option>
        <option v-for="value in STATUSES" :key="value" :value="value">
          {{ humanise(value) }}
        </option>
      </select>
      <button class="btn btn-outline" @click="load(1)">Apply</button>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="5" />

    <EmptyState
      v-else-if="!items.length"
      icon="✉️"
      title="No leads yet"
      message="Inquiries raised on your published projects will appear here."
    />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Buyer</th>
            <th>Project</th>
            <th>Interest</th>
            <th>Received</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <template v-for="lead in items" :key="lead.id">
            <tr>
              <td>
                <strong>{{ lead.name }}</strong>
                <div class="tiny muted">{{ lead.mobile }}</div>
              </td>
              <td class="small">{{ lead.property?.name || '-' }}</td>
              <td class="small">
                {{ lead.configuration || 'Any' }}
                <div v-if="lead.budget" class="tiny muted">
                  Budget {{ formatPrice(lead.budget) }}
                </div>
              </td>
              <td class="tiny">{{ formatDateTime(lead.created_at) }}</td>
              <td>
                <select
                  class="select status-select"
                  :value="lead.status"
                  @change="updateStatus(lead, $event.target.value)"
                >
                  <option v-for="value in STATUSES" :key="value" :value="value">
                    {{ humanise(value) }}
                  </option>
                </select>
              </td>
              <td>
                <button
                  class="btn btn-ghost btn-sm"
                  @click="expanded = expanded === lead.id ? null : lead.id"
                >
                  {{ expanded === lead.id ? 'Hide' : 'Details' }}
                </button>
              </td>
            </tr>
            <tr v-if="expanded === lead.id" class="detail-row">
              <td colspan="6">
                <div class="lead-detail">
                  <div>
                    <span class="stat-label">Contact</span>
                    <p class="small mb-0">
                      <a :href="`tel:${lead.mobile}`">{{ lead.mobile }}</a>
                      <template v-if="lead.email">
                        · <a :href="`mailto:${lead.email}`">{{ lead.email }}</a>
                      </template>
                    </p>
                  </div>
                  <div>
                    <span class="stat-label">Preferred time</span>
                    <p class="small mb-0">{{ humanise(lead.preferred_time) || 'Any time' }}</p>
                  </div>
                  <div>
                    <span class="stat-label">Source</span>
                    <p class="small mb-0">{{ lead.source }}</p>
                  </div>
                  <div class="wide">
                    <span class="stat-label">Message</span>
                    <p class="small mb-0">{{ lead.message || 'No message provided.' }}</p>
                  </div>
                  <div class="wide row">
                    <a class="btn btn-whatsapp btn-sm" :href="`https://wa.me/91${lead.mobile}`" target="_blank" rel="noopener">
                      WhatsApp
                    </a>
                    <a class="btn btn-outline btn-sm" :href="`tel:${lead.mobile}`">Call</a>
                    <span class="badge" :class="`badge-${INQUIRY_STATUS_VARIANT[lead.status] || 'neutral'}`">
                      {{ humanise(lead.status) }}
                    </span>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>

<style scoped>
.toolbar { display: grid; grid-template-columns: 1fr 210px auto; gap: 10px; padding: 14px; margin-bottom: 16px; }

.status-select { min-width: 165px; padding-block: 6px; font-size: 0.82rem; }

.detail-row td { background: var(--ink-50); }
.lead-detail { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 16px; padding: 6px 0; }
.lead-detail .wide { grid-column: 1 / -1; }

@media (max-width: 700px) {
  .toolbar { grid-template-columns: 1fr; }
}
</style>
