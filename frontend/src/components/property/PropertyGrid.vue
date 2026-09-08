<script setup>
import PropertyCard from './PropertyCard.vue'
import SkeletonCard from '@/components/feedback/SkeletonCard.vue'
import EmptyState from '@/components/feedback/EmptyState.vue'

defineProps({
  properties: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  skeletonCount: { type: Number, default: 6 },
  emptyTitle: { type: String, default: 'No properties found' },
  emptyMessage: { type: String, default: 'Try widening your budget or removing a filter.' },
  compact: { type: Boolean, default: false },
})
</script>

<template>
  <!-- One component owns the loading, empty and loaded states for every list of
       properties on the site, so they never drift apart. -->
  <SkeletonCard v-if="loading" :count="skeletonCount" />

  <div v-else-if="properties.length" class="grid grid-cards">
    <PropertyCard
      v-for="property in properties"
      :key="property.id"
      :property="property"
      :compact="compact"
    />
  </div>

  <EmptyState v-else :title="emptyTitle" :message="emptyMessage">
    <slot name="empty" />
  </EmptyState>
</template>
