<script setup>
import { onMounted, ref } from 'vue'
import { adminApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { formatDate, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
const search = ref('')
const ui = useUiStore()

// The levels are ordered: each one implies the ones before it.
const LEVELS = [
  { value: 'UNVERIFIED', label: 'Unverified' },
  { value: 'MOBILE_VERIFIED', label: 'Mobile verified' },
  { value: 'EMAIL_VERIFIED', label: 'Email verified' },
  { value: 'DOCUMENT_VERIFIED', label: 'Documents seen' },
  { value: 'ADMIN_VERIFIED', label: 'Verified by us' },
]

async function setVerification(client, level) {
  if (level === client.verification_level) return
  const confirmed = await ui.confirm({
    title: `Set ${client.company_name} to ${humanise(level)}?`,
    message:
      'This changes the trust badge shown on every listing this account posts, and is recorded in the audit log. The account is told.',
    confirmLabel: 'Set level',
  })
  if (!confirmed) {
    load(meta.value?.page || 1)
    return
  }
  try {
    await adminApi.setClientVerification(client.id, { verification_level: level })
    client.verification_level = level
    ui.success('Verification level updated')
  } catch (err) {
    ui.error(err.message)
    load(meta.value?.page || 1)
  }
}

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await adminApi.clients({ page, per_page: 20, q: search.value })
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
        <h2>Builders</h2>
        <p>{{ meta?.total || 0 }} builder account(s)</p>
      </div>
    </div>

    <div class="toolbar panel">
      <input
        v-model="search"
        class="input"
        type="search"
        placeholder="Search by company name"
        @keyup.enter="load(1)"
      />
      <button class="btn btn-outline" @click="load(1)">Search</button>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="5" />

    <EmptyState v-else-if="!items.length" icon="🏢" title="No builders registered yet" />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Company</th>
            <th>Owner</th>
            <th>Contact</th>
            <th>Company RERA</th>
            <th>Verification</th>
            <th class="num">Projects</th>
            <th>Joined</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="client in items" :key="client.id">
            <td>
              <strong>{{ client.company_name }}</strong>
              <div v-if="client.city" class="tiny muted">{{ client.city }}, {{ client.state }}</div>
            </td>
            <td class="small">{{ client.owner_name || '-' }}</td>
            <td class="small">
              {{ client.contact_email || '-' }}
              <div class="tiny muted">{{ client.contact_phone }}</div>
            </td>
            <td class="tiny mono">{{ client.rera_reg_no || '-' }}</td>
            <td>
              <!-- The badge on this account's listings is what this sets, so it
                   records the outcome of a check made elsewhere. No identity
                   documents are collected or stored here. -->
              <select
                class="select select-sm"
                :value="client.verification_level"
                @change="setVerification(client, $event.target.value)"
              >
                <option v-for="level in LEVELS" :key="level.value" :value="level.value">
                  {{ level.label }}
                </option>
              </select>
              <div class="tiny muted">{{ humanise(client.profile_type) }}</div>
            </td>
            <td class="num">{{ client.total_properties }}</td>
            <td class="tiny">{{ formatDate(client.created_at) }}</td>
            <td>
              <RouterLink
                class="btn btn-ghost btn-sm"
                :to="{ name: 'admin-properties', query: { client_id: client.id } }"
              >
                Projects
              </RouterLink>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>

<style scoped>
.toolbar { display: grid; grid-template-columns: 1fr auto; gap: 10px; padding: 14px; margin-bottom: 16px; }
.mono { font-family: ui-monospace, 'SFMono-Regular', Menlo, monospace; }
.select-sm { padding: 4px 8px; font-size: 0.85rem; }
</style>
