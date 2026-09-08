<script setup>
import { usePropertyStore } from '@/stores/property'

const store = usePropertyStore()
</script>

<template>
  <!-- A persistent tray keeps the comparison selection visible while the visitor
       keeps browsing, which is the whole point of collecting it. -->
  <Transition name="tray">
    <div v-if="store.compareCount" class="compare-tray">
      <div class="container inner">
        <div class="items">
          <span class="label">Comparing</span>
          <span v-for="item in store.compare" :key="item.id" class="chip">
            <span class="truncate">{{ item.name }}</span>
            <button aria-label="Remove" @click="store.toggleCompare(item)">×</button>
          </span>
        </div>
        <div class="row">
          <button class="btn btn-ghost btn-sm" @click="store.clearCompare()">Clear</button>
          <RouterLink
            class="btn btn-primary btn-sm"
            :to="{ name: 'compare', query: { ids: store.compareIds.join(',') } }"
          >
            Compare {{ store.compareCount }}
          </RouterLink>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.compare-tray {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 90;
  background: var(--white);
  border-top: 1px solid var(--ink-200);
  box-shadow: 0 -6px 24px rgba(15, 23, 32, 0.1);
  padding: 12px 0;
}

.inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.items { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; flex: 1; min-width: 0; }
.label { font-size: 0.8rem; font-weight: 700; color: var(--ink-600); text-transform: uppercase; letter-spacing: 0.04em; }

.chip {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  max-width: 190px;
  padding: 5px 8px 5px 12px;
  border-radius: var(--radius-full);
  background: var(--brand-50);
  border: 1px solid var(--brand-100);
  font-size: 0.82rem;
  color: var(--brand-800);
}
.chip button {
  border: 0;
  background: none;
  color: var(--brand-700);
  font-size: 1.05rem;
  line-height: 1;
  cursor: pointer;
  padding: 0 2px;
}
.chip button:hover { color: var(--danger-600); }

.tray-enter-active, .tray-leave-active { transition: transform 200ms ease, opacity 200ms ease; }
.tray-enter-from, .tray-leave-to { transform: translateY(100%); opacity: 0; }
</style>
