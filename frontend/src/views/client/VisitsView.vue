<script setup>
import { onMounted, reactive, ref } from 'vue'
import { clientApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { formatDate, formatDateTime, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const ui = useUiStore()

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
const status = ref('')
const responding = ref(null)

const TONE = {
  REQUESTED: 'info',
  CONFIRMED: 'success',
  RESCHEDULED: 'warning',
  COMPLETED: 'neutral',
  CANCELLED: 'danger',
}

const SLOTS = ['MORNING', 'AFTERNOON', 'EVENING', 'ANY']

// The reply form defaults to the date the visitor asked for, since agreeing is
// the common case and retyping it is friction.
const reply = reactive({ scheduled_date: '', time_slot: 'EVENING', remarks: '' })

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await clientApi.visits({ page, per_page: 20, status: status.value })
    items.value = res.data || []
    meta.value = res.meta || null
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

function openReply(visit) {
  responding.value = visit.id
  reply.scheduled_date = visit.scheduled_at || visit.requested_at || ''
  reply.time_slot = visit.time_slot || 'EVENING'
  reply.remarks = ''
}

async function respond(visit, nextStatus) {
  try {
    await clientApi.respondToVisit(visit.id, {
      status: nextStatus,
      scheduled_date: reply.scheduled_date,
      time_slot: reply.time_slot,
      remarks: reply.remarks,
    })
    ui.success(`Visit ${humanise(nextStatus).toLowerCase()}`)
    responding.value = null
    load(meta.value?.page || 1)
  } catch (err) {
    ui.error(err.message)
  }
}

async function close(visit, nextStatus) {
  try {
    await clientApi.respondToVisit(visit.id, { status: nextStatus })
    ui.success(`Visit ${humanise(nextStatus).toLowerCase()}`)
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
        <h2>Visit requests</h2>
        <p>{{ meta?.total || 0 }} request(s) to view your properties</p>
      </div>
    </div>

    <div class="toolbar panel">
      <select v-model="status" class="select" @change="load(1)">
        <option value="">All statuses</option>
        <option v-for="value in Object.keys(TONE)" :key="value" :value="value">
          {{ humanise(value) }}
        </option>
      </select>
      <button class="btn btn-outline" @click="load(1)">Refresh</button>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="4" />

    <EmptyState
      v-else-if="!items.length"
      icon="📅"
      title="No visit requests yet"
      message="When someone asks to view one of your properties, it appears here."
    />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Visitor</th>
            <th>Property</th>
            <th>Asked for</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <template v-for="visit in items" :key="visit.id">
            <tr>
              <td>
                <strong>{{ visit.name }}</strong>
                <div class="tiny muted">{{ visit.mobile }}</div>
              </td>
              <td class="small">{{ visit.property?.name || '—' }}</td>
              <td class="small">
                {{ formatDate(visit.requested_at) }}
                <div class="tiny muted">{{ humanise(visit.time_slot) }}</div>
              </td>
              <td>
                <span class="badge" :class="`badge-${TONE[visit.status] || 'neutral'}`">
                  {{ humanise(visit.status) }}
                </span>
              </td>
              <td>
                <div class="actions">
                  <a class="btn btn-whatsapp btn-sm" :href="`https://wa.me/91${visit.mobile}`"
                     target="_blank" rel="noopener">WhatsApp</a>
                  <a class="btn btn-outline btn-sm" :href="`tel:${visit.mobile}`">Call</a>
                  <button
                    v-if="!['COMPLETED', 'CANCELLED'].includes(visit.status)"
                    class="btn btn-primary btn-sm"
                    @click="openReply(visit)"
                  >
                    Respond
                  </button>
                </div>
              </td>
            </tr>

            <tr v-if="responding === visit.id" class="reply-row">
              <td colspan="5">
                <div class="reply">
                  <p v-if="visit.message" class="small mb-2">
                    <strong>They said:</strong> {{ visit.message }}
                  </p>

                  <div class="reply-grid">
                    <label class="field mb-0">
                      <span class="label">Confirm for</span>
                      <input v-model="reply.scheduled_date" class="input" type="date" />
                    </label>
                    <label class="field mb-0">
                      <span class="label">Time</span>
                      <select v-model="reply.time_slot" class="select">
                        <option v-for="slot in SLOTS" :key="slot" :value="slot">
                          {{ humanise(slot) }}
                        </option>
                      </select>
                    </label>
                    <label class="field mb-0 grow">
                      <span class="label">Note for the visitor</span>
                      <input
                        v-model="reply.remarks"
                        class="input"
                        placeholder="For example, please call when you reach the gate."
                      />
                    </label>
                  </div>

                  <div class="row-wrap mt-2">
                    <button class="btn btn-primary btn-sm" @click="respond(visit, 'CONFIRMED')">
                      Confirm
                    </button>
                    <button class="btn btn-outline btn-sm" @click="respond(visit, 'RESCHEDULED')">
                      Propose another time
                    </button>
                    <button class="btn btn-ghost btn-sm" @click="close(visit, 'COMPLETED')">
                      Mark completed
                    </button>
                    <button class="btn btn-ghost btn-sm danger" @click="close(visit, 'CANCELLED')">
                      Cancel
                    </button>
                    <button class="btn btn-ghost btn-sm" @click="responding = null">Close</button>
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
.toolbar { display: grid; grid-template-columns: 220px auto; gap: 10px; padding: 14px; margin-bottom: 16px; }
.actions { display: flex; gap: 3px; flex-wrap: wrap; }
.danger { color: var(--danger-600); }

.reply-row td { background: var(--ink-50); }
.reply { padding: 6px 0; }
.reply-grid { display: flex; gap: 12px; flex-wrap: wrap; align-items: flex-end; }
.reply-grid .field { min-width: 160px; }

@media (max-width: 700px) {
  .toolbar { grid-template-columns: 1fr; }
}
</style>
