<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { usePropertyStore } from '@/stores/property'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { PLACEHOLDER_IMAGE } from '@/utils/constants'
import {
  bhkLabel,
  formatAreaRange,
  formatPossession,
  formatPrice,
  formatPriceRange,
  humanise,
} from '@/utils/format'

const props = defineProps({
  property: { type: Object, required: true },
  compact: { type: Boolean, default: false },
  showCompare: { type: Boolean, default: true },
})

const store = usePropertyStore()
const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()

const cover = computed(() => props.property.cover_image || PLACEHOLDER_IMAGE)
const saved = computed(() => store.isFavorite(props.property.id) || props.property.is_favorite)
const comparing = computed(() => store.inCompare(props.property.id))

const isRental = computed(() => props.property.purpose === 'RENT')

// A renter scans for the monthly figure, a buyer for the price band. Same card,
// different headline number.
const priceLabel = computed(() =>
  isRental.value
    ? `${formatPrice(props.property.monthly_rent)}/month`
    : formatPriceRange(props.property.min_price, props.property.max_price),
)

const furnishingLabel = computed(() =>
  isRental.value && props.property.furnishing_type
    ? humanise(props.property.furnishing_type)
    : '',
)

const availabilityLabel = computed(() => {
  if (!isRental.value) return ''
  if (props.property.availability_status === 'RESERVED') return 'Reserved'
  return props.property.available_from
    ? `Available ${formatPossession(props.property.available_from)}`
    : 'Available now'
})
const areaLabel = computed(() => formatAreaRange(props.property.min_area, props.property.max_area))
const configLabel = computed(() => bhkLabel(props.property.bhk_options || []))

async function toggleSave() {
  const result = await store.toggleFavorite(props.property.id).catch((err) => {
    ui.error(err.message)
    return null
  })
  if (result === 'unauthenticated') {
    ui.info('Log in to save properties to your shortlist')
    router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
    return
  }
  if (result === 'added') ui.success('Saved to your shortlist')
  if (result === 'removed') ui.info('Removed from your shortlist')
}

function toggleCompare() {
  if (!store.toggleCompare(props.property)) {
    ui.info('You can compare up to 4 properties at a time')
  }
}
</script>

<template>
  <article class="property-card card" :class="{ compact }">
    <RouterLink
      :to="{ name: 'property-detail', params: { slug: property.slug } }"
      class="media"
      :aria-label="property.name"
    >
      <img :src="cover" :alt="property.name" loading="lazy" />
      <div class="media-badges">
        <span v-if="property.is_rera_verified" class="badge badge-rera">RERA verified</span>
        <span v-if="property.is_featured" class="badge badge-featured">Featured</span>
      </div>
      <span v-if="!isRental && property.construction_status" class="status-chip">
        {{ humanise(property.construction_status) }}
      </span>
      <span v-else-if="isRental" class="status-chip">For rent</span>
    </RouterLink>

    <div class="body">
      <div class="head">
        <h3 class="title">
          <RouterLink :to="{ name: 'property-detail', params: { slug: property.slug } }">
            {{ property.name }}
          </RouterLink>
        </h3>
        <button
          class="save-btn"
          :class="{ active: saved }"
          :aria-pressed="saved"
          :title="saved ? 'Remove from shortlist' : 'Save property'"
          @click.prevent="toggleSave"
        >
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
              d="M12 21s-7.5-4.6-9.6-9A5.4 5.4 0 0 1 12 6.2 5.4 5.4 0 0 1 21.6 12c-2.1 4.4-9.6 9-9.6 9z"
              :fill="saved ? 'currentColor' : 'none'"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>

      <p class="location truncate">
        {{ property.locality }}<span v-if="property.city">, {{ property.city }}</span>
      </p>

      <p v-if="property.builder_name" class="builder truncate tiny muted">
        by {{ property.builder_name }}
      </p>

      <div class="specs">
        <span v-if="configLabel">{{ configLabel }}</span>
        <span v-if="areaLabel !== '-'">{{ areaLabel }}</span>
        <span v-if="isRental && furnishingLabel">{{ furnishingLabel }}</span>
        <span v-else-if="!isRental">{{ formatPossession(property.possession_date) }}</span>
      </div>

      <p v-if="availabilityLabel" class="tiny availability">{{ availabilityLabel }}</p>

      <div class="footer">
        <div>
          <div class="price">{{ priceLabel }}</div>
          <div v-if="isRental && property.security_deposit" class="tiny muted">
            Deposit {{ formatPrice(property.security_deposit) }}
          </div>
          <div v-else-if="property.rera_number" class="tiny muted truncate">
            RERA {{ property.rera_number }}
          </div>
        </div>
        <button
          v-if="showCompare"
          class="btn btn-sm"
          :class="comparing ? 'btn-primary' : 'btn-outline'"
          @click.prevent="toggleCompare"
        >
          {{ comparing ? 'Added' : 'Compare' }}
        </button>
      </div>
    </div>
  </article>
</template>

<style scoped>
.property-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: transform var(--transition), box-shadow var(--transition), border-color var(--transition);
}
.property-card:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow);
  border-color: var(--brand-100);
}

.media {
  position: relative;
  display: block;
  aspect-ratio: 16 / 10;
  background: var(--brand-50);
  overflow: hidden;
}
.media img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 400ms ease;
}
.property-card:hover .media img { transform: scale(1.04); }

.media-badges {
  position: absolute;
  top: 10px;
  left: 10px;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  max-width: calc(100% - 20px);
}

.status-chip {
  position: absolute;
  bottom: 10px;
  left: 10px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: rgba(8, 38, 42, 0.82);
  color: var(--white);
  font-size: 0.72rem;
  font-weight: 600;
  backdrop-filter: blur(4px);
}

.body {
  padding: 15px 16px 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.head { display: flex; align-items: flex-start; gap: 10px; }

.title {
  margin: 0;
  font-size: 1.02rem;
  line-height: 1.32;
  flex: 1;
}
.title a { color: var(--ink-900); }
.title a:hover { color: var(--brand-700); }

.save-btn {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: 1px solid var(--ink-200);
  border-radius: 50%;
  background: var(--white);
  color: var(--ink-500);
  cursor: pointer;
  transition: all var(--transition);
}
.save-btn:hover { color: var(--danger-600); border-color: var(--danger-600); }
.save-btn.active { color: var(--danger-600); border-color: var(--danger-100); background: var(--danger-100); }

.location { margin: 0; font-size: 0.88rem; color: var(--ink-600); }
.availability { margin: 0; color: var(--keep, var(--success-600)); font-weight: 600; }
.builder { margin: 0; }

.specs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  margin-top: 4px;
  font-size: 0.82rem;
  color: var(--ink-700);
}
.specs span { position: relative; }
.specs span + span::before {
  content: '';
  position: absolute;
  left: -8px;
  top: 50%;
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--ink-300);
  transform: translateY(-50%);
}

.footer {
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--ink-100);
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 10px;
}
.footer .price { font-size: 1.06rem; }

.compact .media { aspect-ratio: 16 / 9; }
.compact .body { padding: 12px 13px; }
.compact .specs { display: none; }
</style>
