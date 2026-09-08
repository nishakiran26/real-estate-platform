<script setup>
import { onMounted, reactive, ref } from 'vue'
import { authApi, customerApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'
import { formatDateTime } from '@/utils/format'

const auth = useAuthStore()
const ui = useUiStore()

const profile = reactive({
  full_name: auth.user?.full_name || '',
  email: auth.user?.email || '',
  mobile: auth.user?.mobile || '',
})
const passwords = reactive({ current_password: '', new_password: '' })
const history = ref([])

onMounted(async () => {
  try {
    const res = await authApi.loginHistory()
    history.value = res.data || []
  } catch {
    history.value = []
  }
})

const saveProfile = useSubmit(() => customerApi.updateProfile(profile), {
  successMessage: 'Account details updated',
  onSuccess: () => auth.loadProfile(),
})

const changePassword = useSubmit(() => authApi.changePassword(passwords), {
  successMessage: 'Password changed',
  onSuccess: () => {
    passwords.current_password = ''
    passwords.new_password = ''
  },
})

async function logoutEverywhere() {
  const confirmed = await ui.confirm({
    title: 'Sign out of all devices?',
    message: 'Every active session will be ended, including this one.',
    confirmLabel: 'Sign out everywhere',
    danger: true,
  })
  if (!confirmed) return
  await auth.logout(true)
  window.location.href = '/login'
}
</script>

<template>
  <div class="stack">
    <div class="two-col">
      <section class="panel">
        <div class="panel-head"><h3>Account details</h3></div>
        <div v-if="saveProfile.error.value" class="alert alert-error mb-2">
          {{ saveProfile.error.value }}
        </div>

        <form @submit.prevent="saveProfile.submit()">
          <label class="field">
            <span class="label">Your name</span>
            <input v-model="profile.full_name" class="input" required />
          </label>
          <label class="field">
            <span class="label">Email</span>
            <input v-model="profile.email" class="input" type="email" />
          </label>
          <label class="field">
            <span class="label">Mobile</span>
            <input v-model="profile.mobile" class="input" inputmode="numeric" />
          </label>
          <button class="btn btn-primary" type="submit" :disabled="saveProfile.submitting.value">
            Save
          </button>
        </form>
      </section>

      <section class="panel">
        <div class="panel-head"><h3>Password</h3></div>
        <div v-if="changePassword.error.value" class="alert alert-error mb-2">
          {{ changePassword.error.value }}
        </div>

        <form @submit.prevent="changePassword.submit()">
          <label class="field">
            <span class="label">Current password</span>
            <input v-model="passwords.current_password" class="input" type="password" required />
          </label>
          <label class="field">
            <span class="label">New password</span>
            <input
              v-model="passwords.new_password"
              class="input"
              type="password"
              required
              minlength="8"
            />
          </label>
          <div class="row">
            <button
              class="btn btn-primary"
              type="submit"
              :disabled="changePassword.submitting.value"
            >
              Update password
            </button>
            <button class="btn btn-ghost" type="button" @click="logoutEverywhere">
              Sign out everywhere
            </button>
          </div>
        </form>
      </section>
    </div>

    <section class="panel">
      <div class="panel-head"><h3>Recent sign-ins</h3></div>
      <div v-if="history.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>When</th>
              <th>Method</th>
              <th>Device</th>
              <th>IP address</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in history" :key="entry.id">
              <td class="small">{{ formatDateTime(entry.login_at) }}</td>
              <td class="small">{{ entry.method }}</td>
              <td class="small">{{ entry.device || 'Unknown' }}</td>
              <td class="tiny muted">{{ entry.ip_address }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="muted small mb-0">No sign-in activity recorded yet.</p>
    </section>
  </div>
</template>

<style scoped>
.two-col { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 20px; align-items: start; }
</style>
