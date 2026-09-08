<script setup>
import { onMounted, ref } from 'vue'
import { adminApi } from '@/services/api'
import { ANALYTICS_PERIODS } from '@/utils/constants'
import { formatDateTime, formatNumber, relativeTime } from '@/utils/format'
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
    const res = await adminApi.dashboard({ period: period.value })
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
        <h2 class="mb-0">Platform overview</h2>
        <p class="muted small mb-0">Listings, people and engagement across the platform</p>
      </div>
      <select v-model="period" class="select period" @change="load">
        <option v-for="option in ANALYTICS_PERIODS" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div class="grid grid-4">
      <StatTile label="Total users" :value="data?.stats?.total_users || 0" />
      <StatTile label="Builders" :value="data?.stats?.total_clients || 0" />
      <StatTile label="Buyers" :value="data?.stats?.total_customers || 0" />
      <StatTile label="Total properties" :value="data?.stats?.total_properties || 0" />
    </div>

    <div class="grid grid-4">
      <StatTile label="Published" :value="data?.stats?.active_properties || 0" tone="success" />
      <StatTile
        label="Awaiting review"
        :value="data?.stats?.pending_properties || 0"
        tone="warning"
        meta="Needs an approval decision"
      />
      <StatTile label="Rejected" :value="data?.stats?.rejected_properties || 0" tone="danger" />
      <StatTile label="Drafts" :value="data?.stats?.draft_properties || 0" />
    </div>

    <!-- The catalogue splits two ways, and the totals above hide which half is
         growing. This row only appears once there is rental stock to report. -->
    <div v-if="data?.stats?.rental_properties" class="grid grid-4">
      <StatTile label="For sale" :value="data?.stats?.sale_properties || 0" />
      <StatTile label="To rent" :value="data?.stats?.rental_properties || 0" />
      <StatTile
        label="Let"
        :value="data?.stats?.rented_properties || 0"
        meta="Live listings that are off the market"
      />
      <StatTile
        label="Expiring soon"
        :value="data?.stats?.expiring_soon || 0"
        :tone="data?.stats?.expiring_soon ? 'warning' : undefined"
        meta="Rentals lapsing within 14 days"
      />
    </div>

    <div class="grid grid-4">
      <StatTile label="Property views" :value="data?.stats?.total_views || 0" />
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
    </div>

    <div class="two-col">
      <section class="panel">
        <div class="panel-head"><h3>Most viewed projects</h3></div>
        <div v-if="data?.most_viewed?.length" class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>Project</th>
                <th class="num">Views</th>
                <th class="num">Leads</th>
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
                <td class="num">{{ formatNumber(item.inquiries) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="muted small mb-0">No views recorded in this period.</p>
      </section>

      <section class="panel">
        <div class="panel-head"><h3>Most contacted projects</h3></div>
        <div v-if="data?.most_contacted?.length" class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>Project</th>
                <th class="num">Leads</th>
                <th class="num">Conversion</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in data.most_contacted" :key="item.property_id">
                <td>
                  <RouterLink :to="{ name: 'admin-property-detail', params: { id: item.property_id } }">
                    {{ item.name }}
                  </RouterLink>
                </td>
                <td class="num">{{ formatNumber(item.inquiries) }}</td>
                <td class="num">{{ item.conversion_rate.toFixed(1) }}%</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="muted small mb-0">No inquiries in this period.</p>
      </section>

      <section class="panel">
        <div class="panel-head"><h3>Popular cities</h3></div>
        <ul v-if="data?.popular_cities?.length" class="rank">
          <li v-for="(item, index) in data.popular_cities" :key="item.slug">
            <span class="rank-no">{{ index + 1 }}</span>
            <span class="grow">{{ item.label }}</span>
            <strong>{{ formatNumber(item.count) }}</strong>
          </li>
        </ul>
        <p v-else class="muted small mb-0">No data yet.</p>
      </section>

      <section class="panel">
        <div class="panel-head"><h3>Most searched localities</h3></div>
        <ul v-if="data?.searched_localities?.length" class="rank">
          <li v-for="(item, index) in data.searched_localities" :key="item.slug || index">
            <span class="rank-no">{{ index + 1 }}</span>
            <span class="grow">{{ item.label }}</span>
            <strong>{{ formatNumber(item.count) }}</strong>
          </li>
        </ul>
        <p v-else class="muted small mb-0">No searches recorded yet.</p>
      </section>

      <section class="panel">
        <div class="panel-head">
          <h3>Recent registrations</h3>
          <RouterLink class="btn btn-ghost btn-sm" :to="{ name: 'admin-users' }">All users</RouterLink>
        </div>
        <ul v-if="data?.recent_registrations?.length" class="rank">
          <li v-for="user in data.recent_registrations" :key="user.id">
            <span class="grow">
              <strong>{{ user.full_name }}</strong>
              <div class="tiny muted">{{ user.email }}</div>
            </span>
            <span class="tiny muted">{{ relativeTime(user.created_at) }}</span>
          </li>
        </ul>
        <p v-else class="muted small mb-0">No new registrations.</p>
      </section>

      <section class="panel">
        <div class="panel-head">
          <h3>Recent inquiries</h3>
          <RouterLink class="btn btn-ghost btn-sm" :to="{ name: 'admin-inquiries' }">
            All leads
          </RouterLink>
        </div>
        <ul v-if="data?.recent_inquiries?.length" class="rank">
          <li v-for="lead in data.recent_inquiries" :key="lead.id">
            <span class="grow">
              <strong>{{ lead.name }}</strong>
              <div class="tiny muted">{{ lead.property?.name }}</div>
            </span>
            <span class="tiny muted">{{ formatDateTime(lead.created_at) }}</span>
          </li>
        </ul>
        <p v-else class="muted small mb-0">No inquiries yet.</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.period { width: auto; min-width: 160px; }
.charts { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 16px; }
.two-col { display: grid; grid-template-columns: repeat(auto-fit, minmax(330px, 1fr)); gap: 20px; align-items: start; }

.rank { list-style: none; margin: 0; padding: 0; display: grid; gap: 10px; }
.rank li { display: flex; align-items: center; gap: 12px; padding-bottom: 9px; border-bottom: 1px solid var(--ink-100); }
.rank li:last-child { border-bottom: 0; padding-bottom: 0; }
.rank-no {
  width: 22px; height: 22px;
  display: grid; place-items: center;
  border-radius: 50%;
  background: var(--brand-50);
  color: var(--brand-800);
  font-size: 0.72rem;
  font-weight: 800;
}
</style>
