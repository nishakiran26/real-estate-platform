<script setup>
import { onMounted, ref } from 'vue'
import { adminApi } from '@/services/api'
import { formatDateTime, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')

const ACTION_TONE = {
  PROPERTY_APPROVED: 'success',
  PROPERTY_PUBLISHED: 'success',
  PROPERTY_REJECTED: 'danger',
  USER_SUSPENDED: 'danger',
  CLIENT_SUSPENDED: 'danger',
  RERA_VERIFIED: 'success',
  FEATURED_CHANGED: 'info',
}

async function load(page = 1) {
  loading.value = true
  try {
    const res = await adminApi.auditLogs({ page, per_page: 50 })
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
        <h2>Audit log</h2>
        <p>Approvals, rejections, suspensions and content changes</p>
      </div>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="8" />

    <EmptyState
      v-else-if="!items.length"
      icon="📜"
      title="No administrative actions recorded yet"
    />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>When</th>
            <th>Action</th>
            <th>Entity</th>
            <th>Actor</th>
            <th>Details</th>
            <th>IP</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="entry in items" :key="entry.id">
            <td class="tiny">{{ formatDateTime(entry.created_at) }}</td>
            <td>
              <span class="badge" :class="`badge-${ACTION_TONE[entry.action] || 'neutral'}`">
                {{ humanise(entry.action) }}
              </span>
            </td>
            <td class="small">{{ entry.entity }} #{{ entry.entity_id }}</td>
            <td class="small">User #{{ entry.actor_id }}</td>
            <td class="tiny muted details">{{ entry.details }}</td>
            <td class="tiny muted">{{ entry.ip_address }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>

<style scoped>
.details { max-width: 320px; word-break: break-word; }
</style>
