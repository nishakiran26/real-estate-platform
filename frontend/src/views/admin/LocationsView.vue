<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { adminApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { useUiStore } from '@/stores/ui'

const store = usePropertyStore()
const ui = useUiStore()

const selectedCity = ref('')
const savingCity = ref(false)
const savingLocality = ref(false)

const cityForm = reactive({
  id: null,
  name: '',
  state: 'Maharashtra',
  state_code: 'MH',
  latitude: 0,
  longitude: 0,
  is_popular: false,
  display_order: 0,
})

const localityForm = reactive({
  id: null,
  city_id: '',
  name: '',
  pincode: '',
  latitude: 0,
  longitude: 0,
  is_popular: false,
  display_order: 0,
})

const localities = computed(() =>
  selectedCity.value
    ? store.localities.filter((item) => String(item.city_id) === String(selectedCity.value))
    : store.localities,
)

async function refresh() {
  await store.loadReference(true)
}

function editCity(city) {
  Object.assign(cityForm, {
    id: city.id,
    name: city.name,
    state: city.state,
    state_code: city.state_code,
    latitude: city.latitude,
    longitude: city.longitude,
    is_popular: city.is_popular,
    display_order: city.display_order,
  })
}

function resetCity() {
  Object.assign(cityForm, {
    id: null, name: '', state: 'Maharashtra', state_code: 'MH',
    latitude: 0, longitude: 0, is_popular: false, display_order: 0,
  })
}

async function saveCity() {
  savingCity.value = true
  try {
    const payload = { ...cityForm, latitude: Number(cityForm.latitude) || 0, longitude: Number(cityForm.longitude) || 0 }
    if (cityForm.id) await adminApi.updateCity(cityForm.id, payload)
    else await adminApi.createCity(payload)
    ui.success('City saved')
    resetCity()
    await refresh()
  } catch (err) {
    ui.error(err.message)
  } finally {
    savingCity.value = false
  }
}

async function deleteCity(city) {
  const confirmed = await ui.confirm({
    title: `Delete ${city.name}?`,
    message: 'A city that still has localities or listings cannot be deleted.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!confirmed) return
  try {
    await adminApi.deleteCity(city.id)
    ui.success('City deleted')
    await refresh()
  } catch (err) {
    ui.error(err.message)
  }
}

function editLocality(locality) {
  Object.assign(localityForm, {
    id: locality.id,
    city_id: locality.city_id,
    name: locality.name,
    pincode: locality.pincode,
    latitude: locality.latitude,
    longitude: locality.longitude,
    is_popular: locality.is_popular,
    display_order: locality.display_order,
  })
}

function resetLocality() {
  Object.assign(localityForm, {
    id: null, city_id: selectedCity.value || '', name: '', pincode: '',
    latitude: 0, longitude: 0, is_popular: false, display_order: 0,
  })
}

async function saveLocality() {
  savingLocality.value = true
  try {
    const payload = {
      ...localityForm,
      city_id: Number(localityForm.city_id),
      latitude: Number(localityForm.latitude) || 0,
      longitude: Number(localityForm.longitude) || 0,
    }
    if (localityForm.id) await adminApi.updateLocality(localityForm.id, payload)
    else await adminApi.createLocality(payload)
    ui.success('Locality saved')
    resetLocality()
    await refresh()
  } catch (err) {
    ui.error(err.message)
  } finally {
    savingLocality.value = false
  }
}

async function deleteLocality(locality) {
  const confirmed = await ui.confirm({
    title: `Delete ${locality.name}?`,
    message: 'A locality that still has listings cannot be deleted.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!confirmed) return
  try {
    await adminApi.deleteLocality(locality.id)
    ui.success('Locality deleted')
    await refresh()
  } catch (err) {
    ui.error(err.message)
  }
}

onMounted(async () => {
  await store.loadReference()
})
</script>

<template>
  <div class="stack">
    <div class="two-col">
      <section class="panel">
        <div class="panel-head">
          <h3>Cities</h3>
          <button class="btn btn-ghost btn-sm" @click="resetCity">New city</button>
        </div>

        <form class="editor" @submit.prevent="saveCity">
          <div class="form-grid">
            <label class="field">
              <span class="label">Name</span>
              <input v-model="cityForm.name" class="input" required />
            </label>
            <label class="field">
              <span class="label">State</span>
              <input v-model="cityForm.state" class="input" required />
            </label>
            <label class="field">
              <span class="label">State code</span>
              <input v-model="cityForm.state_code" class="input" maxlength="10" />
            </label>
            <label class="field">
              <span class="label">Display order</span>
              <input v-model="cityForm.display_order" class="input" type="number" />
            </label>
          </div>
          <label class="check">
            <input v-model="cityForm.is_popular" type="checkbox" />
            <span>Show in popular cities on the homepage</span>
          </label>
          <button class="btn btn-primary btn-sm" type="submit" :disabled="savingCity">
            {{ cityForm.id ? 'Update city' : 'Add city' }}
          </button>
        </form>

        <div class="table-wrap mt-2">
          <table class="table">
            <thead>
              <tr>
                <th>City</th>
                <th>State</th>
                <th>Popular</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="city in store.cities" :key="city.id">
                <td><strong>{{ city.name }}</strong></td>
                <td class="small">{{ city.state }}</td>
                <td>
                  <span class="badge" :class="city.is_popular ? 'badge-success' : 'badge-neutral'">
                    {{ city.is_popular ? 'Yes' : 'No' }}
                  </span>
                </td>
                <td>
                  <button class="btn btn-ghost btn-sm" @click="editCity(city)">Edit</button>
                  <button class="btn btn-ghost btn-sm danger" @click="deleteCity(city)">
                    Delete
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="panel">
        <div class="panel-head">
          <h3>Localities</h3>
          <button class="btn btn-ghost btn-sm" @click="resetLocality">New locality</button>
        </div>

        <form class="editor" @submit.prevent="saveLocality">
          <div class="form-grid">
            <label class="field">
              <span class="label">City</span>
              <select v-model="localityForm.city_id" class="select" required>
                <option value="">Select a city</option>
                <option v-for="city in store.cities" :key="city.id" :value="city.id">
                  {{ city.name }}
                </option>
              </select>
            </label>
            <label class="field">
              <span class="label">Name</span>
              <input v-model="localityForm.name" class="input" required />
            </label>
            <label class="field">
              <span class="label">Pincode</span>
              <input v-model="localityForm.pincode" class="input" maxlength="6" />
            </label>
            <label class="field">
              <span class="label">Display order</span>
              <input v-model="localityForm.display_order" class="input" type="number" />
            </label>
          </div>
          <label class="check">
            <input v-model="localityForm.is_popular" type="checkbox" />
            <span>Show as a popular locality</span>
          </label>
          <button class="btn btn-primary btn-sm" type="submit" :disabled="savingLocality">
            {{ localityForm.id ? 'Update locality' : 'Add locality' }}
          </button>
        </form>

        <label class="field mt-2">
          <span class="label">Filter by city</span>
          <select v-model="selectedCity" class="select">
            <option value="">All cities</option>
            <option v-for="city in store.cities" :key="city.id" :value="city.id">
              {{ city.name }}
            </option>
          </select>
        </label>

        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>Locality</th>
                <th>Pincode</th>
                <th>Popular</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="locality in localities" :key="locality.id">
                <td>
                  <strong>{{ locality.name }}</strong>
                  <div class="tiny muted">{{ locality.city_name }}</div>
                </td>
                <td class="small">{{ locality.pincode || '-' }}</td>
                <td>
                  <span class="badge" :class="locality.is_popular ? 'badge-success' : 'badge-neutral'">
                    {{ locality.is_popular ? 'Yes' : 'No' }}
                  </span>
                </td>
                <td>
                  <button class="btn btn-ghost btn-sm" @click="editLocality(locality)">Edit</button>
                  <button class="btn btn-ghost btn-sm danger" @click="deleteLocality(locality)">
                    Delete
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.two-col { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 20px; align-items: start; }
.editor { padding: 14px; border: 1px dashed var(--ink-300); border-radius: var(--radius); background: var(--ink-50); }
.danger { color: var(--danger-600); }
</style>
