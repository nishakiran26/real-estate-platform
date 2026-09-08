<script setup>
import { onMounted, reactive, ref } from 'vue'
import { adminApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { useUiStore } from '@/stores/ui'

const store = usePropertyStore()
const ui = useUiStore()

const CATEGORIES = ['GENERAL', 'SPORTS', 'LEISURE', 'SAFETY', 'CONVENIENCE', 'ENVIRONMENT']

const form = reactive({ id: null, name: '', icon: '', category: 'GENERAL', display_order: 0 })
const saving = ref(false)

function edit(amenity) {
  Object.assign(form, {
    id: amenity.id,
    name: amenity.name,
    icon: amenity.icon,
    category: amenity.category,
    display_order: amenity.display_order,
  })
}

function reset() {
  Object.assign(form, { id: null, name: '', icon: '', category: 'GENERAL', display_order: 0 })
}

async function save() {
  saving.value = true
  try {
    const payload = { ...form, display_order: Number(form.display_order) || 0 }
    if (form.id) await adminApi.updateAmenity(form.id, payload)
    else await adminApi.createAmenity(payload)
    ui.success('Amenity saved')
    reset()
    await store.loadReference(true)
  } catch (err) {
    ui.error(err.message)
  } finally {
    saving.value = false
  }
}

async function remove(amenity) {
  const confirmed = await ui.confirm({
    title: `Delete ${amenity.name}?`,
    message: 'It will be removed from the filters and from any listing that uses it.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!confirmed) return
  try {
    await adminApi.deleteAmenity(amenity.id)
    ui.success('Amenity deleted')
    await store.loadReference(true)
  } catch (err) {
    ui.error(err.message)
  }
}

onMounted(() => store.loadReference())
</script>

<template>
  <div class="layout">
    <section class="panel">
      <div class="panel-head">
        <h3>{{ form.id ? 'Edit amenity' : 'Add an amenity' }}</h3>
        <button v-if="form.id" class="btn btn-ghost btn-sm" @click="reset">New</button>
      </div>

      <form @submit.prevent="save">
        <label class="field">
          <span class="label">Name</span>
          <input v-model="form.name" class="input" required placeholder="Swimming Pool" />
        </label>
        <label class="field">
          <span class="label">Category</span>
          <select v-model="form.category" class="select">
            <option v-for="category in CATEGORIES" :key="category" :value="category">
              {{ category }}
            </option>
          </select>
        </label>
        <label class="field">
          <span class="label">Icon key</span>
          <input v-model="form.icon" class="input" placeholder="pool" />
          <span class="field-hint">A short identifier the frontend can map to an icon.</span>
        </label>
        <label class="field">
          <span class="label">Display order</span>
          <input v-model="form.display_order" class="input" type="number" />
        </label>

        <button class="btn btn-primary btn-block" type="submit" :disabled="saving">
          {{ form.id ? 'Update amenity' : 'Add amenity' }}
        </button>
      </form>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>Amenities</h3>
        <span class="tiny muted">{{ store.amenities.length }} total</span>
      </div>

      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Category</th>
              <th class="num">Order</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="amenity in store.amenities" :key="amenity.id">
              <td>
                <strong>{{ amenity.name }}</strong>
                <div class="tiny muted">{{ amenity.slug }}</div>
              </td>
              <td><span class="badge badge-neutral">{{ amenity.category }}</span></td>
              <td class="num">{{ amenity.display_order }}</td>
              <td>
                <button class="btn btn-ghost btn-sm" @click="edit(amenity)">Edit</button>
                <button class="btn btn-ghost btn-sm danger" @click="remove(amenity)">Delete</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.layout { display: grid; grid-template-columns: 330px 1fr; gap: 20px; align-items: start; }
.danger { color: var(--danger-600); }

@media (max-width: 900px) {
  .layout { grid-template-columns: 1fr; }
}
</style>
