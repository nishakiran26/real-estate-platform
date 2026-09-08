<script setup>
import { onMounted } from 'vue'
import { customerApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useAsync } from '@/composables/useAsync'
import StatTile from '@/components/common/StatTile.vue'
import PropertyGrid from '@/components/property/PropertyGrid.vue'

const auth = useAuthStore()
const { data, loading, error, run } = useAsync(() => customerApi.dashboard())

onMounted(run)
</script>

<template>
  <div class="stack">
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <section>
      <h2 class="mb-1">Welcome back, {{ auth.displayName.split(' ')[0] }}</h2>
      <p class="muted small">Your shortlist, inquiries and recommendations in one place.</p>

      <div class="grid grid-4 mt-2">
        <StatTile label="Saved properties" :value="data?.saved_properties || 0" />
        <StatTile label="Inquiries sent" :value="data?.inquiries || 0" />
        <StatTile label="Recently viewed" :value="data?.recently_viewed_count || 0" />
        <StatTile
          label="Unread notifications"
          :value="data?.unread_notifications || 0"
          :tone="data?.unread_notifications ? 'warning' : 'default'"
        />
      </div>
    </section>

    <section v-if="loading || data?.recommended?.length">
      <div class="section-head">
        <div>
          <h3>Recommended for you</h3>
          <p>Based on your preferences and what you have viewed</p>
        </div>
        <RouterLink class="btn btn-outline btn-sm" :to="{ name: 'customer-preferences' }">
          Tune preferences
        </RouterLink>
      </div>
      <PropertyGrid
        :properties="data?.recommended || []"
        :loading="loading"
        :skeleton-count="3"
        compact
        empty-title="No recommendations yet"
        empty-message="Set your preferred city and budget and we will suggest matching projects."
      />
    </section>

    <section v-if="data?.recently_viewed?.length">
      <div class="section-head">
        <div>
          <h3>Recently viewed</h3>
          <p>Pick up where you left off</p>
        </div>
        <RouterLink class="btn btn-outline btn-sm" :to="{ name: 'customer-recent' }">
          View all
        </RouterLink>
      </div>
      <PropertyGrid :properties="data.recently_viewed" compact />
    </section>
  </div>
</template>
