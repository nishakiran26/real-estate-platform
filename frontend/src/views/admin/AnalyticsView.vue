<script setup>
import { onMounted, reactive, ref } from 'vue'
import { adminApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { ANALYTICS_PERIODS } from '@/utils/constants'
import { formatNumber } from '@/utils/format'
import StatTile from '@/components/common/StatTile.vue'
import TrendChart from '@/components/common/TrendChart.vue'

const store = usePropertyStore()

const filters = reactive({ period: '30d', from: '', to: '', city_id: '', client_id: '' })
const data = ref(null)
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = { period: filters.period }
    if (filters.period === 'custom') {
      params.from = filters.from
      params.to = filters.to
    }
    if (filters.city_id) params.city_id = filters.city_id
    if (filters.client_id) params.client_id = filters.client_id

    const res = await adminApi.analytics(params)
    data.value = res.data
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await store.loadReference()
  load()
})
</script>

<template>
  <div class="stack">
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <section class="panel">
      <div class="panel-head"><h3>Report filters</h3></div>
      <div class="filter-row">
        <label class="field mb-0">
          <span class="label">Period</span>
          <select v-model="filters.period" class="select" @change="load">
            <option v-for="option in ANALYTICS_PERIODS" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>

        <template v-if="filters.period === 'custom'">
          <label class="field mb-0">
            <span class="label">From</span>
            <input v-model="filters.from" class="input" type="date" />
          </label>
          <label class="field mb-0">
            <span class="label">To</span>
            <input v-model="filters.to" class="input" type="date" />
          </label>
        </template>

        <label class="field mb-0">
          <span class="label">City</span>
          <select v-model="filters.city_id" class="select">
            <option value="">All cities</option>
            <option v-for="city in store.cities" :key="city.id" :value="city.id">
              {{ city.name }}
            </option>
          </select>
        </label>

        <label class="field mb-0">
          <span class="label">Builder ID</span>
          <input v-model="filters.client_id" class="input" type="number" placeholder="Any" />
        </label>

        <button class="btn btn-primary" @click="load">Apply</button>
      </div>
    </section>

    <div class="grid grid-4">
      <StatTile label="Property views" :value="data?.stats?.total_views || 0" />
      <StatTile label="Unique visitors" :value="data?.stats?.unique_visitors || 0" />
      <StatTile label="Inquiries" :value="data?.stats?.total_inquiries || 0" />
      <StatTile label="WhatsApp leads" :value="data?.stats?.whatsapp_leads || 0" />
    </div>

    <div class="charts">
      <TrendChart title="Daily visitors" :series="data?.charts?.visitors || []" />
      <TrendChart title="Property views" :series="data?.charts?.views || []" color="var(--info-600)" />
      <TrendChart title="New leads" :series="data?.charts?.inquiries || []" color="var(--accent-600)" />
      <TrendChart
        title="New registrations"
        :series="data?.charts?.registrations || []"
        color="#7c4dcf"
      />
      <TrendChart
        title="WhatsApp clicks"
        :series="data?.charts?.whatsapp_clicks || []"
        color="#1f9d55"
      />
    </div>

    <section class="panel">
      <div class="panel-head"><h3>Top projects by engagement</h3></div>
      <div v-if="data?.most_viewed?.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Project</th>
              <th class="num">Views</th>
              <th class="num">Unique</th>
              <th class="num">Inquiries</th>
              <th class="num">WhatsApp</th>
              <th class="num">Saved</th>
              <th class="num">Conversion</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in data.most_viewed" :key="item.property_id">
              <td>
                <RouterLink :to="{ name: 'admin-property-detail', params: { id: item.property_id } }">
                  {{ item.name }}
                </RouterLink>
              </td>
              <td class="num">{{ formatNumber(item.views) }}</td>
              <td class="num">{{ formatNumber(item.unique_visitors) }}</td>
              <td class="num">{{ formatNumber(item.inquiries) }}</td>
              <td class="num">{{ formatNumber(item.whatsapp_clicks) }}</td>
              <td class="num">{{ formatNumber(item.favorites) }}</td>
              <td class="num">{{ item.conversion_rate.toFixed(1) }}%</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="muted small mb-0">No engagement data for this period.</p>
    </section>
  </div>
</template>

<style scoped>
.filter-row { display: flex; gap: 12px; align-items: flex-end; flex-wrap: wrap; }
.filter-row .field { min-width: 150px; }
.charts { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 16px; }
</style>
