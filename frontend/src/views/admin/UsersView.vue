<script setup>
import { onMounted, ref } from 'vue'
import { adminApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { formatDate, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const ui = useUiStore()

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
const role = ref('')
const status = ref('')
const search = ref('')

const STATUS_TONE = { ACTIVE: 'success', INACTIVE: 'neutral', SUSPENDED: 'danger' }

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await adminApi.users({
      page,
      per_page: 20,
      role: role.value,
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

async function setStatus(user, next) {
  const confirmed = await ui.confirm({
    title: `Set ${user.full_name} to ${humanise(next)}?`,
    message:
      next === 'SUSPENDED'
        ? 'A suspended account cannot sign in. Their listings stay as they are.'
        : 'The account will be able to sign in again.',
    confirmLabel: 'Confirm',
    danger: next === 'SUSPENDED',
  })
  if (!confirmed) return

  try {
    await adminApi.updateUserStatus(user.id, { status: next, reason: '' })
    user.status = next
    ui.success('Account status updated')
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
        <h2>Users</h2>
        <p>{{ meta?.total || 0 }} account(s)</p>
      </div>
    </div>

    <div class="toolbar panel">
      <input
        v-model="search"
        class="input"
        type="search"
        placeholder="Search by name, email or mobile"
        @keyup.enter="load(1)"
      />
      <select v-model="role" class="select" @change="load(1)">
        <option value="">All roles</option>
        <option value="ADMIN">Administrator</option>
        <option value="CLIENT">Builder</option>
        <option value="CUSTOMER">Buyer</option>
      </select>
      <select v-model="status" class="select" @change="load(1)">
        <option value="">All statuses</option>
        <option value="ACTIVE">Active</option>
        <option value="INACTIVE">Inactive</option>
        <option value="SUSPENDED">Suspended</option>
      </select>
      <button class="btn btn-outline" @click="load(1)">Apply</button>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="6" />

    <EmptyState v-else-if="!items.length" icon="👥" title="No users match this filter" />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Contact</th>
            <th>Roles</th>
            <th>Verified</th>
            <th>Status</th>
            <th>Joined</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in items" :key="user.id">
            <td><strong>{{ user.full_name }}</strong></td>
            <td class="small">
              {{ user.email }}
              <div class="tiny muted">{{ user.mobile }}</div>
            </td>
            <td>
              <span v-for="r in user.roles" :key="r.id" class="badge badge-neutral role-badge">
                {{ r.code }}
              </span>
            </td>
            <td class="tiny">
              <span :class="user.email_verified ? 'ok' : 'no'">Email</span>
              ·
              <span :class="user.mobile_verified ? 'ok' : 'no'">Mobile</span>
            </td>
            <td>
              <span class="badge" :class="`badge-${STATUS_TONE[user.status] || 'neutral'}`">
                {{ humanise(user.status) }}
              </span>
            </td>
            <td class="tiny">{{ formatDate(user.created_at) }}</td>
            <td>
              <button
                v-if="user.status !== 'SUSPENDED'"
                class="btn btn-ghost btn-sm danger"
                @click="setStatus(user, 'SUSPENDED')"
              >
                Suspend
              </button>
              <button v-else class="btn btn-ghost btn-sm" @click="setStatus(user, 'ACTIVE')">
                Reactivate
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>

<style scoped>
.toolbar { display: grid; grid-template-columns: 1fr 170px 170px auto; gap: 10px; padding: 14px; margin-bottom: 16px; }
.role-badge + .role-badge { margin-left: 4px; }
.ok { color: var(--success-600); font-weight: 700; }
.no { color: var(--ink-400); }
.danger { color: var(--danger-600); }

@media (max-width: 860px) {
  .toolbar { grid-template-columns: 1fr; }
}
</style>
