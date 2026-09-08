<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const ui = useUiStore()
const router = useRouter()

// request -> verify the code -> set a new password
const step = ref('request')
const form = reactive({ email: '', code: '', reset_token: '', new_password: '' })

const request = useSubmit(() => authApi.forgotPassword({ email: form.email.trim() }), {
  successMessage: 'If the account exists, a reset code has been sent',
  onSuccess: () => {
    step.value = 'verify'
  },
})

const verify = useSubmit(
  () =>
    authApi.verifyOtp({
      email: form.email.trim(),
      code: form.code.trim(),
      purpose: 'FORGOT_PASSWORD',
    }),
  {
    onSuccess: (res) => {
      form.reset_token = res.data.reset_token
      step.value = 'reset'
    },
  },
)

const reset = useSubmit(
  () =>
    authApi.resetPassword({
      reset_token: form.reset_token,
      new_password: form.new_password,
    }),
  {
    onSuccess: () => {
      ui.success('Password updated. Please log in.')
      router.push({ name: 'login' })
    },
  },
)
</script>

<template>
  <div class="auth-card panel">
    <h1>Reset your password</h1>

    <form v-if="step === 'request'" @submit.prevent="request.submit()">
      <p class="muted small">
        Enter the email on your account and we will send a verification code.
      </p>
      <div v-if="request.error.value" class="alert alert-error mb-2">{{ request.error.value }}</div>

      <label class="field">
        <span class="label">Email</span>
        <input v-model="form.email" class="input" type="email" required autocomplete="email" />
      </label>

      <button class="btn btn-primary btn-block" type="submit" :disabled="request.submitting.value">
        Send reset code
      </button>
    </form>

    <form v-else-if="step === 'verify'" @submit.prevent="verify.submit()">
      <p class="muted small">
        Enter the 6-digit code sent to <strong>{{ form.email }}</strong
        >.
      </p>
      <div v-if="verify.error.value" class="alert alert-error mb-2">{{ verify.error.value }}</div>

      <label class="field">
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

      <button class="btn btn-primary btn-block" type="submit" :disabled="verify.submitting.value">
        Verify code
      </button>
      <button class="btn btn-ghost btn-block mt-1" type="button" @click="request.submit()">
        Resend code
      </button>
    </form>

    <form v-else @submit.prevent="reset.submit()">
      <p class="muted small">Choose a new password for your account.</p>
      <div v-if="reset.error.value" class="alert alert-error mb-2">{{ reset.error.value }}</div>

      <label class="field">
        <span class="label">New password</span>
        <input
          v-model="form.new_password"
          class="input"
          type="password"
          required
          minlength="8"
          autocomplete="new-password"
        />
        <span class="field-hint">At least 8 characters.</span>
      </label>

      <button class="btn btn-primary btn-block" type="submit" :disabled="reset.submitting.value">
        Set new password
      </button>
    </form>

    <p class="small center mt-2 mb-0">
      <RouterLink :to="{ name: 'login' }">Back to log in</RouterLink>
    </p>
  </div>
</template>

<style scoped>
.auth-card h1 { font-size: 1.5rem; margin-bottom: 14px; }
.code-input { letter-spacing: 0.4em; text-align: center; font-size: 1.2rem; font-weight: 700; }
</style>
