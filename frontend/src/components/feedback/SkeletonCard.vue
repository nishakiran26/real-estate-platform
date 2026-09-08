<script setup>
defineProps({
  count: { type: Number, default: 6 },
  variant: { type: String, default: 'card' }, // card | row
})
</script>

<template>
  <!-- Skeletons mirror the shape of the content they stand in for, so the page
       does not jump when the real data arrives. -->
  <div v-if="variant === 'card'" class="grid grid-cards">
    <div v-for="n in count" :key="n" class="card sk-card">
      <div class="skeleton sk-image"></div>
      <div class="sk-body">
        <div class="skeleton sk-line w-70"></div>
        <div class="skeleton sk-line w-45"></div>
        <div class="skeleton sk-line w-90"></div>
        <div class="skeleton sk-line w-35"></div>
      </div>
    </div>
  </div>

  <div v-else class="sk-rows">
    <div v-for="n in count" :key="n" class="skeleton sk-row"></div>
  </div>
</template>

<style scoped>
.sk-card { overflow: hidden; }
.sk-image { height: 190px; border-radius: 0; }
.sk-body { padding: 16px; display: grid; gap: 10px; }
.sk-line { height: 12px; }
.w-90 { width: 90%; }
.w-70 { width: 70%; }
.w-45 { width: 45%; }
.w-35 { width: 35%; }

.sk-rows { display: grid; gap: 10px; }
.sk-row { height: 54px; }
</style>
