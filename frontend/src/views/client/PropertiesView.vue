<script setup>
import { onMounted, ref } from 'vue'
import { clientApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { STATUS_VARIANT } from '@/utils/constants'
import { formatDate, formatNumber, formatPriceRange, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const ui = useUiStore()

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
const statusFilter = ref('')
const search = ref('')

// Rentals lapse, and an owner who does not know cannot renew. The panel is
// loaded alongside the table so the warning is on the screen where the fix is.
const expiring = ref([])

const AVAILABILITY = [
  { value: 'AVAILABLE', label: 'Available' },
  { value: 'RESERVED', label: 'Reserved' },
  { value: 'RENTED', label: 'Rented out' },
  { value: 'INACTIVE', label: 'Not listed' },
]

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await clientApi.properties({
      page,
      per_page: 15,
      status: statusFilter.value,
      q: search.value,
    })
    items.value = res.data || []
    meta.value = res.meta || null
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
  loadExpiring()
}

// A failure here must not blank the table: the listings still load, the owner
// simply does not get the warning banner this time.
async function loadExpiring() {
  try {
    const res = await clientApi.expiringListings(14)
    expiring.value = res.data || []
  } catch {
    expiring.value = []
  }
}

// Availability is deliberately not part of the approval workflow: letting a
// flat keeps the listing published, with its history, leads and analytics.
async function changeAvailability(item, status) {
  if (status === item.availability_status) return
  try {
    await clientApi.setAvailability(item.id, { availability_status: status })
    item.availability_status = status
    ui.success(status === 'RENTED' ? 'Marked as rented out' : 'Availability updated')
  } catch (err) {
    ui.error(err.message)
    load(meta.value?.page || 1)
  }
}

async function renew(id) {
  try {
    await clientApi.renewListing(id)
    ui.success('Listing renewed')
    load(meta.value?.page || 1)
  } catch (err) {
    ui.error(err.message)
  }
}

async function submitForReview(item) {
  const confirmed = await ui.confirm({
    title: `Submit ${item.name} for review?`,
    message:
      'An administrator will check the project details and RERA information before it goes live. You cannot edit it while it is under review.',
    confirmLabel: 'Submit',
  })
  if (!confirmed) return
  try {
    await clientApi.submitProperty(item.id)
    ui.success('Submitted for review')
    load(meta.value?.page || 1)
  } catch (err) {
    ui.error(err.message)
  }
}

async function remove(item) {
  const confirmed = await ui.confirm({
    title: `Delete ${item.name}?`,
    message: 'This removes the listing and its media. Leads already received are kept.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!confirmed) return
  try {
    await clientApi.deleteProperty(item.id)
    ui.success('Property deleted')
    load(meta.value?.page || 1)
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
        <h2>My properties</h2>
        <p>{{ meta?.total || 0 }} project(s) across all workflow states</p>
      </div>
      <RouterLink class="btn btn-primary" :to="{ name: 'client-property-new' }">
        Add a property
      </RouterLink>
    </div>

    <div class="toolbar panel">
      <input
        v-model="search"
        class="input"
        type="search"
        placeholder="Search by project name"
        @keyup.enter="load(1)"
      />
      <select v-model="statusFilter" class="select" @change="load(1)">
        <option value="">All statuses</option>
        <option value="DRAFT">Draft</option>
        <option value="SUBMITTED">Submitted</option>
        <option value="UNDER_REVIEW">Under review</option>
        <option value="APPROVED">Approved</option>
        <option value="PUBLISHED">Published</option>
        <option value="REJECTED">Rejected</option>
      </select>
      <button class="btn btn-outline" @click="load(1)">Apply</button>
    </div>

    <div v-if="expiring.length" class="alert alert-warning expiry-panel mb-2">
      <div>
        <strong>{{ expiring.length }} rental listing(s) lapse soon.</strong>
        A lapsed listing stops being shown to renters until you renew it.
      </div>
      <ul class="expiry-list">
        <li v-for="row in expiring" :key="row.id">
          <span>{{ row.name }}</span>
          <span class="tiny muted">
            {{ row.has_expired ? 'expired' : `${row.days_left} day(s) left` }}
          </span>
          <button class="btn btn-outline btn-sm" @click="renew(row.id)">Renew</button>
        </li>
      </ul>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="5" />

    <EmptyState
      v-else-if="!items.length"
      icon="🏗️"
      title="No projects yet"
      message="Add your first project, upload photos and submit it for review."
    >
      <RouterLink class="btn btn-primary" :to="{ name: 'client-property-new' }">
        Add a property
      </RouterLink>
    </EmptyState>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Project</th>
            <th>Location</th>
            <th class="num">Price</th>
            <th>Status</th>
            <th>Availability</th>
            <th class="num">Views</th>
            <th>Updated</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id">
            <td>
              <strong>{{ item.name }}</strong>
              <div class="tiny muted">{{ humanise(item.property_type) }}</div>
            </td>
            <td class="small">{{ item.locality }}, {{ item.city }}</td>
            <td class="num">{{ formatPriceRange(item.min_price, item.max_price) }}</td>
            <td>
              <span class="badge" :class="`badge-${STATUS_VARIANT[item.property_status] || 'neutral'}`">
                {{ humanise(item.property_status) }}
              </span>
            </td>
            <td>
              <select
                v-if="item.purpose === 'RENT'"
                class="select select-sm"
                :value="item.availability_status"
                @change="changeAvailability(item, $event.target.value)"
              >
                <option v-for="opt in AVAILABILITY" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
              <!-- A flat for sale has no availability to manage. -->
              <span v-else class="tiny muted">—</span>
            </td>
            <td class="num">{{ formatNumber(item.view_count) }}</td>
            <td class="tiny">{{ formatDate(item.created_at) }}</td>
            <td>
              <div class="actions">
                <RouterLink
                  class="btn btn-ghost btn-sm"
                  :to="{ name: 'client-property-edit', params: { id: item.id } }"
                >
                  Edit
                </RouterLink>
                <RouterLink
                  class="btn btn-ghost btn-sm"
                  :to="{ name: 'client-property-analytics', params: { id: item.id } }"
                >
                  Stats
                </RouterLink>
                <button
                  v-if="['DRAFT', 'REJECTED'].includes(item.property_status)"
                  class="btn btn-primary btn-sm"
                  @click="submitForReview(item)"
                >
                  Submit
                </button>
                <button
                  v-if="item.purpose === 'RENT'"
                  class="btn btn-ghost btn-sm"
                  title="Confirm the flat is still free, without going back through review"
                  @click="renew(item.id)"
                >
                  Renew
                </button>
                <button class="btn btn-ghost btn-sm danger" @click="remove(item)">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>

<style scoped>
.toolbar {
  display: grid;
  grid-template-columns: 1fr 200px auto;
  gap: 10px;
  padding: 14px;
  margin-bottom: 16px;
}

.actions { display: flex; gap: 2px; flex-wrap: wrap; }
.danger { color: var(--danger-600); }

.expiry-panel { display: grid; gap: 8px; }
.expiry-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 6px; }
.expiry-list li {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: space-between;
}
.select-sm { padding: 4px 8px; font-size: 0.85rem; }

@media (max-width: 700px) {
  .toolbar { grid-template-columns: 1fr; }
}
</style>
