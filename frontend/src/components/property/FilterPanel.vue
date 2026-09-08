<script setup>
import { computed, reactive, watch } from 'vue'
import { usePropertyStore } from '@/stores/property'

const props = defineProps({
  modelValue: { type: Object, required: true },
  // The rent search swaps the budget block for rental criteria rather than
  // showing both, so the panel never asks a renter about a sale price.
  purpose: { type: String, default: 'BUY' },
})
const emit = defineEmits(['update:modelValue', 'apply', 'reset'])

const store = usePropertyStore()

// A local copy keeps typing out of the URL until the visitor applies the filter.
const draft = reactive({ ...props.modelValue })

watch(
  () => props.modelValue,
  (value) => Object.assign(draft, value),
  { deep: true },
)

const localities = computed(() => store.localitiesForCity(draft.city))

const isRent = computed(() => props.purpose === 'RENT')

const COUNTED_KEYS = [
  'city', 'locality', 'property_type', 'status', 'bhk', 'min_area', 'max_area',
  'amenities', 'q',
  // Purpose-specific keys are counted only when that panel is on screen, so the
  // badge never counts a filter the visitor cannot see.
]

const activeCount = computed(() => {
  const keys = isRent.value
    ? [...COUNTED_KEYS, 'min_rent', 'max_rent', 'furnishing', 'tenant_preference',
       'available_from', 'pet_allowed']
    : [...COUNTED_KEYS, 'min_price', 'max_price', 'rera_verified']
  return keys.filter(
    (key) => draft[key] !== '' && draft[key] !== undefined && draft[key] !== null,
  ).length
})

function applyRent(range) {
  draft.min_rent = range.min || ''
  draft.max_rent = range.max || ''
}

function isRentBand(range) {
  return (
    String(draft.min_rent || '') === String(range.min || '') &&
    String(draft.max_rent || '') === String(range.max || '')
  )
}

const selectedAmenities = computed({
  get: () => (draft.amenities ? String(draft.amenities).split(',').filter(Boolean) : []),
  set: (values) => {
    draft.amenities = values.join(',')
  },
})

function toggleAmenity(id) {
  const current = selectedAmenities.value
  const key = String(id)
  selectedAmenities.value = current.includes(key)
    ? current.filter((item) => item !== key)
    : [...current, key]
}

function onCityChange() {
  draft.locality = ''
}

// A preset band sets both ends at once; an open-ended top stays blank.
function applyBudget(range) {
  draft.min_price = range.min || ''
  draft.max_price = range.max || ''
}

function isBudget(range) {
  return (
    String(draft.min_price || '') === String(range.min || '') &&
    String(draft.max_price || '') === String(range.max || '')
  )
}

function apply() {
  emit('update:modelValue', { ...draft })
  emit('apply', { ...draft })
}

function reset() {
  for (const key of Object.keys(draft)) draft[key] = ''
  emit('reset')
}
</script>

