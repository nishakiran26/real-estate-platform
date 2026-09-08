<script setup>
import { onMounted, ref } from 'vue'
import { customerApi } from '@/services/api'
import { INQUIRY_STATUS_VARIANT } from '@/utils/constants'
import { formatDateTime, humanise } from '@/utils/format'
import EmptyState from '@/components/feedback/EmptyState.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')

async function load(page = 1) {
  loading.value = true
  try {
    const res = await customerApi.inquiries({ page, per_page: 20 })
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
        <h2>My inquiries</h2>
        <p>Requests you have sent to builders</p>
      </div>
    </div>

    <SkeletonCard v-if="loading" variant="row" :count="4" />

    <EmptyState
      v-else-if="!items.length"
      icon="✉️"
      title="No inquiries yet"
      message="When you contact a builder from a listing, the request appears here."
    >
      <RouterLink class="btn btn-primary" :to="{ name: 'properties' }">Find a project</RouterLink>
    </EmptyState>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Project</th>
            <th>Configuration</th>
            <th>Sent</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id">
            <td>
              <RouterLink
                v-if="item.property"
                :to="{ name: 'property-detail', params: { slug: item.property.slug } }"
              >
                <strong>{{ item.property.name }}</strong>
              </RouterLink>
              <span v-else class="muted">Listing removed</span>
              <div v-if="item.message" class="tiny muted clamp-2">{{ item.message }}</div>
            </td>
            <td>{{ item.configuration || 'Any' }}</td>
            <td class="tiny">{{ formatDateTime(item.created_at) }}</td>
            <td>
              <span class="badge" :class="`badge-${INQUIRY_STATUS_VARIANT[item.status] || 'neutral'}`">
                {{ humanise(item.status) }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>
