<script setup>
import { onMounted, ref } from 'vue'
import { adminApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { formatDateTime, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const ui = useUiStore()

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
// Open first: this is a work queue, and the work is the complaints nobody has
// answered yet.
const status = ref('OPEN')
const reason = ref('')

const REPORT_VARIANT = {
  OPEN: 'warning',
  REVIEWING: 'info',
  ACTIONED: 'success',
  DISMISSED: 'neutral',
}

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await adminApi.reports({
      page,
      per_page: 25,
      status: status.value,
      reason: reason.value,
    })
    items.value = res.data || []
    meta.value = res.meta || null
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

// The note is typed on the row rather than in the dialog, because it is what
// the owner reads when their listing is taken down and it deserves more than a
// one-line prompt.
const notes = ref({})

async function resolve(item, newStatus, unpublish = false) {
  const confirmed = await ui.confirm({
    title: unpublish ? `Take down ${item.property?.name}?` : `Mark as ${humanise(newStatus)}?`,
    message: unpublish
      ? 'The listing goes back to the owner as rejected, with your note as the reason, so they can see what was wrong and fix it. Their leads and analytics are kept.'
      : 'This records your decision on the report. The listing is not changed.',
    confirmLabel: unpublish ? 'Take it down' : 'Confirm',
    danger: unpublish,
  })
  if (!confirmed) return
  try {
    await adminApi.resolveReport(item.id, {
      status: newStatus,
      action_note: notes.value[item.id] || '',
      unpublish,
    })
    delete notes.value[item.id]
    ui.success('Report updated')
    load(meta.value?.page || 1)
  } catch (err) {
    ui.error(err.message)
  }
}

async function sweep() {
  try {
    const res = await adminApi.sweepExpiredListings()
    const { expired = 0, warned = 0 } = res.data || {}
    ui.success(`${expired} listing(s) retired, ${warned} owner(s) warned`)
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
        <h2>Reported listings</h2>
        <p>{{ meta?.total || 0 }} report(s) — oldest first, because that is the one that has waited longest</p>
      </div>
      <button
        class="btn btn-outline"
        title="Retire rentals that have lapsed and warn owners whose listings are about to"
        @click="sweep"
      >
        Run expiry sweep
      </button>
    </div>

    <div class="toolbar panel">
      <select v-model="status" class="select" @change="load(1)">
        <option value="">All statuses</option>
        <option value="OPEN">Open</option>
        <option value="REVIEWING">Reviewing</option>
        <option value="ACTIONED">Actioned</option>
        <option value="DISMISSED">Dismissed</option>
      </select>
      <select v-model="reason" class="select" @change="load(1)">
        <option value="">All reasons</option>
        <option value="ALREADY_RENTED">Already let or sold</option>
        <option value="INCORRECT_PRICE">Wrong price</option>
        <option value="INCORRECT_PHOTOS">Wrong photos</option>
        <option value="INCORRECT_LOCATION">Wrong location</option>
        <option value="DUPLICATE">Duplicate</option>
        <option value="FAKE_LISTING">Does not exist</option>
        <option value="FRAUD">Fraud</option>
        <option value="OTHER">Other</option>
      </select>
      <button class="btn btn-outline" @click="load(1)">Apply</button>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="5" />

    <EmptyState
      v-else-if="!items.length"
      icon="🛡️"
      title="Nothing reported"
      message="No listings are waiting on a moderation decision."
    />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Listing</th>
            <th>Reason</th>
            <th>Details</th>
            <th>Status</th>
            <th>Reported</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id">
            <td>
              <RouterLink
                v-if="item.property"
                :to="{ name: 'admin-property-detail', params: { id: item.property_id } }"
              >
                {{ item.property.name }}
              </RouterLink>
              <span v-else class="muted">Listing #{{ item.property_id }}</span>
              <!-- A listing reported repeatedly deserves attention before one
                   reported once, so the count is on the row itself. -->
              <div v-if="item.open_reports > 1" class="tiny warn">
                {{ item.open_reports }} open reports on this listing
              </div>
            </td>
            <td class="small">{{ humanise(item.reason) }}</td>
            <td class="small details">{{ item.details || '—' }}</td>
            <td>
              <span class="badge" :class="`badge-${REPORT_VARIANT[item.status] || 'neutral'}`">
                {{ humanise(item.status) }}
              </span>
            </td>
            <td class="tiny">{{ formatDateTime(item.created_at) }}</td>
            <td>
              <input
                v-if="!['ACTIONED', 'DISMISSED'].includes(item.status)"
                v-model="notes[item.id]"
                class="input note"
                placeholder="Note for the owner"
              />
              <p v-else-if="item.action_note" class="tiny muted mb-0">{{ item.action_note }}</p>
              <div class="actions">
                <button
                  v-if="item.status === 'OPEN'"
                  class="btn btn-ghost btn-sm"
                  @click="resolve(item, 'REVIEWING')"
                >
                  Looking into it
                </button>
                <button
                  v-if="item.status !== 'ACTIONED'"
                  class="btn btn-primary btn-sm"
                  @click="resolve(item, 'ACTIONED', true)"
                >
                  Uphold and take down
                </button>
                <button
                  v-if="item.status !== 'DISMISSED'"
                  class="btn btn-ghost btn-sm"
                  @click="resolve(item, 'DISMISSED')"
                >
                  Dismiss
                </button>
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
  grid-template-columns: 200px 220px auto;
  gap: 10px;
  padding: 14px;
  margin-bottom: 16px;
}

.actions { display: flex; gap: 2px; flex-wrap: wrap; }
.details { max-width: 320px; }
.note { min-width: 190px; margin-bottom: 6px; font-size: 0.85rem; padding: 5px 8px; }
.warn { color: var(--warning-600); }

@media (max-width: 700px) {
  .toolbar { grid-template-columns: 1fr; }
}
</style>
