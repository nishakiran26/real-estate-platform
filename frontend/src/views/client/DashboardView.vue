<script setup>
import { onMounted, ref } from 'vue'
import { clientApi } from '@/services/api'
import { ANALYTICS_PERIODS, INQUIRY_STATUS_VARIANT } from '@/utils/constants'
import { formatNumber, humanise, relativeTime } from '@/utils/format'
import StatTile from '@/components/common/StatTile.vue'
import TrendChart from '@/components/common/TrendChart.vue'

const period = ref('30d')
const data = ref(null)
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await clientApi.dashboard({ period: period.value })
    data.value = res.data
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="stack">
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="row-between">
      <div>
        <h2 class="mb-0">Portfolio overview</h2>
        <p class="muted small mb-0">Views, leads and conversions across your projects</p>
      </div>
      <select v-model="period" class="select period" @change="load">
        <option v-for="option in ANALYTICS_PERIODS" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div class="grid grid-4">
      <StatTile label="Total properties" :value="data?.stats?.total_properties || 0" />
      <StatTile label="Published" :value="data?.stats?.active_properties || 0" tone="success" />
      <StatTile label="Awaiting review" :value="data?.stats?.pending_properties || 0" tone="warning" />
      <StatTile label="Rejected" :value="data?.stats?.rejected_properties || 0" tone="danger" />
    </div>

    <!-- One account can hold a flat for sale and one to let, and the two behave
         nothing alike. A single figure would hide which is which. -->
    <div v-if="data?.stats?.rental_properties" class="grid grid-4">
      <StatTile label="For sale" :value="data?.stats?.sale_properties || 0" />
      <StatTile label="To rent" :value="data?.stats?.rental_properties || 0" />
      <StatTile
        label="Rented out"
        :value="data?.stats?.rented_properties || 0"
        meta="Not shown in search"
      />
      <StatTile
        label="Lapsing soon"
        :value="data?.stats?.expiring_soon || 0"
        :tone="data?.stats?.expiring_soon ? 'warning' : undefined"
        meta="Renew from My properties"
      />
    </div>

    <div v-if="data?.stats?.pending_visits" class="alert alert-info mb-2">
      <strong>{{ data.stats.pending_visits }} viewing request(s) waiting on you.</strong>
      Renting rarely converts without a viewing.
      <RouterLink :to="{ name: 'client-visits' }">Answer them</RouterLink>
    </div>

    <div class="grid grid-4">
      <StatTile label="Total views" :value="data?.stats?.total_views || 0" />
      <StatTile label="Unique visitors" :value="data?.stats?.unique_visitors || 0" />
      <StatTile
        label="Inquiries"
        :value="data?.stats?.total_inquiries || 0"
        :meta="
          data?.stats?.rent_leads
            ? `${data.stats.buy_leads} to buy, ${data.stats.rent_leads} to rent`
            : undefined
        "
      />
      <StatTile
        label="Conversion rate"
        :value="(data?.stats?.conversion_rate || 0).toFixed(2)"
        suffix="%"
        meta="Inquiries per view"
      />
    </div>

    <div class="grid grid-4">
      <StatTile label="WhatsApp leads" :value="data?.stats?.whatsapp_leads || 0" />
      <StatTile label="Phone clicks" :value="data?.stats?.phone_clicks || 0" />
      <StatTile label="Brochure downloads" :value="data?.stats?.brochure_downloads || 0" />
      <StatTile label="Saved by buyers" :value="data?.stats?.favorites || 0" />
    </div>

    <div class="charts">
      <TrendChart title="Property views" :series="data?.charts?.views || []" />
      <TrendChart
        title="Inquiries"
        :series="data?.charts?.inquiries || []"
        color="var(--accent-600)"
      />
      <TrendChart
        title="WhatsApp clicks"
        :series="data?.charts?.whatsapp_clicks || []"
        color="#1f9d55"
      />
    </div>

    <div class="two-col">
      <section class="panel">
        <div class="panel-head">
          <h3>Top performing projects</h3>
          <RouterLink class="btn btn-ghost btn-sm" :to="{ name: 'client-properties' }">
            All projects
          </RouterLink>
        </div>

        <div v-if="data?.top_properties?.length" class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>Project</th>
                <th class="num">Views</th>
                <th class="num">Leads</th>
                <th class="num">Conversion</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in data.top_properties" :key="item.property_id">
                <td>
                  <RouterLink
                    :to="{ name: 'client-property-analytics', params: { id: item.property_id } }"
                  >
                    {{ item.name }}
                  </RouterLink>
                </td>
                <td class="num">{{ formatNumber(item.views) }}</td>
                <td class="num">{{ formatNumber(item.inquiries) }}</td>
                <td class="num">{{ item.conversion_rate.toFixed(1) }}%</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="muted small mb-0">No engagement recorded in this period yet.</p>
      </section>

      <section class="panel">
        <div class="panel-head">
          <h3>Latest leads</h3>
          <RouterLink class="btn btn-ghost btn-sm" :to="{ name: 'client-leads' }">
            All leads
          </RouterLink>
        </div>

        <ul v-if="data?.recent_inquiries?.length" class="lead-list">
          <li v-for="lead in data.recent_inquiries" :key="lead.id">
            <div class="grow">
              <strong>{{ lead.name }}</strong>
              <div class="tiny muted">
                {{ lead.property?.name }} · {{ relativeTime(lead.created_at) }}
              </div>
            </div>
            <span class="badge" :class="`badge-${INQUIRY_STATUS_VARIANT[lead.status] || 'neutral'}`">
              {{ humanise(lead.status) }}
            </span>
          </li>
        </ul>
        <p v-else class="muted small mb-0">No leads received yet.</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.period { width: auto; min-width: 160px; }

.charts { display: grid; grid-template-columns: repeat(auto-fit, minmax(290px, 1fr)); gap: 16px; }
.two-col { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 20px; align-items: start; }

.lead-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 10px; }
.lead-list li {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--ink-100);
}
.lead-list li:last-child { border-bottom: 0; padding-bottom: 0; }
</style>
