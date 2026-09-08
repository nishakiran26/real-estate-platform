<script setup>
import EmptyState from '@/components/feedback/EmptyState.vue'
</script>

<template>
  <div class="container section">
    <EmptyState
      icon="🧭"
      title="We could not find that page"
      message="The link may be out of date, or the project may no longer be listed."
    >
      <RouterLink class="btn btn-primary" :to="{ name: 'home' }">Go to the homepage</RouterLink>
      <RouterLink class="btn btn-outline" :to="{ name: 'properties' }">
        Browse properties
      </RouterLink>
    </EmptyState>
  </div>
</template>
