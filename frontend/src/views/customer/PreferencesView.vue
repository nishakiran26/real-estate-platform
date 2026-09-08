<script setup>
import { computed, onMounted, reactive } from 'vue'
import { customerApi } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { usePropertyStore } from '@/stores/property'
import { useUiStore } from '@/stores/ui'
import { useSubmit } from '@/composables/useAsync'

const auth = useAuthStore()
const store = usePropertyStore()
const ui = useUiStore()

const prefs = reactive({
  preferred_city: '',
  preferred_locality: '',
  min_budget: 0,
  max_budget: 0,
  property_type: '',
  bhk: 0,
  purpose: 'BUY',
  looking_to_buy: false,
  looking_to_rent: false,
  monthly_rent_min: 0,
  monthly_rent_max: 0,
  preferred_furnishing: '',
})

const channels = reactive({
  email_enabled: true,
  sms_enabled: true,
  whatsapp_enabled: false,
  in_app_enabled: true,
})

const consent = reactive({
  property_updates: false,
  new_project_alerts: false,
  promotional: false,
  whatsapp: false,
  email: true,
})

const localityOptions = computed(() => {
  const city = store.cities.find((item) => item.name === prefs.preferred_city)
  return city ? store.localities.filter((locality) => locality.city_id === city.id) : []
})

onMounted(async () => {
  await store.loadReference()
  const [p, n] = await Promise.all([
    customerApi.preferences().catch(() => null),
    customerApi.notificationPreferences().catch(() => null),
  ])
  if (p?.data) Object.assign(prefs, p.data)
  if (n?.data) Object.assign(channels, n.data)
  if (auth.user?.consent) Object.assign(consent, auth.user.consent)
})

const savePrefs = useSubmit(
  () =>
    customerApi.savePreferences({
      ...prefs,
      min_budget: Number(prefs.min_budget) || 0,
      max_budget: Number(prefs.max_budget) || 0,
      bhk: Number(prefs.bhk) || 0,
      monthly_rent_min: Number(prefs.monthly_rent_min) || 0,
      monthly_rent_max: Number(prefs.monthly_rent_max) || 0,
    }),
  { successMessage: 'Search preferences saved' },
)

const saveChannels = useSubmit(() => customerApi.saveNotificationPreferences(channels), {
  successMessage: 'Notification channels updated',
})

const saveConsent = useSubmit(() => customerApi.updateConsent(consent), {
  successMessage: 'Communication preferences updated',
  onSuccess: () => auth.loadProfile(),
})

async function unsubscribeAll() {
  const confirmed = await ui.confirm({
    title: 'Unsubscribe from all marketing?',
    message:
      'You will stop receiving project alerts and promotional messages. Transactional messages such as OTP and inquiry updates still apply.',
    confirmLabel: 'Unsubscribe',
    danger: true,
  })
  if (!confirmed) return
  await customerApi.unsubscribe()
  Object.assign(consent, {
    property_updates: false,
    new_project_alerts: false,
    promotional: false,
    whatsapp: false,
  })
  await auth.loadProfile()
  ui.success('You have been unsubscribed from marketing communication')
}
</script>

