<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { publicApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { useSeo } from '@/composables/useSeo'
import EmptyState from '@/components/feedback/EmptyState.vue'
import {
  formatArea, formatDate, formatPossession, formatPriceRange, humanise, pricePerSqft,
} from '@/utils/format'

const route = useRoute()
const store = usePropertyStore()
const { setMeta } = useSeo()

const items = ref([])
const loading = ref(false)

const ids = computed(() => {
  const raw = route.query.ids ? String(route.query.ids) : store.compareIds.join(',')
  return raw.split(',').map((value) => Number(value)).filter(Boolean)
})

/** Every amenity across the selected projects, so gaps are visible as gaps. */
const allAmenities = computed(() => {
  const seen = new Map()
  for (const item of items.value) {
    for (const amenity of item.amenities || []) seen.set(amenity.id, amenity.name)
  }
  return [...seen.entries()].map(([id, name]) => ({ id, name }))
})

function hasAmenity(item, amenityId) {
  return (item.amenities || []).some((amenity) => amenity.id === amenityId)
}

async function load() {
  if (!ids.value.length) {
    items.value = []
    return
  }
  loading.value = true
  try {
    const res = await publicApi.compare(ids.value)
    items.value = res.data || []
  } finally {
    loading.value = false
  }
}

function remove(item) {
  store.toggleCompare(item)
  items.value = items.value.filter((entry) => entry.id !== item.id)
}

onMounted(() => {
  setMeta({
    title: 'Compare properties',
    description: 'Compare price, carpet area, possession date, RERA status and amenities side by side.',
  })
  load()
})

watch(() => route.query.ids, load)
</script>

<template>
  <div class="container section">
    <header class="section-head">
      <div>
        <h1>Compare properties</h1>
        <p>Price, area, possession and RERA status side by side</p>
      </div>
      <button v-if="items.length" class="btn btn-outline btn-sm" @click="store.clearCompare()">
        Clear selection
      </button>
    </header>

    <div v-if="loading" class="skeleton" style="height: 420px"></div>

    <EmptyState
      v-else-if="!items.length"
      icon="⚖️"
      title="Nothing to compare yet"
      message="Add properties from any listing with the Compare button, then return here."
    >
      <RouterLink class="btn btn-primary" :to="{ name: 'properties' }">Browse properties</RouterLink>
    </EmptyState>

    <div v-else class="table-wrap">
      <table class="table compare-table">
        <thead>
          <tr>
            <th class="row-head">Project</th>
            <th v-for="item in items" :key="item.id">
              <div class="col-head">
                <img
                  v-if="item.cover_image"
                  :src="item.cover_image"
                  :alt="item.name"
                  class="thumb"
                  loading="lazy"
                />
                <RouterLink :to="{ name: 'property-detail', params: { slug: item.slug } }">
                  {{ item.name }}
                </RouterLink>
                <button class="btn btn-ghost btn-sm" @click="remove(item)">Remove</button>
              </div>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <th class="row-head">Price</th>
            <td v-for="item in items" :key="item.id" class="price">
              {{ formatPriceRange(item.min_price, item.max_price) }}
            </td>
          </tr>
          <tr>
            <th class="row-head">Configurations</th>
            <td v-for="item in items" :key="item.id">
              {{ item.configurations?.map((c) => c.name).join(', ') || '-' }}
            </td>
          </tr>
          <tr>
            <th class="row-head">Carpet area</th>
            <td v-for="item in items" :key="item.id">
              {{ formatArea(item.min_area) }} - {{ formatArea(item.max_area) }}
            </td>
          </tr>
          <tr>
            <th class="row-head">Rate per sq.ft.</th>
            <td v-for="item in items" :key="item.id">
              {{ pricePerSqft(item.min_price, item.min_area) }}
            </td>
          </tr>
          <tr>
            <th class="row-head">Location</th>
            <td v-for="item in items" :key="item.id">
              {{ item.locality?.name }}, {{ item.city?.name }}
            </td>
          </tr>
          <tr>
            <th class="row-head">Status</th>
            <td v-for="item in items" :key="item.id">
              {{ humanise(item.construction_status) }}
            </td>
          </tr>
          <tr>
            <th class="row-head">Possession</th>
            <td v-for="item in items" :key="item.id">
              {{ formatPossession(item.possession_date) }}
            </td>
          </tr>
          <tr>
            <th class="row-head">RERA number</th>
            <td v-for="item in items" :key="item.id" class="tiny">
              {{ item.rera_details?.registration_number || 'Not available' }}
            </td>
          </tr>
          <tr>
            <th class="row-head">RERA status</th>
            <td v-for="item in items" :key="item.id">
              <span class="badge" :class="item.is_rera_verified ? 'badge-rera' : 'badge-neutral'">
                {{ item.is_rera_verified ? 'Verified' : humanise(item.rera_details?.verification_status || 'NOT_AVAILABLE') }}
              </span>
            </td>
          </tr>
          <tr>
            <th class="row-head">Completion declared</th>
            <td v-for="item in items" :key="item.id">
              {{ formatDate(item.rera_details?.completion_date) }}
            </td>
          </tr>
          <tr>
            <th class="row-head">Builder</th>
            <td v-for="item in items" :key="item.id">{{ item.builder_name || '-' }}</td>
          </tr>
          <tr v-for="amenity in allAmenities" :key="amenity.id">
            <th class="row-head">{{ amenity.name }}</th>
            <td v-for="item in items" :key="item.id">
              <span :class="hasAmenity(item, amenity.id) ? 'yes' : 'no'">
                {{ hasAmenity(item, amenity.id) ? '✓' : '—' }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.compare-table { min-width: 720px; }
.row-head {
  position: sticky;
  left: 0;
  background: var(--ink-50);
  z-index: 1;
  min-width: 150px;
  text-transform: none;
  font-size: 0.8rem;
  color: var(--ink-700);
}

.col-head { display: flex; flex-direction: column; gap: 6px; align-items: flex-start; min-width: 180px; }
.thumb { width: 100%; height: 96px; object-fit: cover; border-radius: var(--radius); }
.col-head a { font-weight: 700; color: var(--ink-900); font-size: 0.92rem; }

.yes { color: var(--success-600); font-weight: 800; }
.no { color: var(--ink-300); }
</style>
