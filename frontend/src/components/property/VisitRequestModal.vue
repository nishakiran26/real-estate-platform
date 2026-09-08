<script setup>
import { computed, reactive, ref } from 'vue'
import { publicApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const props = defineProps({
  property: { type: Object, required: true },
})
const emit = defineEmits(['close'])

const auth = useAuthStore()
const ui = useUiStore()
const done = ref(false)

const SLOTS = [
  { value: 'MORNING', label: 'Morning (9am - 12pm)' },
  { value: 'AFTERNOON', label: 'Afternoon (12pm - 4pm)' },
  { value: 'EVENING', label: 'Evening (4pm - 8pm)' },
  { value: 'ANY', label: 'Any time' },
]

// A signed-in visitor should not retype what the platform already knows.
const form = reactive({
  name: auth.user?.full_name || '',
  mobile: auth.user?.mobile || '',
  email: auth.user?.email || '',
  visit_date: '',
  time_slot: 'EVENING',
  message: '',
  consent_contact: false,
})

const today = new Date().toISOString().slice(0, 10)
const maxDate = computed(() => {
  const d = new Date()
  d.setMonth(d.getMonth() + 3)
  return d.toISOString().slice(0, 10)
})

const { submit, submitting, error, fieldErrors } = useSubmit(
  () => publicApi.requestVisit(props.property.slug, { ...form }),
  {
    onSuccess: () => {
      done.value = true
      ui.success('Visit requested. The owner will confirm a time.')
    },
  },
)
</script>

<template>
  <div class="modal-backdrop" role="dialog" aria-modal="true" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-head">
        <div>
          <h3>Schedule a visit</h3>
          <p class="tiny muted mb-0">{{ property.name }}</p>
        </div>
        <button class="close" aria-label="Close" @click="emit('close')">×</button>
      </div>

      <div v-if="done" class="sent">
        <div class="sent-icon" aria-hidden="true">✓</div>
        <h4>Request sent</h4>
        <p class="small muted">
          The owner will confirm a time with you. You can track it under My visits.
        </p>
        <button class="btn btn-primary" @click="emit('close')">Done</button>
      </div>

      <form v-else @submit.prevent="submit()">
        <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

        <div class="form-grid">
          <label class="field">
            <span class="label">Preferred date <span class="req">*</span></span>
            <input
              v-model="form.visit_date"
              class="input"
              :class="{ 'input-error': fieldErrors.visit_date }"
              type="date"
              :min="today"
              :max="maxDate"
              required
            />
            <span v-if="fieldErrors.visit_date" class="field-error">
              {{ fieldErrors.visit_date }}
            </span>
          </label>

          <label class="field">
            <span class="label">Preferred time</span>
            <select v-model="form.time_slot" class="select">
              <option v-for="slot in SLOTS" :key="slot.value" :value="slot.value">
                {{ slot.label }}
              </option>
            </select>
          </label>
        </div>

        <label class="field">
          <span class="label">Name <span class="req">*</span></span>
          <input
            v-model="form.name"
            class="input"
            :class="{ 'input-error': fieldErrors.name }"
            required
          />
          <span v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</span>
        </label>

        <div class="form-grid">
          <label class="field">
            <span class="label">Mobile <span class="req">*</span></span>
            <input
              v-model="form.mobile"
              class="input"
              :class="{ 'input-error': fieldErrors.mobile }"
              inputmode="numeric"
              required
              placeholder="10-digit mobile number"
            />
            <span v-if="fieldErrors.mobile" class="field-error">{{ fieldErrors.mobile }}</span>
          </label>

          <label class="field">
            <span class="label">Email</span>
            <input v-model="form.email" class="input" type="email" />
          </label>
        </div>

        <label class="field">
          <span class="label">Anything the owner should know?</span>
          <textarea
            v-model="form.message"
            class="textarea"
            rows="2"
            placeholder="For example, I can only visit after 6pm on weekdays."
          ></textarea>
        </label>

        <label class="check">
          <input v-model="form.consent_contact" type="checkbox" required />
          <span class="tiny">
            I agree to be contacted about this visit by phone, SMS or WhatsApp.
          </span>
        </label>

        <div class="modal-actions">
          <button class="btn btn-outline" type="button" @click="emit('close')">Cancel</button>
          <button class="btn btn-primary" type="submit" :disabled="submitting">
            <span v-if="submitting" class="spinner"></span>
            {{ submitting ? 'Requesting…' : 'Request visit' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 32, 0.5);
  display: grid;
  place-items: center;
  padding: 20px;
  z-index: 1100;
  overflow-y: auto;
}

.modal {
  background: var(--white);
  border-radius: var(--radius-lg);
  padding: 24px;
  width: min(520px, 100%);
  box-shadow: var(--shadow-lg);
}

.modal-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}
.modal-head h3 { margin-bottom: 2px; }

.close {
  border: 0;
  background: none;
  font-size: 1.5rem;
  line-height: 1;
  color: var(--ink-400);
  cursor: pointer;
}
.close:hover { color: var(--ink-700); }

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  margin-top: 18px;
}

.sent { text-align: center; padding: 14px 6px 6px; }
.sent-icon {
  width: 44px;
  height: 44px;
  margin: 0 auto 12px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--success-100);
  color: var(--success-600);
  font-size: 1.25rem;
  font-weight: 800;
}
</style>
