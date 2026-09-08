<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { clientApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { useUiStore } from '@/stores/ui'
import { DOC_TYPES, MEDIA_TYPES, STATUS_VARIANT } from '@/utils/constants'
import { formatDate, humanise } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const store = usePropertyStore()
const ui = useUiStore()

const propertyId = computed(() => (route.params.id ? Number(route.params.id) : null))
const isEdit = computed(() => Boolean(propertyId.value))

const step = ref('basics') // basics | configurations | rera | media | review
const steps = [
  { key: 'basics', label: 'Basics' },
  { key: 'configurations', label: 'Configurations' },
  { key: 'rera', label: 'RERA' },
  { key: 'media', label: 'Photos & documents' },
  { key: 'review', label: 'Review' },
]

const form = reactive({
  name: '',
  project_name: '',
  builder_name: '',
  property_type: 'APARTMENT',
  construction_status: 'UNDER_CONSTRUCTION',
  city_id: '',
  locality_id: '',
  address: '',
  pincode: '',
  latitude: '',
  longitude: '',
  description: '',
  possession_date: '',
  meta_title: '',
  meta_description: '',
})

const configurations = ref([])
const amenityIds = ref([])
const rera = reactive({
  registration_number: '',
  authority: 'MahaRERA',
  registration_date: '',
  completion_date: '',
})

const property = ref(null)
const media = ref([])
const documents = ref([])
const statusHistory = ref([])

const loading = ref(false)
const saving = ref(false)
const uploading = ref(false)
const error = ref('')
const fieldErrors = ref({})

const localities = computed(() =>
  store.localities.filter((locality) => String(locality.city_id) === String(form.city_id)),
)

const canEdit = computed(
  () => !property.value || ['DRAFT', 'REJECTED', 'APPROVED', 'PUBLISHED'].includes(property.value.property_status),
)

function addConfiguration() {
  configurations.value.push({
    name: '',
    carpet_area: 0,
    built_up_area: 0,
    super_built_up_area: 0,
    bedrooms: 1,
    bathrooms: 1,
    balconies: 1,
    price: 0,
    min_price: 0,
    max_price: 0,
    availability: 'AVAILABLE',
    number_of_units: 0,
  })
}

function removeConfiguration(index) {
  configurations.value.splice(index, 1)
}

function toggleAmenity(id) {
  amenityIds.value = amenityIds.value.includes(id)
    ? amenityIds.value.filter((item) => item !== id)
    : [...amenityIds.value, id]
}

function buildPayload() {
  return {
    ...form,
    city_id: Number(form.city_id) || 0,
    locality_id: Number(form.locality_id) || 0,
    latitude: Number(form.latitude) || 0,
    longitude: Number(form.longitude) || 0,
    configurations: configurations.value.map((config) => ({
      ...config,
      carpet_area: Number(config.carpet_area) || 0,
      built_up_area: Number(config.built_up_area) || 0,
      super_built_up_area: Number(config.super_built_up_area) || 0,
      bedrooms: Number(config.bedrooms) || 0,
      bathrooms: Number(config.bathrooms) || 0,
      balconies: Number(config.balconies) || 0,
      price: Number(config.price) || 0,
      min_price: Number(config.min_price) || 0,
      max_price: Number(config.max_price) || 0,
      number_of_units: Number(config.number_of_units) || 0,
    })),
    amenity_ids: amenityIds.value,
    rera: rera.registration_number ? { ...rera } : null,
  }
}

async function save({ silent = false } = {}) {
  saving.value = true
  error.value = ''
  fieldErrors.value = {}
  try {
    const payload = buildPayload()
    const res = isEdit.value
      ? await clientApi.updateProperty(propertyId.value, payload)
      : await clientApi.createProperty(payload)

    property.value = res.data
    if (!silent) ui.success(isEdit.value ? 'Changes saved' : 'Draft created')

    // A new listing needs an id before photos can be attached to it.
    if (!isEdit.value) {
      router.replace({ name: 'client-property-edit', params: { id: res.data.id } })
    }
    return res.data
  } catch (err) {
    error.value = err.message
    fieldErrors.value = err.fields || {}
    ui.error(err.message)
    return null
  } finally {
    saving.value = false
  }
}

async function submitForReview() {
  const saved = await save({ silent: true })
  if (!saved) return

  const confirmed = await ui.confirm({
    title: 'Submit for review?',
    message:
      'An administrator will verify the project and RERA details before it is published. You will be notified of the outcome.',
    confirmLabel: 'Submit for review',
  })
  if (!confirmed) return

  try {
    await clientApi.submitProperty(propertyId.value)
    ui.success('Submitted for review')
    router.push({ name: 'client-properties' })
  } catch (err) {
    ui.error(err.message)
  }
}

async function uploadMedia(event, mediaType) {
  const file = event.target.files?.[0]
  if (!file || !propertyId.value) return
  uploading.value = true
  try {
    await clientApi.uploadMedia(propertyId.value, file, mediaType)
    await loadMedia()
    ui.success('Image uploaded')
  } catch (err) {
    ui.error(err.message)
  } finally {
    uploading.value = false
    event.target.value = ''
  }
}

async function uploadDocument(event, docType) {
  const file = event.target.files?.[0]
  if (!file || !propertyId.value) return
  uploading.value = true
  try {
    await clientApi.uploadDocument(propertyId.value, file, docType, file.name)
    await loadMedia()
    ui.success('Document uploaded')
  } catch (err) {
    ui.error(err.message)
  } finally {
    uploading.value = false
    event.target.value = ''
  }
}

async function setCover(item) {
  await clientApi.setCover(propertyId.value, item.id)
  await loadMedia()
  ui.success('Cover image updated')
}

async function deleteMedia(item) {
  const confirmed = await ui.confirm({ title: 'Delete this image?', danger: true, confirmLabel: 'Delete' })
  if (!confirmed) return
  await clientApi.deleteMedia(propertyId.value, item.id)
  await loadMedia()
}

async function deleteDocument(item) {
  const confirmed = await ui.confirm({ title: `Delete ${item.name}?`, danger: true, confirmLabel: 'Delete' })
  if (!confirmed) return
  await clientApi.deleteDocument(propertyId.value, item.id)
  await loadMedia()
}

async function loadMedia() {
  if (!propertyId.value) return
  const [m, d] = await Promise.all([
    clientApi.media(propertyId.value).catch(() => ({ data: [] })),
    clientApi.documents(propertyId.value).catch(() => ({ data: [] })),
  ])
  media.value = m.data || []
  documents.value = d.data || []
}

onMounted(async () => {
  await store.loadReference()
  if (!isEdit.value) {
    addConfiguration()
    return
  }

  loading.value = true
  try {
    const res = await clientApi.property(propertyId.value)
    const item = res.data
    property.value = item

    Object.assign(form, {
      name: item.name,
      project_name: item.project_name,
      builder_name: item.builder_name,
      property_type: item.property_type,
      construction_status: item.construction_status,
      city_id: item.city?.id || '',
      locality_id: item.locality?.id || '',
      address: item.address,
      pincode: item.pincode,
      latitude: item.latitude || '',
      longitude: item.longitude || '',
      description: item.description,
      possession_date: item.possession_date || '',
      meta_title: item.meta_title,
      meta_description: item.meta_description,
    })

    configurations.value = (item.configurations || []).map((config) => ({ ...config }))
    if (!configurations.value.length) addConfiguration()
    amenityIds.value = (item.amenities || []).map((amenity) => amenity.id)

    if (item.rera_details) {
      Object.assign(rera, {
        registration_number: item.rera_details.registration_number || '',
        authority: item.rera_details.authority || 'MahaRERA',
        registration_date: item.rera_details.registration_date || '',
        completion_date: item.rera_details.completion_date || '',
      })
    }

    await loadMedia()
    const history = await clientApi.statusHistory(propertyId.value).catch(() => ({ data: [] }))
    statusHistory.value = history.data || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="form-view">
    <div class="section-head">
      <div>
        <h2>{{ isEdit ? 'Edit property' : 'Add a property' }}</h2>
        <p>
          <template v-if="property">
            <span class="badge" :class="`badge-${STATUS_VARIANT[property.property_status] || 'neutral'}`">
              {{ humanise(property.property_status) }}
            </span>
          </template>
          <template v-else>Saved as a draft until you submit it for review</template>
        </p>
      </div>
      <div class="row">
        <RouterLink class="btn btn-ghost" :to="{ name: 'client-properties' }">Cancel</RouterLink>
        <button class="btn btn-outline" :disabled="saving" @click="save()">
          {{ saving ? 'Saving…' : 'Save draft' }}
        </button>
        <button v-if="isEdit" class="btn btn-primary" :disabled="saving" @click="submitForReview">
          Submit for review
        </button>
      </div>
    </div>

    <div v-if="property?.rejection_reason" class="alert alert-error mb-2">
      <strong>Rejected:</strong> {{ property.rejection_reason }}
      <div class="tiny mt-1">Correct the details above and submit again.</div>
    </div>

    <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

    <nav class="steps">
      <button
        v-for="item in steps"
        :key="item.key"
        class="step"
        :class="{ active: step === item.key }"
        @click="step = item.key"
      >
        {{ item.label }}
      </button>
    </nav>

    <!-- Basics ------------------------------------------------------------ -->
    <section v-show="step === 'basics'" class="panel">
      <div class="form-grid">
        <label class="field">
          <span class="label">Project name <span class="req">*</span></span>
          <input
            v-model="form.name"
            class="input"
            :class="{ 'input-error': fieldErrors.name }"
            required
          />
          <span v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</span>
        </label>

        <label class="field">
          <span class="label">Builder name shown on the listing</span>
          <input v-model="form.builder_name" class="input" />
        </label>

        <label class="field">
          <span class="label">Property type <span class="req">*</span></span>
          <select v-model="form.property_type" class="select">
            <option v-for="type in store.propertyTypes" :key="type.value" :value="type.value">
              {{ type.label }}
            </option>
          </select>
        </label>

        <label class="field">
          <span class="label">Construction status <span class="req">*</span></span>
          <select v-model="form.construction_status" class="select">
            <option
              v-for="status in store.constructionStatuses"
              :key="status.value"
              :value="status.value"
            >
              {{ status.label }}
            </option>
          </select>
        </label>

        <label class="field">
          <span class="label">City <span class="req">*</span></span>
          <select v-model="form.city_id" class="select" required @change="form.locality_id = ''">
            <option value="">Select a city</option>
            <option v-for="city in store.cities" :key="city.id" :value="city.id">
              {{ city.name }}
            </option>
          </select>
          <span v-if="fieldErrors.city_id" class="field-error">{{ fieldErrors.city_id }}</span>
        </label>

        <label class="field">
          <span class="label">Locality <span class="req">*</span></span>
          <select v-model="form.locality_id" class="select" required :disabled="!form.city_id">
            <option value="">Select a locality</option>
            <option v-for="locality in localities" :key="locality.id" :value="locality.id">
              {{ locality.name }}
            </option>
          </select>
          <span v-if="fieldErrors.locality_id" class="field-error">
            {{ fieldErrors.locality_id }}
          </span>
        </label>

        <label class="field">
          <span class="label">Pincode</span>
          <input v-model="form.pincode" class="input" maxlength="6" inputmode="numeric" />
        </label>

        <label class="field">
          <span class="label">Possession date</span>
          <input v-model="form.possession_date" class="input" type="date" />
        </label>

        <label class="field">
          <span class="label">Latitude</span>
          <input v-model="form.latitude" class="input" type="number" step="any" />
        </label>

        <label class="field">
          <span class="label">Longitude</span>
          <input v-model="form.longitude" class="input" type="number" step="any" />
        </label>
      </div>

      <label class="field">
        <span class="label">Address</span>
        <input v-model="form.address" class="input" placeholder="Street, landmark, area" />
      </label>

      <label class="field">
        <span class="label">Description</span>
        <textarea
          v-model="form.description"
          class="textarea"
          rows="6"
          placeholder="Describe the project, its location advantages and what makes it distinctive."
        ></textarea>
      </label>

      <div class="panel-head mt-2"><h4>Amenities</h4></div>
      <div class="amenity-grid">
        <label v-for="amenity in store.amenities" :key="amenity.id" class="check">
          <input
            type="checkbox"
            :checked="amenityIds.includes(amenity.id)"
            @change="toggleAmenity(amenity.id)"
          />
          <span>{{ amenity.name }}</span>
        </label>
      </div>
    </section>

    <!-- Configurations ---------------------------------------------------- -->
    <section v-show="step === 'configurations'" class="panel">
      <div class="panel-head">
        <h4>Unit configurations</h4>
        <button class="btn btn-outline btn-sm" @click="addConfiguration">Add configuration</button>
      </div>
      <p class="muted small">
        At least one configuration with a price is required before a project can be submitted.
      </p>

      <div v-for="(config, index) in configurations" :key="index" class="config card card-pad">
        <div class="row-between mb-1">
          <strong>Configuration {{ index + 1 }}</strong>
          <button
            v-if="configurations.length > 1"
            class="btn btn-ghost btn-sm"
            @click="removeConfiguration(index)"
          >
            Remove
          </button>
        </div>

        <div class="form-grid">
          <label class="field">
            <span class="label">Name <span class="req">*</span></span>
            <input v-model="config.name" class="input" placeholder="2 BHK" required />
          </label>
          <label class="field">
            <span class="label">Price (₹) <span class="req">*</span></span>
            <input v-model="config.price" class="input" type="number" min="0" required />
          </label>
          <label class="field">
            <span class="label">Carpet area (sq.ft.)</span>
            <input v-model="config.carpet_area" class="input" type="number" min="0" />
          </label>
          <label class="field">
            <span class="label">Built-up area (sq.ft.)</span>
            <input v-model="config.built_up_area" class="input" type="number" min="0" />
          </label>
          <label class="field">
            <span class="label">Super built-up (sq.ft.)</span>
            <input v-model="config.super_built_up_area" class="input" type="number" min="0" />
          </label>
          <label class="field">
            <span class="label">Bedrooms</span>
            <input v-model="config.bedrooms" class="input" type="number" min="0" max="20" />
          </label>
          <label class="field">
            <span class="label">Bathrooms</span>
            <input v-model="config.bathrooms" class="input" type="number" min="0" max="20" />
          </label>
          <label class="field">
            <span class="label">Balconies</span>
            <input v-model="config.balconies" class="input" type="number" min="0" max="20" />
          </label>
          <label class="field">
            <span class="label">Units available</span>
            <input v-model="config.number_of_units" class="input" type="number" min="0" />
          </label>
          <label class="field">
            <span class="label">Availability</span>
            <select v-model="config.availability" class="select">
              <option value="AVAILABLE">Available</option>
              <option value="COMING_SOON">Coming soon</option>
              <option value="SOLD_OUT">Sold out</option>
            </select>
          </label>
        </div>
      </div>
    </section>

    <!-- RERA -------------------------------------------------------------- -->
    <section v-show="step === 'rera'" class="panel">
      <div class="panel-head"><h4>RERA registration</h4></div>
      <p class="muted small">
        Buyers filter on RERA-verified projects. An administrator checks these details before the
        listing is marked verified.
      </p>

      <div class="form-grid">
        <label class="field">
          <span class="label">Registration number</span>
          <input v-model="rera.registration_number" class="input" placeholder="P52100000000" />
        </label>
        <label class="field">
          <span class="label">Authority</span>
          <input v-model="rera.authority" class="input" placeholder="MahaRERA" />
        </label>
        <label class="field">
          <span class="label">Registration date</span>
          <input v-model="rera.registration_date" class="input" type="date" />
        </label>
        <label class="field">
          <span class="label">Declared completion date</span>
          <input v-model="rera.completion_date" class="input" type="date" />
        </label>
      </div>

      <div v-if="property?.rera_details" class="alert alert-info tiny">
        Current verification status:
        <strong>{{ humanise(property.rera_details.verification_status) }}</strong>
        <template v-if="property.rera_details.verified_at">
          (updated {{ formatDate(property.rera_details.verified_at) }})
        </template>
      </div>

      <div class="panel-head mt-3"><h4>Search engine details</h4></div>
      <label class="field">
        <span class="label">Meta title</span>
        <input v-model="form.meta_title" class="input" maxlength="300" />
        <span class="field-hint">Left blank, one is generated from the project and location.</span>
      </label>
      <label class="field">
        <span class="label">Meta description</span>
        <textarea v-model="form.meta_description" class="textarea" rows="3" maxlength="500"></textarea>
      </label>
    </section>

    <!-- Media ------------------------------------------------------------- -->
    <section v-show="step === 'media'" class="panel">
      <div v-if="!isEdit" class="alert alert-info">
        Save the draft first — photos are attached to a saved project.
      </div>

      <template v-else>
        <div class="panel-head">
          <h4>Photos</h4>
          <span v-if="uploading" class="tiny muted">Uploading…</span>
        </div>

        <div class="uploaders">
          <label v-for="type in MEDIA_TYPES" :key="type.value" class="uploader">
            <input type="file" accept="image/jpeg,image/png,image/webp" @change="uploadMedia($event, type.value)" />
            <span class="uploader-icon" aria-hidden="true">＋</span>
            <span class="tiny">{{ type.label }}</span>
          </label>
        </div>
        <p class="tiny muted">JPG, PNG or WEBP up to 5 MB each.</p>

        <div v-if="media.length" class="media-grid mt-2">
          <figure v-for="item in media" :key="item.id" class="media-item">
            <img :src="item.file_url" :alt="item.media_type" loading="lazy" />
            <figcaption>
              <span class="tiny">{{ humanise(item.media_type) }}</span>
              <span v-if="item.is_cover_image" class="badge badge-featured">Cover</span>
            </figcaption>
            <div class="media-actions">
              <button v-if="!item.is_cover_image" class="btn btn-ghost btn-sm" @click="setCover(item)">
                Make cover
              </button>
              <button class="btn btn-ghost btn-sm danger" @click="deleteMedia(item)">Delete</button>
            </div>
          </figure>
        </div>

        <hr class="divider" />

        <div class="panel-head"><h4>Documents</h4></div>
        <div class="uploaders">
          <label v-for="type in DOC_TYPES" :key="type.value" class="uploader">
            <input type="file" accept="application/pdf" @change="uploadDocument($event, type.value)" />
            <span class="uploader-icon" aria-hidden="true">＋</span>
            <span class="tiny">{{ type.label }}</span>
          </label>
        </div>
        <p class="tiny muted">PDF up to 20 MB each.</p>

        <ul v-if="documents.length" class="doc-list mt-2">
          <li v-for="doc in documents" :key="doc.id">
            <span class="grow">
              <strong>{{ doc.name }}</strong>
              <span class="tiny muted"> · {{ humanise(doc.doc_type) }}</span>
            </span>
            <span class="tiny muted">{{ doc.download_count }} download(s)</span>
            <button class="btn btn-ghost btn-sm danger" @click="deleteDocument(doc)">Delete</button>
          </li>
        </ul>
      </template>
    </section>

    <!-- Review ------------------------------------------------------------ -->
    <section v-show="step === 'review'" class="panel">
      <div class="panel-head"><h4>Before you submit</h4></div>
      <ul class="checklist">
        <li :class="{ done: form.name && form.city_id && form.locality_id }">
          Project name and location
        </li>
        <li :class="{ done: configurations.some((c) => c.name && Number(c.price) > 0) }">
          At least one configuration with a price
        </li>
        <li :class="{ done: form.description?.length > 40 }">A description buyers can act on</li>
        <li :class="{ done: media.length > 0 }">At least one photo</li>
        <li :class="{ done: Boolean(rera.registration_number) }">RERA registration number</li>
      </ul>

      <div v-if="statusHistory.length" class="mt-3">
        <div class="panel-head"><h4>Approval history</h4></div>
        <ul class="history">
          <li v-for="entry in statusHistory" :key="entry.id">
            <span class="badge" :class="`badge-${STATUS_VARIANT[entry.to_status] || 'neutral'}`">
              {{ humanise(entry.to_status) }}
            </span>
            <span class="tiny muted">{{ entry.created_at }}</span>
            <span v-if="entry.remarks" class="small">{{ entry.remarks }}</span>
          </li>
        </ul>
      </div>

      <button class="btn btn-primary mt-3" :disabled="!isEdit || saving" @click="submitForReview">
        Submit for review
      </button>
    </section>
  </div>
</template>

<style scoped>
.steps {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  padding: 4px;
  background: var(--ink-100);
  border-radius: var(--radius);
  overflow-x: auto;
}
.step {
  padding: 8px 15px;
  border: 0;
  border-radius: var(--radius-sm);
  background: none;
  font-size: 0.86rem;
  font-weight: 600;
  color: var(--ink-600);
  cursor: pointer;
  white-space: nowrap;
}
.step.active { background: var(--white); color: var(--brand-800); box-shadow: var(--shadow-sm); }

.amenity-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 4px 16px; }

.config { margin-bottom: 14px; background: var(--ink-50); }

.uploaders { display: flex; gap: 10px; flex-wrap: wrap; }
.uploader {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 118px;
  height: 84px;
  border: 1px dashed var(--ink-300);
  border-radius: var(--radius);
  cursor: pointer;
  text-align: center;
  padding: 6px;
  transition: all var(--transition);
}
.uploader:hover { border-color: var(--brand-500); background: var(--brand-50); }
.uploader input { display: none; }
.uploader-icon { font-size: 1.2rem; color: var(--brand-600); }

.media-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 14px; }
.media-item { margin: 0; border: 1px solid var(--ink-200); border-radius: var(--radius); overflow: hidden; background: var(--white); }
.media-item img { width: 100%; height: 118px; object-fit: cover; }
.media-item figcaption { display: flex; align-items: center; justify-content: space-between; gap: 6px; padding: 7px 9px; }
.media-actions { display: flex; gap: 2px; padding: 0 5px 7px; }

.doc-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
.doc-list li { display: flex; align-items: center; gap: 10px; padding: 10px 12px; border: 1px solid var(--ink-200); border-radius: var(--radius); }

.checklist { list-style: none; margin: 0; padding: 0; display: grid; gap: 10px; }
.checklist li { display: flex; align-items: center; gap: 10px; color: var(--ink-600); }
.checklist li::before {
  content: '○';
  width: 20px; height: 20px;
  display: grid; place-items: center;
  border-radius: 50%;
  background: var(--ink-100);
  color: var(--ink-400);
  font-size: 0.7rem;
}
.checklist li.done { color: var(--ink-800); }
.checklist li.done::before { content: '✓'; background: var(--success-100); color: var(--success-600); font-weight: 800; }

.history { list-style: none; margin: 0; padding: 0; display: grid; gap: 9px; }
.history li { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }

.danger { color: var(--danger-600); }
</style>