<template>
  <aside class="filters panel">
    <div class="panel-head">
      <h3>Filters <span v-if="activeCount" class="badge badge-info">{{ activeCount }}</span></h3>
      <button class="btn btn-ghost btn-sm" @click="reset">Reset</button>
    </div>

    <div class="field">
      <label class="label" for="f-city">City</label>
      <select id="f-city" v-model="draft.city" class="select" @change="onCityChange">
        <option value="">All cities</option>
        <option v-for="city in store.cities" :key="city.id" :value="city.slug">
          {{ city.name }}
        </option>
      </select>
    </div>

    <div class="field">
      <label class="label" for="f-locality">Locality</label>
      <select id="f-locality" v-model="draft.locality" class="select" :disabled="!draft.city">
        <option value="">{{ draft.city ? 'All localities' : 'Select a city first' }}</option>
        <option v-for="locality in localities" :key="locality.id" :value="locality.slug">
          {{ locality.name }}
        </option>
      </select>
    </div>

    <div class="field">
      <span class="label">Bedrooms</span>
      <div class="pill-group">
        <button
          v-for="option in [1, 2, 3, 4, 5]"
          :key="option"
          class="pill"
          :class="{ active: String(draft.bhk) === String(option) }"
          @click="draft.bhk = String(draft.bhk) === String(option) ? '' : String(option)"
        >
          {{ option }} BHK
        </button>
      </div>
    </div>

    <!-- Rent, or price: never both. -->
    <div v-if="isRent" class="field">
      <span class="label">Monthly rent (₹)</span>
      <div class="pair">
        <input v-model="draft.min_rent" class="input" type="number" min="0" placeholder="Min" />
        <input v-model="draft.max_rent" class="input" type="number" min="0" placeholder="Max" />
      </div>
      <div class="pill-group mt-1">
        <button
          v-for="range in store.rentRanges"
          :key="range.label"
          class="pill"
          :class="{ active: isRentBand(range) }"
          @click="applyRent(range)"
        >
          {{ range.label }}
        </button>
      </div>
    </div>

    <div v-else class="field">
      <span class="label">Budget (₹)</span>
      <div class="pair">
        <input v-model="draft.min_price" class="input" type="number" min="0" placeholder="Min" />
        <input v-model="draft.max_price" class="input" type="number" min="0" placeholder="Max" />
      </div>
      <div class="pill-group mt-1">
        <button
          v-for="range in store.budgetRanges"
          :key="range.label"
          class="pill"
          :class="{ active: isBudget(range) }"
          @click="applyBudget(range)"
        >
          {{ range.label }}
        </button>
      </div>
    </div>

    <template v-if="isRent">
      <div class="field">
        <label class="label" for="f-furnishing">Furnishing</label>
        <select id="f-furnishing" v-model="draft.furnishing" class="select">
          <option value="">Any</option>
          <option v-for="option in store.furnishingOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>

      <div class="field">
        <label class="label" for="f-available">Availability</label>
        <select id="f-available" v-model="draft.available_from" class="select">
          <option value="">Any time</option>
          <option
            v-for="option in store.availabilityOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </option>
        </select>
      </div>

      <div class="field">
        <label class="label" for="f-tenant">Suitable for</label>
        <select id="f-tenant" v-model="draft.tenant_preference" class="select">
          <option value="">Anyone</option>
          <option v-for="option in store.tenantPreferences" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
        <span class="field-hint">Listings open to anyone are always included.</span>
      </div>

      <div class="field">
        <label class="check">
          <input
            type="checkbox"
            :checked="draft.pet_allowed === 'true'"
            @change="draft.pet_allowed = $event.target.checked ? 'true' : ''"
          />
          <span>Pet friendly</span>
        </label>
      </div>
    </template>

    <div class="field">
      <label class="label" for="f-type">Property type</label>
      <select id="f-type" v-model="draft.property_type" class="select">
        <option value="">Any type</option>
        <option v-for="type in store.propertyTypes" :key="type.value" :value="type.value">
          {{ type.label }}
        </option>
      </select>
    </div>

    <div class="field">
      <label class="label" for="f-status">Construction status</label>
      <select id="f-status" v-model="draft.status" class="select">
        <option value="">Any status</option>
        <option
          v-for="status in store.constructionStatuses"
          :key="status.value"
          :value="status.value"
        >
          {{ status.label }}
        </option>
      </select>
    </div>

    <div class="field">
      <span class="label">Carpet area (sq.ft.)</span>
      <div class="pair">
        <input v-model="draft.min_area" class="input" type="number" min="0" placeholder="Min" />
        <input v-model="draft.max_area" class="input" type="number" min="0" placeholder="Max" />
      </div>
    </div>

    <!-- RERA registration is a project-level fact, so it has no meaning on a
         search for individual flats to let. -->
    <div v-if="!isRent" class="field">
      <label class="check">
        <input
          type="checkbox"
          :checked="draft.rera_verified === 'true'"
          @change="draft.rera_verified = $event.target.checked ? 'true' : ''"
        />
        <span>Show only RERA-verified projects</span>
      </label>
    </div>

    <div v-if="store.amenities.length" class="field">
      <span class="label">Amenities</span>
      <div class="amenity-list">
        <label v-for="amenity in store.amenities" :key="amenity.id" class="check">
          <input
            type="checkbox"
            :checked="selectedAmenities.includes(String(amenity.id))"
            @change="toggleAmenity(amenity.id)"
          />
          <span>{{ amenity.name }}</span>
        </label>
      </div>
    </div>

    <button class="btn btn-primary btn-block" @click="apply">Apply filters</button>
  </aside>
</template>

<style scoped>
.filters { position: sticky; top: calc(var(--header-height) + 16px); }

.pair { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }

.pill-group { display: flex; flex-wrap: wrap; gap: 6px; }
.pill {
  padding: 6px 11px;
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-full);
  background: var(--white);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--ink-700);
  cursor: pointer;
  transition: all var(--transition);
}
.pill:hover { border-color: var(--brand-500); color: var(--brand-700); }
.pill.active { background: var(--brand-700); border-color: var(--brand-700); color: var(--white); }

.amenity-list {
  max-height: 208px;
  overflow-y: auto;
  padding-right: 6px;
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  padding: 10px 12px;
}
.amenity-list .check { margin-bottom: 7px; }
.amenity-list .check:last-child { margin-bottom: 0; }

@media (max-width: 980px) {
  .filters { position: static; }
}
</style>