<template>
  <div class="stack">
    <section class="panel">
      <div class="panel-head">
        <h3>What are you looking for?</h3>
      </div>
      <p class="muted small">
        These drive the recommendations on your dashboard and the alerts you receive.
      </p>

      <form @submit.prevent="savePrefs.submit()">
        <!-- The intent decides which catalogue the dashboard recommends from.
             Both can be on: it is one account, not two. -->
        <fieldset class="intents">
          <legend class="label">I am looking to</legend>
          <label class="check">
            <input v-model="prefs.looking_to_rent" type="checkbox" />
            <span>Rent a home</span>
          </label>
          <label class="check">
            <input v-model="prefs.looking_to_buy" type="checkbox" />
            <span>Buy a property</span>
          </label>
        </fieldset>

        <div v-if="prefs.looking_to_rent" class="form-grid">
          <label class="field">
            <span class="label">Rent from (₹ per month)</span>
            <input v-model="prefs.monthly_rent_min" class="input" type="number" min="0" />
          </label>
          <label class="field">
            <span class="label">Rent up to (₹ per month)</span>
            <input v-model="prefs.monthly_rent_max" class="input" type="number" min="0" />
          </label>
          <label class="field">
            <span class="label">Furnishing</span>
            <select v-model="prefs.preferred_furnishing" class="select">
              <option value="">No preference</option>
              <option value="UNFURNISHED">Unfurnished</option>
              <option value="SEMI_FURNISHED">Semi-furnished</option>
              <option value="FULLY_FURNISHED">Fully furnished</option>
            </select>
          </label>
        </div>

        <div class="form-grid">
          <label class="field">
            <span class="label">Preferred city</span>
            <select v-model="prefs.preferred_city" class="select">
              <option value="">No preference</option>
              <option v-for="city in store.cities" :key="city.id" :value="city.name">
                {{ city.name }}
              </option>
            </select>
          </label>

          <label class="field">
            <span class="label">Preferred locality</span>
            <select v-model="prefs.preferred_locality" class="select" :disabled="!localityOptions.length">
              <option value="">No preference</option>
              <option
                v-for="locality in localityOptions"
                :key="locality.id"
                :value="locality.name"
              >
                {{ locality.name }}
              </option>
            </select>
          </label>

          <label class="field">
            <span class="label">Minimum budget (₹)</span>
            <input v-model="prefs.min_budget" class="input" type="number" min="0" />
          </label>

          <label class="field">
            <span class="label">Maximum budget (₹)</span>
            <input v-model="prefs.max_budget" class="input" type="number" min="0" />
          </label>

          <label class="field">
            <span class="label">Property type</span>
            <select v-model="prefs.property_type" class="select">
              <option value="">Any</option>
              <option v-for="type in store.propertyTypes" :key="type.value" :value="type.value">
                {{ type.label }}
              </option>
            </select>
          </label>

          <label class="field">
            <span class="label">Bedrooms</span>
            <select v-model="prefs.bhk" class="select">
              <option :value="0">Any</option>
              <option v-for="n in 5" :key="n" :value="n">{{ n }} BHK</option>
            </select>
          </label>

          <label class="field">
            <span class="label">Purpose</span>
            <select v-model="prefs.purpose" class="select">
              <option value="BUY">Buy to live in</option>
              <option value="INVEST">Invest</option>
              <option value="RENT">Rent</option>
            </select>
          </label>
        </div>

        <button class="btn btn-primary" type="submit" :disabled="savePrefs.submitting.value">
          Save preferences
        </button>
      </form>
    </section>

    <section class="panel">
      <div class="panel-head"><h3>How we contact you</h3></div>
      <p class="muted small">
        Each consent is separate and can be withdrawn at any time. Withdrawing consent never affects
        transactional messages such as OTP codes.
      </p>

      <div class="two-col">
        <form @submit.prevent="saveConsent.submit()">
          <span class="label">Marketing consent</span>
          <label class="check">
            <input v-model="consent.property_updates" type="checkbox" />
            <span>Updates on projects I have shown interest in</span>
          </label>
          <label class="check">
            <input v-model="consent.new_project_alerts" type="checkbox" />
            <span>New project launches in my preferred locations</span>
          </label>
          <label class="check">
            <input v-model="consent.promotional" type="checkbox" />
            <span>Offers and promotional communication</span>
          </label>
          <label class="check">
            <input v-model="consent.whatsapp" type="checkbox" />
            <span>Contact me on WhatsApp</span>
          </label>
          <label class="check">
            <input v-model="consent.email" type="checkbox" />
            <span>Contact me by email</span>
          </label>

          <div class="row mt-2">
            <button class="btn btn-primary" type="submit" :disabled="saveConsent.submitting.value">
              Save consent
            </button>
            <button class="btn btn-ghost" type="button" @click="unsubscribeAll">
              Unsubscribe from all
            </button>
          </div>
        </form>

        <form @submit.prevent="saveChannels.submit()">
          <span class="label">Delivery channels</span>
          <label class="check">
            <input v-model="channels.email_enabled" type="checkbox" />
            <span>Email</span>
          </label>
          <label class="check">
            <input v-model="channels.sms_enabled" type="checkbox" />
            <span>SMS</span>
          </label>
          <label class="check">
            <input v-model="channels.whatsapp_enabled" type="checkbox" />
            <span>WhatsApp</span>
          </label>
          <label class="check">
            <input v-model="channels.in_app_enabled" type="checkbox" />
            <span>In-app notifications</span>
          </label>

          <button
            class="btn btn-primary mt-2"
            type="submit"
            :disabled="saveChannels.submitting.value"
          >
            Save channels
          </button>
        </form>
      </div>
    </section>
  </div>
</template>

<style scoped>
.intents {
  border: 1px solid var(--line-200);
  border-radius: var(--radius-md);
  padding: 12px 14px;
  margin-bottom: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px 20px;
  align-items: center;
}
.intents legend {
  padding: 0 6px;
}
.two-col { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 26px; }
</style>
