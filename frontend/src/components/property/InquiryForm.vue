<script setup>
import { reactive, ref } from 'vue'
import { publicApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const props = defineProps({
  property: { type: Object, required: true },
  compact: { type: Boolean, default: false },
})

const auth = useAuthStore()
const ui = useUiStore()
const sent = ref(false)

// A signed-in buyer should not retype what the platform already knows.
const form = reactive({
  name: auth.user?.full_name || '',
  mobile: auth.user?.mobile || '',
  email: auth.user?.email || '',
  message: `I am interested in ${props.property.name}. Please share more details.`,
  configuration_id: props.property.configurations?.[0]?.id || null,
  budget: 0,
  preferred_time: '',
  consent_contact: false,
})

const { submit, submitting, error, fieldErrors } = useSubmit(
  () =>
    publicApi.createInquiry({
      property_id: props.property.id,
      name: form.name,
      mobile: form.mobile,
      email: form.email,
      message: form.message,
      configuration_id: form.configuration_id ? Number(form.configuration_id) : null,
      budget: Number(form.budget) || 0,
      preferred_time: form.preferred_time,
      source: 'WEB',
      consent_contact: form.consent_contact,
    }),
  {
    onSuccess: () => {
      sent.value = true
      ui.success('Your details have been sent to the builder')
    },
  },
)
</script>

<template>
  <div class="inquiry">
    <div v-if="sent" class="sent">
      <div class="sent-icon" aria-hidden="true">✓</div>
      <h4>Request sent</h4>
      <p class="small muted mb-0">
        {{ property.builder_name || 'The builder' }} has your details and will get in touch. You can
        track this in your inquiries.
      </p>
    </div>

    <form v-else @submit.prevent="submit">
      <h4 v-if="!compact">Contact the builder</h4>
      <p v-if="!compact" class="small muted">
        Your details go directly to {{ property.builder_name || 'the builder' }}.
      </p>

      <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

      <label class="field">
        <span class="label">Name <span class="req">*</span></span>
        <input
          v-model="form.name"
          class="input"
          :class="{ 'input-error': fieldErrors.name }"
          required
          placeholder="Your full name"
        />
        <span v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</span>
      </label>

      <label class="field">
        <span class="label">Mobile <span class="req">*</span></span>
        <input
          v-model="form.mobile"
          class="input"
          :class="{ 'input-error': fieldErrors.mobile }"
          required
          inputmode="numeric"
          placeholder="10-digit mobile number"
        />
        <span v-if="fieldErrors.mobile" class="field-error">{{ fieldErrors.mobile }}</span>
      </label>

      <label class="field">
        <span class="label">Email</span>
        <input
          v-model="form.email"
          class="input"
          :class="{ 'input-error': fieldErrors.email }"
          type="email"
          placeholder="you@example.com"
        />
        <span v-if="fieldErrors.email" class="field-error">{{ fieldErrors.email }}</span>
      </label>

      <label v-if="property.configurations?.length" class="field">
        <span class="label">Configuration of interest</span>
        <select v-model="form.configuration_id" class="select">
          <option :value="null">Any configuration</option>
          <option v-for="config in property.configurations" :key="config.id" :value="config.id">
            {{ config.name }} · {{ Math.round(config.carpet_area) }} sq.ft.
          </option>
        </select>
      </label>

      <label class="field">
        <span class="label">Preferred time to call</span>
        <select v-model="form.preferred_time" class="select">
          <option value="">Any time</option>
          <option value="MORNING">Morning (9am - 12pm)</option>
          <option value="AFTERNOON">Afternoon (12pm - 4pm)</option>
          <option value="EVENING">Evening (4pm - 8pm)</option>
        </select>
      </label>

      <label class="field">
        <span class="label">Message</span>
        <textarea v-model="form.message" class="textarea" rows="3"></textarea>
      </label>

      <!-- Consent is explicit and unbundled: the lead is not stored without it. -->
      <label class="check">
        <input v-model="form.consent_contact" type="checkbox" required />
        <span class="tiny">
          I agree to be contacted about this project by phone, SMS, WhatsApp or email. I can
          withdraw this at any time.
        </span>
      </label>
      <span v-if="fieldErrors.consent_contact" class="field-error">
        {{ fieldErrors.consent_contact }}
      </span>

      <button class="btn btn-primary btn-block mt-1" type="submit" :disabled="submitting">
        <span v-if="submitting" class="spinner"></span>
        {{ submitting ? 'Sending…' : 'Send my details' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.inquiry h4 { margin-bottom: 4px; }
.inquiry form > .field:first-of-type { margin-top: 14px; }

.sent { text-align: center; padding: 18px 6px; }
.sent-icon {
  width: 42px;
  height: 42px;
  margin: 0 auto 12px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--success-100);
  color: var(--success-600);
  font-size: 1.2rem;
  font-weight: 800;
}
</style>
