<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { customerApi, publicApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSeo } from '@/composables/useSeo'
import { formatPrice } from '@/utils/format'
import FilterPanel from '@/components/property/FilterPanel.vue'
import PropertyGrid from '@/components/property/PropertyGrid.vue'
import PaginationBar from '@/components/common/PaginationBar.vue'

const route = useRoute()
const router = useRouter()
const store = usePropertyStore()
const auth = useAuthStore()
const ui = useUiStore()
const { setMeta } = useSeo()

const FILTER_KEYS = [
  'city', 'locality', 'property_type', 'status', 'bhk',
  'min_price', 'max_price', 'min_area', 'max_area',
  'rera_verified', 'amenities', 'builder', 'q',
  // Rental criteria. They travel in the URL like every other filter, so a rent
  // search stays shareable.
  'min_rent', 'max_rent', 'furnishing', 'tenant_preference', 'available_from',
  'pet_allowed',
]

const filters = reactive(Object.fromEntries(FILTER_KEYS.map((key) => [key, ''])))
const sort = ref('relevance')
const page = ref(1)
const mobileFiltersOpen = ref(false)

/**
 * The URL is the source of truth for a search, so a result page can be shared,
 * bookmarked and reopened with the browser back button intact.
 */
function readQuery() {
  const query = route.query
  for (const key of FILTER_KEYS) {
    filters[key] = query[key] !== undefined ? String(query[key]) : ''
  }
  // The /city/:citySlug and /city/:citySlug/:localitySlug routes carry the
  // location in the path instead of the query string.
  if (route.params.citySlug) filters.city = String(route.params.citySlug)
  if (route.params.localitySlug) filters.locality = String(route.params.localitySlug)

  sort.value = query.sort ? String(query.sort) : 'relevance'
  page.value = Number(query.page) || 1
}

function writeQuery(extra = {}) {
  const query = {}
  for (const key of FILTER_KEYS) {
    // Location that lives in the path is not duplicated in the query string.
    if (key === 'city' && route.params.citySlug) continue
    if (key === 'locality' && route.params.localitySlug) continue
    if (filters[key] !== '' && filters[key] !== undefined) query[key] = filters[key]
  }
  if (sort.value && sort.value !== 'relevance') query.sort = sort.value
  const target = { ...query, ...extra }
  if (Number(target.page) === 1) delete target.page
  router.push({ name: route.name, params: route.params, query: target })
}

// Purpose comes from the route, not a control: /rent is the rent page and
// /properties is the buy page, so the URL alone says which catalogue is shown.
const purpose = computed(() => (route.meta.purpose === 'RENT' ? 'RENT' : 'BUY'))
const isRent = computed(() => purpose.value === 'RENT')

async function load() {
  const params = {
    ...filters,
    purpose: purpose.value,
    sort: sort.value,
    page: page.value,
    per_page: 12,
  }
  await store.search(params)
  updateSeo()
  loadRentGuide()
}

// What a flat here actually goes for. It answers the question a renter has
// before they can judge any single listing, and it is shown only when there are
// enough listings behind it to mean something.
const rentGuide = ref(null)

async function loadRentGuide() {
  if (!isRent.value) {
    rentGuide.value = null
    return
  }
  const city = store.cities.find((item) => item.slug === filters.city)
  const locality = store.localities.find((item) => item.slug === filters.locality)
  try {
    const res = await publicApi.rentGuide({
      city_id: city?.id,
      locality_id: locality?.id,
      bhk: filters.bhk || undefined,
    })
    // A single listing is not a market rate, so below that the guide stays off
    // rather than presenting one flat as what the area costs.
    rentGuide.value = (res.data?.listings || 0) >= 3 ? res.data : null
  } catch {
    rentGuide.value = null
  }
}

// Saving a search is only useful once it is a search. A locality, a BHK or a
// rent ceiling is enough to be worth watching; "everything" is not, and would
// notify on every flat posted.
const canSaveSearch = computed(
  () => Boolean(filters.locality || filters.bhk || filters.max_rent || filters.min_rent),
)
const savingSearch = ref(false)

