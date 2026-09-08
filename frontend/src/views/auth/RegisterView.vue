<script setup>
import { computed, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()

/**
 * What someone says they are, rather than which database role they map to. A
 * buyer and a renter are both customers; an owner, builder and agent all get a
 * listing account and differ only by profile type.
 */
const PROFILES = [
  {
    value: 'BUYER',
    label: 'Buy a property',
    hint: 'Search, shortlist and contact builders',
    listing: false,
  },
  {
    value: 'RENTER',
    label: 'Rent a place to live',
    hint: 'Find rentals and schedule visits',
    listing: false,
  },
  {
    value: 'OWNER',
    label: 'Rent out or sell my property',
    hint: 'List your own flat and receive enquiries',
    listing: true,
  },
  {
    value: 'BUILDER',
    label: 'Builder or developer',
    hint: 'List projects and manage leads',
    listing: true,
  },
  {
    value: 'AGENT',
    label: 'Real estate agent',
    hint: 'List on behalf of owners',
    listing: true,
  },
]

const form = reactive({
  profile_type: 'BUYER',
  full_name: '',
  email: '',
  mobile: '',
  password: '',
  company_name: '',
  consent_property_updates: true,
  consent_new_project_alerts: true,
  consent_promotional: false,
  consent_whatsapp: false,
})

// A customer can be looking to buy and to rent at once — one account, both
// intents. Listing accounts are defined by their profile type instead.
const intents = reactive({ BUY: true, RENT: false })

const selected = computed(() => PROFILES.find((p) => p.value === form.profile_type))
const isListingAccount = computed(() => selected.value?.listing === true)

// Choosing "rent a place" pre-selects renting, so the intent boxes agree with
// what was just picked instead of contradicting it.
watch(
  () => form.profile_type,
  (value) => {
    if (value === 'BUYER') {
      intents.BUY = true
      intents.RENT = false
    } else if (value === 'RENTER') {
      intents.BUY = false
      intents.RENT = true
    }
  },
)

const intentError = computed(() =>
  !isListingAccount.value && !intents.BUY && !intents.RENT
    ? 'Choose whether you are looking to buy, to rent, or both'
    : '',
)

const { submit, submitting, error, fieldErrors } = useSubmit(
  () => {
    if (intentError.value) throw new Error(intentError.value)
    return auth.register({
      ...form,
      full_name: form.full_name.trim(),
      email: form.email.trim(),
      mobile: form.mobile.trim(),
      company_name: isListingAccount.value ? form.company_name.trim() : '',
      intents: isListingAccount.value
        ? []
        : Object.keys(intents).filter((key) => intents[key]),
    })
  },
  {
    onSuccess: () => {
      ui.success('Account created. Please verify your contact details.')
      router.push({ name: 'verify-otp' })
    },
  },
)
</script>

<template>
  <div class="auth-card panel">
    <h1>Create your account</h1>
    <p class="muted small">
      Already registered?
      <RouterLink :to="{ name: 'login' }">Log in</RouterLink>
    </p>

    <div class="field intent-block">
      <label class="label" for="profile-type">I want to <span class="req">*</span></label>
      <select id="profile-type" v-model="form.profile_type" class="select">
        <option v-for="profile in PROFILES" :key="profile.value" :value="profile.value">
          {{ profile.label }}
        </option>
      </select>
      <span class="field-hint">{{ selected?.hint }}</span>
    </div>

    <!-- A buyer who is also open to renting stays on one account. -->
    <fieldset v-if="!isListingAccount" class="intents">
      <legend class="label">I am looking to</legend>
      <label class="check">
        <input v-model="intents.BUY" type="checkbox" />
        <span>Buy a property</span>
      </label>
      <label class="check">
        <input v-model="intents.RENT" type="checkbox" />
        <span>Rent a property</span>
      </label>
      <p v-if="intentError" class="field-error">{{ intentError }}</p>
      <p v-else class="tiny muted mb-0">
        Pick both if you are open to either. You can change this any time, and you never need a
        second account.
      </p>
    </fieldset>

    <p v-else class="tiny muted upgrade-note">
      You will get a listing account and can post properties once an administrator approves them.
    </p>

    <form @submit.prevent="submit()">
      <div v-if="error" class="alert alert-error mb-2">{{ error }}</div>

      <label class="field">
        <span class="label">Full name <span class="req">*</span></span>
        <input
          v-model="form.full_name"
          class="input"
          :class="{ 'input-error': fieldErrors.full_name }"
          required
          autocomplete="name"
        />
        <span v-if="fieldErrors.full_name" class="field-error">{{ fieldErrors.full_name }}</span>
      </label>

      <label v-if="isListingAccount" class="field">
        <span class="label">
          {{ form.profile_type === "OWNER" ? "Your name or company (optional)" : "Company name" }}
          <span v-if="form.profile_type !== 'OWNER'" class="req">*</span>
        </span>
        <input
          v-model="form.company_name"
          class="input"
          :class="{ 'input-error': fieldErrors.company_name }"
          :required="form.profile_type !== 'OWNER'"
          placeholder="Registered company or firm name"
        />
        <span v-if="fieldErrors.company_name" class="field-error">
          {{ fieldErrors.company_name }}
        </span>
      </label>

      <div class="form-grid">
        <label class="field">
          <span class="label">Email <span class="req">*</span></span>
          <input
            v-model="form.email"
            class="input"
            :class="{ 'input-error': fieldErrors.email }"
            type="email"
            required
            autocomplete="email"
          />
          <span v-if="fieldErrors.email" class="field-error">{{ fieldErrors.email }}</span>
        </label>

        <label class="field">
          <span class="label">Mobile <span class="req">*</span></span>
          <input
            v-model="form.mobile"
            class="input"
            :class="{ 'input-error': fieldErrors.mobile }"
            inputmode="numeric"
            required
            autocomplete="tel"
            placeholder="10-digit number"
          />
          <span v-if="fieldErrors.mobile" class="field-error">{{ fieldErrors.mobile }}</span>
        </label>
      </div>

      <label class="field">
        <span class="label">Password <span class="req">*</span></span>
        <input
          v-model="form.password"
          class="input"
          :class="{ 'input-error': fieldErrors.password }"
          type="password"
          required
          minlength="8"
          autocomplete="new-password"
        />
        <span class="field-hint">At least 8 characters.</span>
        <span v-if="fieldErrors.password" class="field-error">{{ fieldErrors.password }}</span>
      </label>

      <fieldset class="consent">
        <legend class="label">Keep me posted about</legend>
        <p class="tiny muted mb-1">
          Each of these is a separate consent and can be withdrawn from your account at any time.
        </p>
        <label class="check">
          <input v-model="form.consent_property_updates" type="checkbox" />
          <span>Updates on projects I have shown interest in</span>
        </label>
        <label class="check">
          <input v-model="form.consent_new_project_alerts" type="checkbox" />
          <span>New project launches in my preferred locations</span>
        </label>
        <label class="check">
          <input v-model="form.consent_promotional" type="checkbox" />
          <span>Offers and promotional communication</span>
        </label>
        <label class="check">
          <input v-model="form.consent_whatsapp" type="checkbox" />
          <span>Contact me on WhatsApp</span>
        </label>
      </fieldset>

      <button class="btn btn-primary btn-block" type="submit" :disabled="submitting">
        <span v-if="submitting" class="spinner"></span>
        {{ submitting ? 'Creating account…' : 'Create account' }}
      </button>

      <p class="tiny muted center mt-2 mb-0">
        By continuing you agree to our
        <RouterLink :to="{ name: 'static-page', params: { slug: 'terms-and-conditions' } }">
          terms
        </RouterLink>
        and
        <RouterLink :to="{ name: 'static-page', params: { slug: 'privacy-policy' } }">
          privacy policy </RouterLink
        >.
      </p>
    </form>
  </div>
</template>

<style scoped>
.auth-card h1 { font-size: 1.6rem; margin-bottom: 4px; }

.intent-block { margin: 20px 0 14px; }

.intents {
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  padding: 14px 16px;
  margin: 0 0 20px;
  background: var(--brand-50);
}
.intents legend { padding: 0 6px; }
.intents .check:last-of-type { margin-bottom: 8px; }

.upgrade-note {
  margin: 0 0 20px;
  padding: 11px 14px;
  border-radius: var(--radius);
  background: var(--brand-50);
  border: 1px solid var(--brand-100);
}

.consent {
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  padding: 14px 16px;
  margin: 0 0 18px;
}
.consent legend { padding: 0 6px; }
</style>
