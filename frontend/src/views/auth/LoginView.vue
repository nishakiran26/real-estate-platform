<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()
const route = useRoute()

const mode = ref('password') // password | otp
const otpSent = ref(false)
const form = reactive({ email: '', mobile: '', password: '', code: '' })

function afterLogin() {
  const redirect = route.query.redirect ? String(route.query.redirect) : null
  ui.success(`Welcome back, ${auth.displayName.split(' ')[0]}`)
  router.push(redirect || auth.homeRoute)
}

const passwordLogin = useSubmit(
  () => auth.login({ email: form.email.trim(), password: form.password }),
  { onSuccess: afterLogin },
)

const otpLogin = useSubmit(
  () => auth.loginWithOtp({ mobile: form.mobile.trim(), code: form.code.trim() }),
  { onSuccess: afterLogin },
)

const sendCode = useSubmit(
  () => authApi.sendOtp({ mobile: form.mobile.trim(), purpose: 'LOGIN' }),
  {
    successMessage: 'Verification code sent',
    onSuccess: () => {
      otpSent.value = true
    },
  },
)
</script>

<template>
  <div class="auth-card panel">
    <h1>Log in</h1>
    <p class="muted small">
      New here?
      <RouterLink :to="{ name: 'register' }">Create an account</RouterLink>
    </p>

    <div class="tabs" role="tablist">
      <button
        class="tab"
        :class="{ active: mode === 'password' }"
        role="tab"
        :aria-selected="mode === 'password'"
        @click="mode = 'password'"
      >
        Email and password
      </button>
      <button
        class="tab"
        :class="{ active: mode === 'otp' }"
        role="tab"
        :aria-selected="mode === 'otp'"
        @click="mode = 'otp'"
      >
        Mobile OTP
      </button>
    </div>

    <form v-if="mode === 'password'" @submit.prevent="passwordLogin.submit()">
      <div v-if="passwordLogin.error.value" class="alert alert-error mb-2">
        {{ passwordLogin.error.value }}
      </div>

      <label class="field">
        <span class="label">Email</span>
        <input
          v-model="form.email"
          class="input"
          :class="{ 'input-error': passwordLogin.fieldErrors.value.email }"
          type="email"
          required
          autocomplete="email"
          placeholder="you@example.com"
        />
        <span v-if="passwordLogin.fieldErrors.value.email" class="field-error">
          {{ passwordLogin.fieldErrors.value.email }}
        </span>
      </label>

      <label class="field">
        <span class="label">Password</span>
        <input
          v-model="form.password"
          class="input"
          type="password"
          required
          autocomplete="current-password"
          placeholder="Your password"
        />
      </label>

      <div class="row-between mb-2">
        <RouterLink class="small" :to="{ name: 'forgot-password' }">Forgot password?</RouterLink>
      </div>

      <button class="btn btn-primary btn-block" type="submit" :disabled="passwordLogin.submitting.value">
        <span v-if="passwordLogin.submitting.value" class="spinner"></span>
        {{ passwordLogin.submitting.value ? 'Signing in…' : 'Log in' }}
      </button>
    </form>

    <form v-else @submit.prevent="otpSent ? otpLogin.submit() : sendCode.submit()">
      <div v-if="otpLogin.error.value || sendCode.error.value" class="alert alert-error mb-2">
        {{ otpLogin.error.value || sendCode.error.value }}
      </div>

      <label class="field">
        <span class="label">Mobile number</span>
        <input
          v-model="form.mobile"
          class="input"
          inputmode="numeric"
          required
          :disabled="otpSent"
          autocomplete="tel"
          placeholder="10-digit mobile number"
        />
      </label>

      <label v-if="otpSent" class="field">
        <span class="label">Verification code</span>
        <input
          v-model="form.code"
          class="input code-input"
          inputmode="numeric"
          maxlength="6"
          required
          placeholder="000000"
        />
        <span class="field-hint">
          In development the code is printed to the API server log.
        </span>
      </label>

      <button
        class="btn btn-primary btn-block"
        type="submit"
        :disabled="otpLogin.submitting.value || sendCode.submitting.value"
      >
        {{ otpSent ? 'Verify and log in' : 'Send verification code' }}
      </button>

      <button
        v-if="otpSent"
        class="btn btn-ghost btn-block mt-1"
        type="button"
        @click="sendCode.submit()"
      >
        Resend code
      </button>
    </form>
  </div>
</template>

<style scoped>
.auth-card h1 { font-size: 1.6rem; margin-bottom: 4px; }

.tabs {
  display: flex;
  gap: 4px;
  margin: 20px 0 18px;
  padding: 4px;
  background: var(--ink-100);
  border-radius: var(--radius);
}
.tab {
  flex: 1;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: none;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--ink-600);
  cursor: pointer;
  transition: all var(--transition);
}
.tab.active { background: var(--white); color: var(--brand-800); box-shadow: var(--shadow-sm); }

.code-input { letter-spacing: 0.4em; text-align: center; font-size: 1.2rem; font-weight: 700; }
</style>
