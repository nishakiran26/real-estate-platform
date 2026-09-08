<script setup>
import { reactive, ref } from 'vue'
import { publicApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const props = defineProps({
  property: { type: Object, required: true },
})
const emit = defineEmits(['close'])

const ui = useUiStore()
const done = ref(false)

// The reasons a listing is actually wrong, in the order they occur. "Already
// let" is first because it is by far the most common and the least accusatory:
// most wrong listings are stale, not dishonest.
const REASONS = [
  { value: 'ALREADY_RENTED', label: 'It has already been let or sold' },
  { value: 'INCORRECT_PRICE', label: 'The rent or price is wrong' },
  { value: 'INCORRECT_PHOTOS', label: 'The photos are not of this property' },
  { value: 'INCORRECT_LOCATION', label: 'The location is wrong' },
  { value: 'DUPLICATE', label: 'It is a duplicate of another listing' },
  { value: 'FAKE_LISTING', label: 'The property does not exist' },
  { value: 'FRAUD', label: 'I was asked for money or a deposit up front' },
  { value: 'OTHER', label: 'Something else' },
]

const form = reactive({
  reason: 'ALREADY_RENTED',
  details: '',
})

const { submit, submitting, error, fieldErrors } = useSubmit(
  () => publicApi.reportProperty(props.property.slug, { ...form }),
  {
    onSuccess: () => {
      done.value = true
      ui.success('Thank you. Our team will review this listing.')
    },
  },
)
</script>

<template>
  <div class="modal-backdrop" role="dialog" aria-modal="true" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-head">
        <div>
          <h3>Report this listing</h3>
          <p class="tiny muted mb-0">{{ property.name }}</p>
        </div>
        <button class="close" aria-label="Close" @click="emit('close')">×</button>
      </div>

      <div v-if="done" class="sent">
        <div class="sent-icon" aria-hidden="true">✓</div>
        <h4>Report sent</h4>
        <p class="small muted">
          Our team will look into it. Reports like yours are what keep the
          listings here worth trusting.
        </p>
        <button class="btn btn-primary" @click="emit('close')">Done</button>
      </div>

      <form v-else @submit.prevent="submit()">
        <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

        <fieldset class="reasons">
          <legend class="label">What is wrong with it?</legend>
          <label v-for="reason in REASONS" :key="reason.value" class="reason">
            <input v-model="form.reason" type="radio" :value="reason.value" />
            <span class="small">{{ reason.label }}</span>
          </label>
        </fieldset>

        <label class="field">
          <span class="label">Anything else we should know?</span>
          <textarea
            v-model="form.details"
            class="textarea"
            :class="{ 'input-error': fieldErrors.details }"
            rows="3"
            placeholder="For example, I called and was told it was let three weeks ago."
          ></textarea>
          <span v-if="fieldErrors.details" class="field-error">{{ fieldErrors.details }}</span>
        </label>

        <div class="modal-actions">
          <button class="btn btn-outline" type="button" @click="emit('close')">Cancel</button>
          <button class="btn btn-primary" type="submit" :disabled="submitting">
            <span v-if="submitting" class="spinner"></span>
            {{ submitting ? 'Sending…' : 'Send report' }}
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

.reasons { border: 0; padding: 0; margin: 0 0 14px; display: grid; gap: 7px; }
.reason { display: flex; align-items: center; gap: 9px; cursor: pointer; }

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
