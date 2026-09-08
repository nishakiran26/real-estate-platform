<script setup>
defineProps({
  icon: { type: String, default: '🏙️' },
  title: { type: String, default: 'Nothing here yet' },
  message: { type: String, default: '' },
})
</script>

<template>
  <div class="empty-state">
    <div class="empty-icon" aria-hidden="true">{{ icon }}</div>
    <h3>{{ title }}</h3>
    <p v-if="message" class="muted">{{ message }}</p>
    <div class="empty-actions">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.empty-icon {
  font-size: 2.6rem;
  margin-bottom: 10px;
  opacity: 0.75;
}
.empty-actions:not(:empty) {
  margin-top: 18px;
  display: flex;
  gap: 10px;
  justify-content: center;
  flex-wrap: wrap;
}
</style>
