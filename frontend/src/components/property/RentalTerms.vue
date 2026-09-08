<script setup>
import { computed } from 'vue'
import { formatDate, formatPrice, humanise } from '@/utils/format'

const props = defineProps({
  rental: { type: Object, required: true },
})

// A renter's real monthly outgoing is rent plus maintenance, unless it is
// already included. Showing it saves them doing the arithmetic.
const monthlyTotal = computed(() => {
  const rent = Number(props.rental.monthly_rent) || 0
  const maintenance = props.rental.maintenance_included
    ? 0
    : Number(props.rental.maintenance_amount) || 0
  return rent + maintenance
})

const showsTotal = computed(
  () => !props.rental.maintenance_included && Number(props.rental.maintenance_amount) > 0,
)

const leaseLabel = computed(() => {
  const min = Number(props.rental.min_lease_months) || 0
  const max = Number(props.rental.max_lease_months) || 0
  if (!min && !max) return 'Negotiable'
  if (!max || max === min) return `${min} months`
  return `${min} - ${max} months`
})

const availabilityLabel = computed(() => {
  if (!props.rental.available_from) return 'Immediately'
  const from = new Date(props.rental.available_from)
  return from <= new Date() ? 'Immediately' : formatDate(props.rental.available_from)
})
</script>

<template>
  <div class="rental-terms">
    <div class="headline">
      <div>
        <span class="stat-label">Monthly rent</span>
        <div class="rent">{{ formatPrice(rental.monthly_rent) }}</div>
      </div>
      <div>
        <span class="stat-label">Security deposit</span>
        <div class="deposit">{{ formatPrice(rental.security_deposit) }}</div>
      </div>
    </div>

    <p v-if="showsTotal" class="total tiny">
      About <strong>{{ formatPrice(monthlyTotal) }}</strong> a month including maintenance.
    </p>

    <dl class="terms">
      <div>
        <dt>Maintenance</dt>
        <dd>
          <template v-if="rental.maintenance_included">Included in the rent</template>
          <template v-else-if="rental.maintenance_amount">
            {{ formatPrice(rental.maintenance_amount) }} a month, extra
          </template>
          <template v-else>Not specified</template>
        </dd>
      </div>
      <div>
        <dt>Available from</dt>
        <dd>{{ availabilityLabel }}</dd>
      </div>
      <div>
        <dt>Furnishing</dt>
        <dd>{{ humanise(rental.furnishing_type) || 'Not specified' }}</dd>
      </div>
      <div>
        <dt>Lease duration</dt>
        <dd>{{ leaseLabel }}</dd>
      </div>
      <div>
        <dt>Notice period</dt>
        <dd>
          <template v-if="rental.notice_period_days">
            {{ rental.notice_period_days }} days
          </template>
          <template v-else>Negotiable</template>
        </dd>
      </div>
      <div>
        <dt>Suitable for</dt>
        <dd>{{ humanise(rental.tenant_preference) || 'Anyone' }}</dd>
      </div>
      <div>
        <dt>Pets</dt>
        <dd>{{ rental.pet_allowed ? 'Allowed' : 'Not allowed' }}</dd>
      </div>
    </dl>

    <div v-if="rental.included_items?.length" class="included">
      <span class="stat-label">Comes with</span>
      <ul class="items">
        <li v-for="item in rental.included_items" :key="item">
          <span class="tick" aria-hidden="true">✓</span>{{ item }}
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.headline {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--ink-200);
}
.rent { font-size: 1.5rem; font-weight: 800; color: var(--ink-900); letter-spacing: -0.01em; }
.deposit { font-size: 1.2rem; font-weight: 700; color: var(--ink-800); }

.total { margin: 12px 0 0; color: var(--ink-600); }
.total strong { color: var(--ink-900); }

.terms {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 16px;
  margin: 18px 0 0;
}
.terms div { display: flex; flex-direction: column; gap: 3px; }
.terms dt {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--ink-500);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.terms dd { margin: 0; font-size: 0.94rem; font-weight: 600; color: var(--ink-900); }

.included { margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--ink-200); }
.items {
  list-style: none;
  padding: 0;
  margin: 10px 0 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 9px;
}
.items li { display: flex; align-items: center; gap: 8px; font-size: 0.9rem; }
.tick {
  width: 19px;
  height: 19px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--brand-50);
  color: var(--brand-700);
  font-size: 0.66rem;
  font-weight: 800;
  flex-shrink: 0;
}
</style>
