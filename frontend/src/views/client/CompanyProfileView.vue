<script setup>
import { onMounted, reactive, ref } from 'vue'
import { clientApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const ui = useUiStore()

const form = reactive({
  company_name: '',
  company_logo_url: '',
  company_description: '',
  website: '',
  contact_phone: '',
  contact_email: '',
  address: '',
  city: '',
  state: '',
  pincode: '',
  gstin: '',
  rera_reg_no: '',
})

const loading = ref(true)
const uploading = ref(false)

onMounted(async () => {
  try {
    const res = await clientApi.profile()
    Object.assign(form, {
      company_name: res.data.company_name || '',
      company_logo_url: res.data.company_logo_url || '',
      company_description: res.data.company_description || '',
      website: res.data.website || '',
      contact_phone: res.data.contact_phone || '',
      contact_email: res.data.contact_email || '',
      address: res.data.address || '',
      city: res.data.city || '',
      state: res.data.state || '',
      pincode: res.data.pincode || '',
      gstin: res.data.gstin || '',
      rera_reg_no: res.data.rera_reg_no || '',
    })
  } catch (err) {
    ui.error(err.message)
  } finally {
    loading.value = false
  }
})

const { submit, submitting, error, fieldErrors } = useSubmit(
  () => clientApi.updateProfile(form),
  { successMessage: 'Company profile updated' },
)

async function uploadLogo(event) {
  const file = event.target.files?.[0]
  if (!file) return
  uploading.value = true
  try {
    const res = await clientApi.uploadLogo(file)
    form.company_logo_url = res.data.url
    ui.success('Logo uploaded. Save to apply it.')
  } catch (err) {
    ui.error(err.message)
  } finally {
    uploading.value = false
    event.target.value = ''
  }
}
</script>

<template>
  <div class="panel">
    <div class="panel-head">
      <h3>Company profile</h3>
    </div>
    <p class="muted small">
      This is what buyers see on your listings and on your builder page.
    </p>

    <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

    <form @submit.prevent="submit()">
      <div class="logo-row">
        <div class="logo-preview">
          <img v-if="form.company_logo_url" :src="form.company_logo_url" alt="Company logo" />
          <span v-else aria-hidden="true">{{ (form.company_name || '?').charAt(0) }}</span>
        </div>
        <div>
          <label class="btn btn-outline btn-sm">
            {{ uploading ? 'Uploading…' : 'Upload logo' }}
            <input
              type="file"
              accept="image/jpeg,image/png,image/webp"
              hidden
              @change="uploadLogo"
            />
          </label>
          <p class="tiny muted mt-1 mb-0">JPG, PNG or WEBP up to 5 MB.</p>
        </div>
      </div>

      <label class="field">
        <span class="label">Company name <span class="req">*</span></span>
        <input
          v-model="form.company_name"
          class="input"
          :class="{ 'input-error': fieldErrors.company_name }"
          required
        />
        <span v-if="fieldErrors.company_name" class="field-error">
          {{ fieldErrors.company_name }}
        </span>
      </label>

      <label class="field">
        <span class="label">About the company</span>
        <textarea
          v-model="form.company_description"
          class="textarea"
          rows="5"
          placeholder="Years in business, completed projects, what you are known for."
        ></textarea>
      </label>

      <div class="form-grid">
        <label class="field">
          <span class="label">Website</span>
          <input v-model="form.website" class="input" type="url" placeholder="https://" />
        </label>
        <label class="field">
          <span class="label">Company RERA registration</span>
          <input v-model="form.rera_reg_no" class="input" />
        </label>
        <label class="field">
          <span class="label">Contact phone</span>
          <input v-model="form.contact_phone" class="input" inputmode="numeric" />
          <span class="field-hint">Used for the WhatsApp button on your listings.</span>
        </label>
        <label class="field">
          <span class="label">Contact email</span>
          <input v-model="form.contact_email" class="input" type="email" />
        </label>
        <label class="field">
          <span class="label">City</span>
          <input v-model="form.city" class="input" />
        </label>
        <label class="field">
          <span class="label">State</span>
          <input v-model="form.state" class="input" />
        </label>
        <label class="field">
          <span class="label">Pincode</span>
          <input v-model="form.pincode" class="input" maxlength="6" />
        </label>
        <label class="field">
          <span class="label">GSTIN</span>
          <input v-model="form.gstin" class="input" />
        </label>
      </div>

      <label class="field">
        <span class="label">Registered address</span>
        <textarea v-model="form.address" class="textarea" rows="3"></textarea>
      </label>

      <button class="btn btn-primary" type="submit" :disabled="submitting || loading">
        {{ submitting ? 'Saving…' : 'Save profile' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.logo-row { display: flex; align-items: center; gap: 18px; margin-bottom: 22px; }

.logo-preview {
  width: 74px;
  height: 74px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--ink-200);
  display: grid;
  place-items: center;
  background: var(--brand-50);
  color: var(--brand-800);
  font-size: 1.8rem;
  font-weight: 800;
  font-family: var(--font-display);
  overflow: hidden;
  flex-shrink: 0;
}
.logo-preview img { width: 100%; height: 100%; object-fit: cover; }
</style>
