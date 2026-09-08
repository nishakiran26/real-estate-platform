<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { adminApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { STATUS_VARIANT } from '@/utils/constants'
import { formatDate, formatNumber, formatPriceRange, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const props = defineProps({
  // The approval queue is this same table pinned to the pending statuses.
  queueMode: { type: Boolean, default: false },
})

const route = useRoute()
const ui = useUiStore()

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')
const status = ref(props.queueMode ? 'SUBMITTED' : '')
const search = ref('')
// Two catalogues share one queue, so an administrator reviewing rentals should
// not have to read past sale listings.
const purpose = ref('')

async function load(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await adminApi.properties({
      page,
      per_page: 15,
      status: status.value,
      purpose: purpose.value,
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

async function toggleFeatured(item) {
  try {
    await adminApi.setFeatured(item.id, !item.is_featured)
    item.is_featured = !item.is_featured
    ui.success(item.is_featured ? 'Marked as featured' : 'Removed from featured')
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
        <h2>{{ queueMode ? 'Approval queue' : 'All properties' }}</h2>
        <p>
          {{ meta?.total || 0 }} listing(s)
          <template v-if="queueMode"> waiting for a decision</template>
        </p>
      </div>
    </div>

    <div class="toolbar panel">
      <input
        v-model="search"
        class="input"
        type="search"
        placeholder="Search by project or builder"
        @keyup.enter="load(1)"
      />
      <select v-model="status" class="select" @change="load(1)">
        <option value="">All statuses</option>
        <option value="SUBMITTED">Submitted</option>
        <option value="UNDER_REVIEW">Under review</option>
        <option value="PUBLISHED">Published</option>
        <option value="APPROVED">Approved</option>
        <option value="REJECTED">Rejected</option>
        <option value="DRAFT">Draft</option>
      </select>
      <select v-model="purpose" class="select" @change="load(1)">
        <option value="">Buy and rent</option>
        <option value="BUY">To buy</option>
        <option value="RENT">To rent</option>
      </select>
      <button class="btn btn-outline" @click="load(1)">Apply</button>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="6" />

    <EmptyState
      v-else-if="!items.length"
      icon="📋"
      :title="queueMode ? 'Nothing waiting for review' : 'No properties found'"
      :message="
        queueMode
          ? 'Every submitted listing has been dealt with.'
          : 'Try a different status filter or search term.'
      "
    />

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Project</th>
            <th>Builder</th>
            <th>Location</th>
            <th class="num">Price</th>
            <th>RERA</th>
            <th>Status</th>
            <th class="num">Views</th>
            <th>Submitted</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id">
            <td>
              <strong>{{ item.name }}</strong>
              <div class="tiny muted">{{ humanise(item.property_type) }}</div>
            </td>
            <td class="small">{{ item.builder_name || '-' }}</td>
            <td class="small">{{ item.locality }}, {{ item.city }}</td>
            <td class="num">{{ formatPriceRange(item.min_price, item.max_price) }}</td>
            <td>
              <span class="badge" :class="item.is_rera_verified ? 'badge-rera' : 'badge-neutral'">
                {{ item.is_rera_verified ? 'Verified' : 'Unverified' }}
              </span>
            </td>
            <td>
              <span class="badge" :class="`badge-${STATUS_VARIANT[item.property_status] || 'neutral'}`">
                {{ humanise(item.property_status) }}
              </span>
            </td>
            <td class="num">{{ formatNumber(item.view_count) }}</td>
            <td class="tiny">{{ formatDate(item.created_at) }}</td>
            <td>
              <div class="actions">
                <RouterLink
                  class="btn btn-primary btn-sm"
                  :to="{ name: 'admin-property-detail', params: { id: item.id } }"
                >
                  Review
                </RouterLink>
                <button
                  v-if="['PUBLISHED', 'APPROVED'].includes(item.property_status)"
                  class="btn btn-ghost btn-sm"
                  @click="toggleFeatured(item)"
                >
                  {{ item.is_featured ? 'Unfeature' : 'Feature' }}
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
.toolbar { display: grid; grid-template-columns: 1fr 200px 160px auto; gap: 10px; padding: 14px; margin-bottom: 16px; }
.actions { display: flex; gap: 3px; flex-wrap: wrap; }

@media (max-width: 700px) {
  .toolbar { grid-template-columns: 1fr; }
}
</style>
