<script setup>
import { reactive, ref } from 'vue'
import { authApi, customerApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useSubmit } from '@/composables/useAsync'

const auth = useAuthStore()

const profile = reactive({
  full_name: auth.user?.full_name || '',
  email: auth.user?.email || '',
  mobile: auth.user?.mobile || '',
  avatar_url: auth.user?.avatar_url || '',
})

const passwords = reactive({ current_password: '', new_password: '' })
const passwordDone = ref(false)

const saveProfile = useSubmit(() => customerApi.updateProfile(profile), {
  successMessage: 'Profile updated',
  onSuccess: () => auth.loadProfile(),
})

const changePassword = useSubmit(() => authApi.changePassword(passwords), {
  successMessage: 'Password changed',
  onSuccess: () => {
    passwords.current_password = ''
    passwords.new_password = ''
    passwordDone.value = true
  },
})
</script>

<template>
  <div class="profile-grid">
    <section class="panel">
      <div class="panel-head"><h3>My profile</h3></div>

      <div v-if="saveProfile.error.value" class="alert alert-error mb-2">
        {{ saveProfile.error.value }}
      </div>

      <form @submit.prevent="saveProfile.submit()">
        <label class="field">
          <span class="label">Full name</span>
          <input v-model="profile.full_name" class="input" required />
        </label>

        <div class="form-grid">
          <label class="field">
            <span class="label">Email</span>
            <input v-model="profile.email" class="input" type="email" />
            <span class="field-hint">
              <template v-if="auth.user?.email_verified">Verified</template>
              <template v-else>
                Not verified —
                <RouterLink :to="{ name: 'verify-otp' }">verify now</RouterLink>
              </template>
            </span>
          </label>

          <label class="field">
            <span class="label">Mobile</span>
            <input v-model="profile.mobile" class="input" inputmode="numeric" />
            <span class="field-hint">
              <template v-if="auth.user?.mobile_verified">Verified</template>
              <template v-else>
                Not verified —
                <RouterLink :to="{ name: 'verify-otp' }">verify now</RouterLink>
              </template>
            </span>
          </label>
        </div>

        <p class="tiny muted">
          Changing your email or mobile clears its verified status until you confirm the new value.
        </p>

        <button class="btn btn-primary" type="submit" :disabled="saveProfile.submitting.value">
          Save changes
        </button>
      </form>
    </section>

    <section class="panel">
      <div class="panel-head"><h3>Change password</h3></div>

      <div v-if="changePassword.error.value" class="alert alert-error mb-2">
        {{ changePassword.error.value }}
      </div>
      <div v-if="passwordDone" class="alert alert-success mb-2">
        Password changed. Other devices have been signed out.
      </div>

      <form @submit.prevent="changePassword.submit()">
        <label class="field">
          <span class="label">Current password</span>
          <input
            v-model="passwords.current_password"
            class="input"
            type="password"
            required
            autocomplete="current-password"
          />
        </label>

        <label class="field">
          <span class="label">New password</span>
          <input
            v-model="passwords.new_password"
            class="input"
            type="password"
            required
            minlength="8"
            autocomplete="new-password"
          />
          <span class="field-hint">At least 8 characters.</span>
        </label>

        <button class="btn btn-primary" type="submit" :disabled="changePassword.submitting.value">
          Update password
        </button>
      </form>
    </section>
  </div>
</template>

<style scoped>
.profile-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 20px; align-items: start; }
</style>
