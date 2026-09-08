<script setup>
import { formatNumber } from '@/utils/format'

defineProps({
  label: { type: String, required: true },
  value: { type: [Number, String], default: 0 },
  meta: { type: String, default: '' },
  suffix: { type: String, default: '' },
  tone: { type: String, default: 'default' }, // default | success | warning | danger
})
</script>

<template>
  <div class="stat" :class="`tone-${tone}`">
    <span class="stat-label">{{ label }}</span>
    <div class="stat-value">
      {{ typeof value === 'number' ? formatNumber(value) : value }}<span
        v-if="suffix"
        class="suffix"
        >{{ suffix }}</span
      >
    </div>
    <span v-if="meta" class="stat-meta">{{ meta }}</span>
  </div>
</template>

<style scoped>
.suffix { font-size: 0.9rem; font-weight: 600; color: var(--ink-500); margin-left: 2px; }

.tone-success { border-left: 3px solid var(--success-600); }
.tone-warning { border-left: 3px solid var(--warning-600); }
.tone-danger { border-left: 3px solid var(--danger-600); }
</style>
