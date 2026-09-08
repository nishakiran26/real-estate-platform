<script setup>
import { onMounted, ref } from 'vue'
import { customerApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import PropertyGrid from '@/components/property/PropertyGrid.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const store = usePropertyStore()
const items = ref([])
const meta = ref(null)
const loading = ref(true)
const error = ref('')

async function load(page = 1) {
  loading.value = true
  try {
    const res = await customerApi.favorites({ page, per_page: 12 })
    items.value = res.data || []
    meta.value = res.meta || null
    store.syncFavoritesFromCards(items.value)
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
        <h2>Saved properties</h2>
        <p>{{ meta?.total || 0 }} project(s) on your shortlist</p>
      </div>
    </div>

    <PropertyGrid
      :properties="items"
      :loading="loading"
      empty-title="Your shortlist is empty"
      empty-message="Tap the heart on any listing to keep it here for later."
    >
      <template #empty>
        <RouterLink class="btn btn-primary" :to="{ name: 'properties' }">
          Browse properties
        </RouterLink>
      </template>
    </PropertyGrid>

    <PaginationBar :meta="meta" @change="load" />
  </div>
</template>
