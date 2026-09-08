<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePropertyStore } from '@/stores/property'

const store = usePropertyStore()
const router = useRouter()

// Buy or rent is the first decision, and it reshapes everything below it: a
// buyer picks a budget in lakhs, a renter a monthly rent in thousands.
const purpose = ref('BUY')
const isRent = computed(() => purpose.value === 'RENT')

const city = ref('')
const locality = ref('')
const bhk = ref('')
const budget = ref('')
const keyword = ref('')

const localities = computed(() => store.localitiesForCity(city.value))
const bands = computed(() => (isRent.value ? store.rentRanges : store.budgetRanges))

function onCityChange() {
  locality.value = ''
}

// The two catalogues price differently, so a band chosen for one is meaningless
// in the other.
function onPurposeChange(next) {
  purpose.value = next
  budget.value = ''
}

function submit() {
  const query = {}
  if (city.value) query.city = city.value
  if (locality.value) query.locality = locality.value
  if (bhk.value) query.bhk = bhk.value
  if (keyword.value.trim()) query.q = keyword.value.trim()

  // Presets carry both ends of the band; an open-ended top is omitted.
  const range = budget.value !== '' ? bands.value[Number(budget.value)] : null
  if (range) {
    if (isRent.value) {
      if (range.min) query.min_rent = range.min
      if (range.max) query.max_rent = range.max
    } else {
      if (range.min) query.min_price = range.min
      if (range.max) query.max_price = range.max
    }
  }

  router.push({ name: isRent.value ? 'rent' : 'properties', query })
}
</script>

<template>
  <form class="hero-search" @submit.prevent="submit">
    <div class="purpose-toggle" role="group" aria-label="What are you looking for?">
      <button
        type="button"
        class="purpose"
        :class="{ active: !isRent }"
        :aria-pressed="!isRent"
        @click="onPurposeChange('BUY')"
      >
        Buy
      </button>
      <button
        type="button"
        class="purpose"
        :class="{ active: isRent }"
        :aria-pressed="isRent"
        @click="onPurposeChange('RENT')"
      >
        Rent
      </button>
    </div>

    <div class="search-row">
      <label class="cell">
        <span class="cell-label">City</span>
        <select v-model="city" class="cell-input" @change="onCityChange">
          <option value="">All cities</option>
          <option v-for="option in store.cities" :key="option.id" :value="option.slug">
            {{ option.name }}
          </option>
        </select>
      </label>

      <label class="cell">
        <span class="cell-label">Locality</span>
        <select v-model="locality" class="cell-input" :disabled="!city">
          <option value="">{{ city ? 'All localities' : 'Select a city first' }}</option>
          <option v-for="option in localities" :key="option.id" :value="option.slug">
            {{ option.name }}
          </option>
        </select>
      </label>

      <label class="cell">
        <span class="cell-label">Bedrooms</span>
        <select v-model="bhk" class="cell-input">
          <option value="">Any</option>
          <option v-for="option in [1, 2, 3, 4, 5]" :key="option" :value="option">
            {{ option }} BHK
          </option>
        </select>
      </label>

      <label class="cell">
        <span class="cell-label">{{ isRent ? 'Monthly rent' : 'Budget' }}</span>
        <select v-model="budget" class="cell-input">
          <option value="">{{ isRent ? 'Any rent' : 'Any budget' }}</option>
          <option v-for="(range, index) in bands" :key="range.label" :value="index">
            {{ range.label }}
          </option>
        </select>
      </label>

      <button class="btn btn-accent search-btn" type="submit">Search</button>
    </div>

    <div class="keyword-row">
      <input
        v-model="keyword"
        class="keyword"
        type="search"
        :placeholder="
          isRent
            ? 'Or search by locality or society name, for example Wakad'
            : 'Or search by project or builder name, for example Green Valley'
        "
        aria-label="Search by project or builder name"
      />
    </div>
  </form>
</template>

<style scoped>
.purpose-toggle {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  margin-bottom: 10px;
  border-radius: var(--radius);
  background: var(--ink-100);
}
.purpose {
  padding: 7px 22px;
  border: 0;
  border-radius: var(--radius-sm);
  background: none;
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--ink-600);
  cursor: pointer;
  transition: all var(--transition);
}
.purpose.active {
  background: var(--brand-700);
  color: var(--white);
  box-shadow: var(--shadow-sm);
}

.hero-search {
  background: var(--white);
  border-radius: var(--radius-lg);
  padding: 14px;
  box-shadow: var(--shadow-lg);
}

.search-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr) auto;
  gap: 8px;
  align-items: stretch;
}

.cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 10px;
  border-radius: var(--radius);
  border: 1px solid var(--ink-200);
  min-width: 0;
}
.cell:focus-within { border-color: var(--brand-500); }

.cell-label {
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--ink-500);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.cell-input {
  border: 0;
  background: none;
  padding: 2px 0;
  font-size: 0.92rem;
  font-weight: 600;
  color: var(--ink-900);
  cursor: pointer;
  width: 100%;
}
.cell-input:focus { outline: none; }
.cell-input:disabled { color: var(--ink-400); cursor: not-allowed; }

.search-btn { padding-inline: 30px; }

.keyword-row { margin-top: 8px; }
.keyword {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  background: var(--ink-50);
  font-size: 0.9rem;
}
.keyword:focus { outline: none; border-color: var(--brand-500); background: var(--white); }

@media (max-width: 900px) {
  .search-row { grid-template-columns: 1fr 1fr; }
  .search-btn { grid-column: 1 / -1; }
}
@media (max-width: 520px) {
  .search-row { grid-template-columns: 1fr; }
}
</style>