async function saveSearch() {
  if (!auth.isAuthenticated) {
    // The search is kept in the URL, so signing in returns to exactly it.
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  savingSearch.value = true
  try {
    const city = store.cities.find((item) => item.slug === filters.city)
    const locality = store.localities.find((item) => item.slug === filters.locality)
    const res = await customerApi.saveRentalAlert({
      city_id: city?.id || 0,
      locality_id: locality?.id || 0,
      bhk: Number(filters.bhk) || 0,
      min_rent: Number(filters.min_rent) || 0,
      max_rent: Number(filters.max_rent) || 0,
      furnishing: filters.furnishing || '',
      tenant_preference: filters.tenant_preference || '',
    })
    ui.success(res.message || 'Search saved')
  } catch (err) {
    ui.error(err.message)
  } finally {
    savingSearch.value = false
  }
}

const heading = computed(() => {
  const parts = []
  if (filters.bhk) parts.push(`${filters.bhk} BHK`)
  parts.push(
    filters.property_type
      ? labelFor(store.propertyTypes, filters.property_type)
      : isRent.value
        ? 'Rentals'
        : 'Properties',
  )
  if (isRent.value) parts.push('for Rent')
  if (filters.locality) parts.push(`in ${titleCase(filters.locality)}`)
  else if (filters.city) parts.push(`in ${titleCase(filters.city)}`)
  return parts.join(' ')
})

function updateSeo() {
  const total = store.meta?.total || 0
  setMeta({
    title: heading.value,
    description: isRent.value
      ? `${total} ${heading.value.toLowerCase()} with monthly rent, deposit, furnishing and availability. Contact owners directly.`
      : `${total} RERA-registered ${heading.value.toLowerCase()} with verified registration details, carpet area, price and possession dates.`,
  })
}

// Renters look for homes, buyers for projects.
const resultNoun = computed(() => {
  const total = store.meta?.total || 0
  if (isRent.value) return total === 1 ? 'home' : 'homes'
  return total === 1 ? 'project' : 'projects'
})

function labelFor(options, value) {
  return options.find((option) => option.value === value)?.label || 'Properties'
}

function titleCase(slug) {
  return String(slug)
    .split('-')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function onApply() {
  page.value = 1
  mobileFiltersOpen.value = false
  writeQuery({ page: 1 })
}

function onReset() {
  for (const key of FILTER_KEYS) filters[key] = ''
  page.value = 1
  // Clearing filters must not silently move a renter to the buy catalogue.
  router.push({ name: isRent.value ? 'rent' : 'properties' })
}

function changePage(next) {
  page.value = next
  writeQuery({ page: next })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function changeSort(value) {
  sort.value = value
  page.value = 1
  writeQuery({ page: 1 })
}

onMounted(async () => {
  await store.loadReference()
  readQuery()
  load()
})

// Reacting to the route rather than to the controls keeps back/forward working.
watch(
  () => route.fullPath,
  () => {
    readQuery()
    load()
  },
)
</script>

<template>
  <div class="container listing">
    <nav class="crumbs tiny" aria-label="Breadcrumb">
      <RouterLink :to="{ name: 'home' }">Home</RouterLink>
      <span aria-hidden="true">/</span>
      <RouterLink :to="{ name: isRent ? 'rent' : 'properties' }">
        {{ isRent ? 'Rent' : 'Properties' }}
      </RouterLink>
      <template v-if="filters.city">
        <span aria-hidden="true">/</span>
        <span>{{ titleCase(filters.city) }}</span>
      </template>
      <template v-if="filters.locality">
        <span aria-hidden="true">/</span>
        <span>{{ titleCase(filters.locality) }}</span>
      </template>
    </nav>

    <header class="listing-head">
      <div>
        <h1>{{ heading }}</h1>
        <p class="muted small mb-0">
          <template v-if="store.loading">Searching…</template>
          <template v-else>
            {{ store.meta?.total || 0 }} {{ resultNoun }} found
          </template>
        </p>
      </div>

      <div class="row">
        <!-- Offered on rent only, and only once the search has been narrowed:
             saving "every flat in Pune" would notify on the whole catalogue. -->
        <button
          v-if="isRent && canSaveSearch"
          class="btn btn-outline btn-sm"
          :disabled="savingSearch"
          @click="saveSearch"
        >
          {{ savingSearch ? 'Saving…' : '🔔 Save this search' }}
        </button>
        <button class="btn btn-outline btn-sm only-mobile" @click="mobileFiltersOpen = true">
          Filters
        </button>
        <label class="sort">
          <span class="visually-hidden">Sort results</span>
          <select class="select" :value="sort" @change="changeSort($event.target.value)">
            <option v-for="option in store.sortOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>
      </div>
    </header>

    <div class="layout">
      <div class="filters-column hide-mobile">
        <FilterPanel v-model="filters" :purpose="purpose" @apply="onApply" @reset="onReset" />
      </div>

      <div class="results">
        <div v-if="store.error" class="alert alert-error mb-2">{{ store.error }}</div>

        <!-- The question a renter has before they can judge any single listing. -->
        <div v-if="rentGuide" class="rent-guide panel">
          <div>
            <span class="tiny muted">Typical rent here</span>
            <strong class="guide-value">{{ formatPrice(rentGuide.average_rent) }}</strong>
            <span class="tiny muted">per month</span>
          </div>
          <div class="guide-range tiny muted">
            {{ formatPrice(rentGuide.min_rent) }} – {{ formatPrice(rentGuide.max_rent) }}
            across {{ rentGuide.listings }} available listings
          </div>
        </div>

        <PropertyGrid
          :properties="store.results"
          :loading="store.loading"
          :skeleton-count="6"
          :empty-title="isRent ? 'No rentals match these filters' : 'No projects match these filters'"
          :empty-message="
            isRent
              ? 'Try widening the rent range, changing the furnishing, or searching a nearby locality.'
              : 'Try widening the budget, removing an amenity, or searching a nearby locality.'
          "
        >
          <template #empty>
            <button class="btn btn-outline" @click="onReset">Clear all filters</button>
          </template>
        </PropertyGrid>

        <PaginationBar :meta="store.meta" @change="changePage" />
      </div>
    </div>

    <Transition name="sheet">
      <div v-if="mobileFiltersOpen" class="sheet-backdrop" @click.self="mobileFiltersOpen = false">
        <div class="sheet">
          <div class="sheet-head">
            <strong>Filters</strong>
            <button class="btn btn-ghost btn-sm" @click="mobileFiltersOpen = false">Close</button>
          </div>
          <FilterPanel v-model="filters" :purpose="purpose" @apply="onApply" @reset="onReset" />
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.listing { padding: 22px 0 40px; }

.crumbs { display: flex; gap: 7px; color: var(--ink-500); margin-bottom: 14px; flex-wrap: wrap; }
.crumbs a { color: var(--ink-600); }

.listing-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.listing-head h1 { font-size: 1.6rem; margin-bottom: 4px; }

.sort .select { min-width: 190px; height: 38px; padding-block: 0; }

.layout { display: grid; grid-template-columns: 288px 1fr; gap: 24px; align-items: start; }

.sheet-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 32, 0.5);
  z-index: 200;
  display: flex;
  align-items: flex-end;
}
.sheet {
  background: var(--ink-50);
  width: 100%;
  max-height: 88vh;
  overflow-y: auto;
  border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  padding: 16px;
}
.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.sheet-enter-active, .sheet-leave-active { transition: opacity 180ms ease; }
.sheet-enter-from, .sheet-leave-to { opacity: 0; }

@media (max-width: 980px) {
  .layout { grid-template-columns: 1fr; }
  .filters-column { display: none; }
}

.rent-guide {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  padding: 12px 16px;
  margin-bottom: 14px;
}
.guide-value { font-size: 1.15rem; margin: 0 6px; }
</style>
