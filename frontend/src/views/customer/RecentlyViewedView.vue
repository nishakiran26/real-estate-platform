<script setup>
import { onMounted } from 'vue'
import { customerApi } from '@/services/api'
import { useAsync } from '@/composables/useAsync'
import PropertyGrid from '@/components/property/PropertyGrid.vue'

const { data, loading, error, run } = useAsync(() => customerApi.recentlyViewed(24), {
  initial: [],
})

onMounted(run)
</script>

<template>
  <div>
    <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

    <div class="section-head">
      <div>
        <h2>Recently viewed</h2>
        <p>Projects you opened, newest first</p>
      </div>
    </div>

    <PropertyGrid
      :properties="data || []"
      :loading="loading"
      empty-title="Nothing viewed yet"
      empty-message="Projects you open will appear here so you can find them again quickly."
    >
      <template #empty>
        <RouterLink class="btn btn-primary" :to="{ name: 'properties' }">
          Start browsing
        </RouterLink>
      </template>
    </PropertyGrid>
  </div>
</template>
