<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()

// Mobile first: it is the channel builders actually call back on.
const channel = ref('mobile')
const form = reactive({ code: '' })
const sent = ref(false)

const target = computed(() =>
  channel.value === 'mobile' ? auth.user?.mobile : auth.user?.email,
)
const purpose = computed(() =>
  channel.value === 'mobile' ? 'MOBILE_VERIFY' : 'EMAIL_VERIFY',
)
const alreadyVerified = computed(() =>
  channel.value === 'mobile' ? auth.user?.mobile_verified : auth.user?.email_verified,
)

const send = useSubmit(
  () =>
    authApi.sendOtp(
      channel.value === 'mobile'
        ? { mobile: auth.user?.mobile, purpose: purpose.value }
        : { email: auth.user?.email, purpose: purpose.value },
    ),
  {
    successMessage: 'Verification code sent',
    onSuccess: () => {
      sent.value = true
    },
  },
)

const verify = useSubmit(
  () =>
    authApi.verifyOtp(
      channel.value === 'mobile'
        ? { mobile: auth.user?.mobile, code: form.code.trim(), purpose: purpose.value }
        : { email: auth.user?.email, code: form.code.trim(), purpose: purpose.value },
    ),
  {
    onSuccess: async () => {
      await auth.loadProfile()
      ui.success('Verified successfully')
      form.code = ''
      sent.value = false
    },
  },
)

function skip() {
  router.push(auth.homeRoute)
}
</script>

<template>
  <div class="auth-card panel">
    <h1>Verify your account</h1>
    <p class="muted small">
      Verifying your contact details helps builders reach you and keeps your account secure.
    </p>

    <div class="tabs" role="tablist">
      <button
        class="tab"
        :class="{ active: channel === 'mobile' }"
        role="tab"
        @click="channel = 'mobile'; sent = false"
      >
        Mobile
        <span v-if="auth.user?.mobile_verified" class="badge badge-success">✓</span>
      </button>
      <button
        class="tab"
        :class="{ active: channel === 'email' }"
        role="tab"
        @click="channel = 'email'; sent = false"
      >
        Email
        <span v-if="auth.user?.email_verified" class="badge badge-success">✓</span>
      </button>
    </div>

    <div v-if="alreadyVerified" class="alert alert-success">
      Your {{ channel }} is already verified.
    </div>

    <form v-else @submit.prevent="sent ? verify.submit() : send.submit()">
      <div v-if="send.error.value || verify.error.value" class="alert alert-error mb-2">
        {{ send.error.value || verify.error.value }}
      </div>

      <p class="small">
        We will send a 6-digit code to <strong>{{ target }}</strong
        >.
      </p>

      <label v-if="sent" class="field">
        <span class="label">Verification code</span>
        <input
          v-model="form.code"
          class="input code-input"
          inputmode="numeric"
          maxlength="6"
          required
        />
        <span class="field-hint">In development the code is printed to the API server log.</span>
      </label>

      <button
        class="btn btn-primary btn-block"
        type="submit"
        :disabled="send.submitting.value || verify.submitting.value"
      >
        {{ sent ? 'Verify' : 'Send code' }}
      </button>

      <button v-if="sent" class="btn btn-ghost btn-block mt-1" type="button" @click="send.submit()">
        Resend code
      </button>
    </form>

    <button class="btn btn-ghost btn-block mt-2" @click="skip">Continue to my dashboard</button>
  </div>
</template>

<style scoped>
.auth-card h1 { font-size: 1.5rem; margin-bottom: 4px; }

.tabs {
  display: flex;
  gap: 4px;
  margin: 18px 0;
  padding: 4px;
  background: var(--ink-100);
  border-radius: var(--radius);
}
.tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: none;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--ink-600);
  cursor: pointer;
}
.tab.active { background: var(--white); color: var(--brand-800); box-shadow: var(--shadow-sm); }

.code-input { letter-spacing: 0.4em; text-align: center; font-size: 1.2rem; font-weight: 700; }
</style>
