<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { publicApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useSeo } from '@/composables/useSeo'
import { EVENT_TYPES, RERA_STATUS_VARIANT } from '@/utils/constants'
import {
  formatArea, formatDate, formatNumber, formatPossession, formatPrice, formatPriceRange, humanise,
  pricePerSqft,
} from '@/utils/format'
import ImageGallery from '@/components/property/ImageGallery.vue'
import RentalTerms from '@/components/property/RentalTerms.vue'
import VisitRequestModal from '@/components/property/VisitRequestModal.vue'
import ReportListingModal from '@/components/property/ReportListingModal.vue'
import InquiryForm from '@/components/property/InquiryForm.vue'
import PropertyGrid from '@/components/property/PropertyGrid.vue'

const route = useRoute()
const router = useRouter()
const store = usePropertyStore()
const auth = useAuthStore()
const ui = useUiStore()
const { setMeta, propertyJsonLd } = useSeo()

const property = ref(null)
const similar = ref([])
const loading = ref(true)
const error = ref('')

const isRental = computed(() => property.value?.purpose === 'RENT')
const saved = computed(() => (property.value ? store.isFavorite(property.value.id) : false))
const comparing = computed(() => (property.value ? store.inCompare(property.value.id) : false))
const visitOpen = ref(false)
const reportOpen = ref(false)

// "About the builder" is wrong for a flat let by the person who lives upstairs.
const posterHeading = computed(() => {
  switch (property.value?.builder?.profile_type) {
    case 'OWNER': return 'About the owner'
    case 'AGENT': return 'About the agent'
    default: return 'About the builder'
  }
})

// Only a level that means something is shown. "Mobile verified" is true of
// every account that finished signing up, so it is not a badge.
const verificationBadge = computed(() => {
  switch (property.value?.builder?.verification_level) {
    case 'ADMIN_VERIFIED': return 'Verified'
    case 'DOCUMENT_VERIFIED': return 'Documents verified'
    default: return ''
  }
})

const brochure = computed(() =>
  property.value?.documents?.find((doc) => doc.doc_type === 'BROCHURE'),
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await publicApi.property(route.params.slug)
    property.value = res.data
    applySeo()
    loadSimilar()
  } catch (err) {
    error.value = err.message
    property.value = null
  } finally {
    loading.value = false
  }
}

async function loadSimilar() {
  try {
    const res = await publicApi.similar(route.params.slug, 4)
    similar.value = res.data || []
  } catch {
    similar.value = []
  }
}

function applySeo() {
  const item = property.value
  if (!item) return
  setMeta({
    title: item.meta_title || `${item.name}, ${item.locality?.name} ${item.city?.name}`,
    description:
      item.meta_description ||
      `${item.name} in ${item.locality?.name}, ${item.city?.name}. RERA registered project with configurations, price and possession details.`,
    image: item.cover_image,
    type: 'article',
    jsonLd: propertyJsonLd(item),
  })
}

async function toggleSave() {
  const result = await store.toggleFavorite(property.value.id).catch((err) => {
    ui.error(err.message)
    return null
  })
  if (result === 'unauthenticated') {
    ui.info('Log in to save this property')
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  if (result === 'added') ui.success('Saved to your shortlist')
  if (result === 'removed') ui.info('Removed from your shortlist')
}

function toggleCompare() {
  if (!store.toggleCompare(property.value)) {
    ui.info('You can compare up to 4 properties at a time')
  }
}

/** The message template lives on the server, so it stays consistent. */
async function openWhatsApp(configuration = '') {
  try {
    const res = await publicApi.whatsappLink(property.value.id, configuration)
    window.open(res.data.url, '_blank', 'noopener')
  } catch (err) {
    ui.error(err.message)
  }
}

function trackClick(eventType) {
  publicApi
    .trackEvent({ event_type: eventType, property_id: property.value.id })
    .catch(() => {})
}

function downloadBrochure() {
  if (!brochure.value) return
  trackClick(EVENT_TYPES.BROCHURE_DOWNLOAD)
  window.open(publicApi.brochureUrl(property.value.id, brochure.value.id), '_blank', 'noopener')
}

function scrollToForm() {
  document.getElementById('inquiry')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

onMounted(load)
watch(() => route.params.slug, load)
</script>

<template>
  <div class="detail">
    <div v-if="loading" class="container section">
      <div class="skeleton" style="height: 340px; border-radius: 16px"></div>
      <div class="skeleton mt-2" style="height: 26px; width: 46%"></div>
      <div class="skeleton mt-1" style="height: 16px; width: 30%"></div>
    </div>

    <div v-else-if="error" class="container section">
      <div class="alert alert-error">{{ error }}</div>
      <RouterLink class="btn btn-primary mt-2" :to="{ name: 'properties' }">
        Browse all properties
      </RouterLink>
    </div>

    <template v-else-if="property">
      <div class="container">
        <nav class="crumbs tiny" aria-label="Breadcrumb">
          <RouterLink :to="{ name: 'home' }">Home</RouterLink>
          <span aria-hidden="true">/</span>
          <RouterLink
            v-if="property.city"
            :to="{ name: 'city', params: { citySlug: property.city.slug } }"
          >
            {{ property.city.name }}
          </RouterLink>
          <span aria-hidden="true">/</span>
          <RouterLink
            v-if="property.locality && property.city"
            :to="{
              name: 'locality',
              params: { citySlug: property.city.slug, localitySlug: property.locality.slug },
            }"
          >
            {{ property.locality.name }}
          </RouterLink>
          <span aria-hidden="true">/</span>
          <span>{{ property.name }}</span>
        </nav>

        <header class="head">
          <div class="head-main">
            <div class="badges">
              <span v-if="property.is_rera_verified" class="badge badge-rera">RERA verified</span>
              <span v-if="property.is_featured" class="badge badge-featured">Featured</span>
              <span class="badge badge-neutral">{{ humanise(property.construction_status) }}</span>
            </div>
            <h1>{{ property.name }}</h1>
            <p class="location">
              {{ property.address || property.locality?.name }}
              <template v-if="property.locality">, {{ property.locality.name }}</template>
              <template v-if="property.city">, {{ property.city.name }}</template>
            </p>
          </div>

          <div class="head-price">
            <template v-if="isRental">
              <div class="price-value">
                {{ formatPrice(property.rental_details?.monthly_rent) }}<span class="per">/month</span>
              </div>
              <div class="tiny muted">
                {{ formatPrice(property.rental_details?.security_deposit) }} deposit
              </div>
            </template>
            <template v-else>
              <div class="price-value">
                {{ formatPriceRange(property.min_price, property.max_price) }}
              </div>
              <div class="tiny muted">
                {{ formatPossession(property.possession_date) }} possession
              </div>
            </template>
          </div>
        </header>

        <div class="layout">
          <div class="main">
            <ImageGallery :media="property.media" :alt="property.name" />

            <div class="quick-facts card">
              <div class="fact">
                <span class="fact-label">Configurations</span>
                <strong>
                  {{ property.configurations?.map((c) => c.name).join(', ') || 'On request' }}
                </strong>
              </div>
              <div class="fact">
                <span class="fact-label">Carpet area</span>
                <strong>{{ formatArea(property.min_area) }} onwards</strong>
              </div>
              <div v-if="isRental" class="fact">
                <span class="fact-label">Furnishing</span>
                <strong>{{ humanise(property.rental_details?.furnishing_type) || 'Unfurnished' }}</strong>
              </div>
              <div v-else class="fact">
                <span class="fact-label">Possession</span>
                <strong>{{ formatPossession(property.possession_date) }}</strong>
              </div>

              <div v-if="isRental" class="fact">
                <span class="fact-label">Available from</span>
                <strong>{{ formatDate(property.rental_details?.available_from) || 'Immediately' }}</strong>
              </div>
              <div v-else class="fact">
                <span class="fact-label">RERA</span>
                <strong>{{ property.rera_details?.registration_number || 'Not available' }}</strong>
              </div>
            </div>

            <section v-if="property.description" class="panel">
              <h2>About {{ property.name }}</h2>
              <p class="description">{{ property.description }}</p>
            </section>

            <!-- A rental has one set of terms; a sale project has a price list. -->
            <section v-if="isRental && property.rental_details" class="panel">
              <div class="panel-head"><h3>Rental terms</h3></div>
              <RentalTerms :rental="property.rental_details" />
            </section>

            <section v-if="!isRental && property.configurations?.length" class="panel">
              <div class="panel-head"><h3>Configurations and pricing</h3></div>
              <div class="table-wrap">
                <table class="table">
                  <thead>
                    <tr>
                      <th>Configuration</th>
                      <th class="num">Carpet area</th>
                      <th class="num">Built-up</th>
                      <th class="num">Price</th>
                      <th class="num">Rate</th>
                      <th>Availability</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="config in property.configurations" :key="config.id">
                      <td>
                        <strong>{{ config.name }}</strong>
                        <div class="tiny muted">
                          {{ config.bedrooms }} bed · {{ config.bathrooms }} bath
                        </div>
                      </td>
                      <td class="num">{{ formatArea(config.carpet_area) }}</td>
                      <td class="num">{{ formatArea(config.built_up_area) }}</td>
                      <td class="num price">{{ formatPriceRange(config.price, config.price) }}</td>
                      <td class="num tiny">{{ pricePerSqft(config.price, config.carpet_area) }}</td>
                      <td>
                        <span
                          class="badge"
                          :class="config.availability === 'AVAILABLE' ? 'badge-success' : 'badge-neutral'"
                        >
                          {{ humanise(config.availability) }}
                        </span>
                      </td>
                      <td>
                        <button class="btn btn-outline btn-sm" @click="openWhatsApp(config.name)">
                          Enquire
                        </button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>

            <section v-if="!isRental" class="panel">
              <div class="panel-head"><h3>RERA information</h3></div>
              <div v-if="property.rera_details?.registration_number" class="rera-grid">
                <div>
                  <span class="fact-label">Registration number</span>
                  <strong class="mono">{{ property.rera_details.registration_number }}</strong>
                </div>
                <div>
                  <span class="fact-label">Authority</span>
                  <strong>{{ property.rera_details.authority || '-' }}</strong>
                </div>
                <div>
                  <span class="fact-label">Registered on</span>
                  <strong>{{ formatDate(property.rera_details.registration_date) }}</strong>
                </div>
                <div>
                  <span class="fact-label">Declared completion</span>
                  <strong>{{ formatDate(property.rera_details.completion_date) }}</strong>
                </div>
                <div>
                  <span class="fact-label">Verification status</span>
                  <span
                    class="badge"
                    :class="`badge-${RERA_STATUS_VARIANT[property.rera_details.verification_status] || 'neutral'}`"
                  >
                    {{ humanise(property.rera_details.verification_status) }}
                  </span>
                </div>
              </div>
              <p v-else class="muted small">
                This project has not supplied RERA registration details yet.
              </p>

              <div class="alert alert-warning mt-2 tiny">
                RERA details are supplied by the builder. Verify the registration number with the
                relevant state RERA authority before making any payment or booking decision.
              </div>
            </section>

            <section v-if="property.amenities?.length" class="panel">
              <div class="panel-head"><h3>Amenities</h3></div>
              <ul class="amenities">
                <li v-for="amenity in property.amenities" :key="amenity.id">
                  <span class="tick" aria-hidden="true">✓</span>{{ amenity.name }}
                </li>
              </ul>
            </section>

            <section v-if="property.latitude && property.longitude" class="panel">
              <div class="panel-head"><h3>Location</h3></div>
              <div class="map">
                <iframe
                  :src="`https://www.openstreetmap.org/export/embed.html?bbox=${property.longitude - 0.012}%2C${property.latitude - 0.008}%2C${property.longitude + 0.012}%2C${property.latitude + 0.008}&layer=mapnik&marker=${property.latitude}%2C${property.longitude}`"
                  :title="`Map showing ${property.name}`"
                  loading="lazy"
                ></iframe>
              </div>
              <p class="tiny muted mt-1 mb-0">
                {{ property.address }}
                <template v-if="property.pincode"> — {{ property.pincode }}</template>
              </p>
            </section>

            <section v-if="property.builder" class="panel">
              <div class="panel-head"><h3>{{ posterHeading }}</h3></div>
              <div class="builder">
                <div class="builder-logo" aria-hidden="true">
                  {{ property.builder.company_name.charAt(0) }}
                </div>
                <div class="grow">
                  <strong>{{ property.builder.company_name }}</strong>
                  <!-- Dealing with an individual owner is a different situation
                       from dealing with a builder, so the page says which. -->
                  <span v-if="verificationBadge" class="badge badge-rera ml-1">
                    {{ verificationBadge }}
                  </span>
                  <p class="small muted mb-1">{{ property.builder.description }}</p>
                  <div class="tiny muted">
                    {{ property.builder.total_properties }} listing(s)
                    <template v-if="property.builder.rera_reg_no">
                      · RERA {{ property.builder.rera_reg_no }}
                    </template>
                    <template v-if="property.posted_by_type === 'ADMIN'">
                      · listed by our team on the owner's behalf
                    </template>
                  </div>
                </div>
                <RouterLink
                  class="btn btn-outline btn-sm"
                  :to="{ name: 'builder', params: { id: property.builder.id } }"
                >
                  View projects
                </RouterLink>
              </div>
            </section>

            <section v-if="similar.length" class="panel">
              <div class="panel-head"><h3>Similar projects nearby</h3></div>
              <PropertyGrid :properties="similar" compact />
            </section>
          </div>

          <aside class="sidebar">
            <div class="contact-card panel" id="inquiry">
              <div class="contact-price">
                <span class="tiny muted">{{ isRental ? 'Monthly rent' : 'Starting from' }}</span>
                <div class="price-value">
                  {{
                    isRental
                      ? formatPrice(property.rental_details?.monthly_rent)
                      : formatPriceRange(property.min_price, 0)
                  }}
                </div>
                <span class="tiny muted">
                  <template v-if="isRental">
                    {{ formatPrice(property.rental_details?.security_deposit) }} deposit
                  </template>
                  <template v-else>
                    {{ property.configurations?.length || 0 }} configuration(s) available
                  </template>
                </span>
              </div>

              <div class="contact-actions">
                <button class="btn btn-whatsapp btn-block" @click="openWhatsApp()">
                  Chat on WhatsApp
                </button>
                <!-- A viewing is how renting actually converts, so it is a
                     primary action rather than buried in the enquiry form. -->
                <button class="btn btn-accent btn-block" @click="visitOpen = true">
                  Schedule a visit
                </button>
                <button
                  v-if="brochure"
                  class="btn btn-outline btn-block"
                  @click="downloadBrochure"
                >
                  Download brochure
                </button>
                <div class="row">
                  <button
                    class="btn btn-outline grow"
                    :class="{ 'btn-primary': saved }"
                    @click="toggleSave"
                  >
                    {{ saved ? 'Saved' : 'Save' }}
                  </button>
                  <button
                    class="btn btn-outline grow"
                    :class="{ 'btn-primary': comparing }"
                    @click="toggleCompare"
                  >
                    {{ comparing ? 'Added' : 'Compare' }}
                  </button>
                </div>
              </div>

              <hr class="divider" />

              <InquiryForm :property="property" />

              <p class="tiny muted mt-2 mb-0">
                {{ formatNumber(property.view_count) }} people viewed this project.
              </p>

              <!-- Deliberately understated and last: the people who notice a
                   stale or dishonest listing are the ones already reading it,
                   and they will not go looking for a way to say so. -->
              <button class="report-link tiny" @click="reportOpen = true">
                Something wrong with this listing? Report it
              </button>
            </div>
          </aside>
        </div>
      </div>

      <!-- On phones the sticky card is off-screen, so the primary actions follow
           the visitor down the page instead. -->
      <div class="mobile-bar only-mobile">
        <button class="btn btn-outline btn-sm" @click="toggleSave">
          {{ saved ? '♥' : '♡' }}
        </button>
        <button class="btn btn-accent grow" @click="visitOpen = true">Visit</button>
        <button class="btn btn-primary grow" @click="scrollToForm">Contact</button>
        <button class="btn btn-whatsapp grow" @click="openWhatsApp()">WhatsApp</button>
      </div>

      <VisitRequestModal
        v-if="visitOpen"
        :property="property"
        @close="visitOpen = false"
      />

      <ReportListingModal
        v-if="reportOpen"
        :property="property"
        @close="reportOpen = false"
      />
    </template>
  </div>
</template>

<style scoped>
.detail { padding: 20px 0 60px; }

.crumbs { display: flex; gap: 7px; color: var(--ink-500); margin-bottom: 14px; flex-wrap: wrap; }
.crumbs a { color: var(--ink-600); }

.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.head h1 { font-size: 1.75rem; margin-bottom: 6px; }
.badges { display: flex; gap: 7px; margin-bottom: 10px; flex-wrap: wrap; }
.location { color: var(--ink-600); margin: 0; }

.head-price { text-align: right; }
.price-value { font-size: 1.5rem; font-weight: 800; color: var(--ink-900); letter-spacing: -0.02em; }
.price-value .per { font-size: 0.92rem; font-weight: 600; color: var(--ink-500); }

.layout { display: grid; grid-template-columns: 1fr 366px; gap: 24px; align-items: start; }
.main { display: grid; gap: 18px; min-width: 0; }

.quick-facts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
  padding: 18px 20px;
}
.fact { display: flex; flex-direction: column; gap: 3px; }
.fact-label {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--ink-500);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.fact strong { color: var(--ink-900); font-size: 0.94rem; }

.description { white-space: pre-line; color: var(--ink-700); margin-bottom: 0; }

.rera-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 18px;
}
.rera-grid > div { display: flex; flex-direction: column; gap: 4px; align-items: flex-start; }
.mono { font-family: ui-monospace, 'SFMono-Regular', Menlo, monospace; font-size: 0.9rem; }

.amenities {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 10px;
}
.amenities li { display: flex; align-items: center; gap: 9px; font-size: 0.9rem; }
.tick {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--brand-50);
  color: var(--brand-700);
  font-size: 0.7rem;
  font-weight: 800;
  flex-shrink: 0;
}

.map { border-radius: var(--radius); overflow: hidden; border: 1px solid var(--ink-200); }
.map iframe { width: 100%; height: 320px; border: 0; display: block; }

.builder { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; }
.builder-logo {
  width: 52px;
  height: 52px;
  border-radius: var(--radius);
  display: grid;
  place-items: center;
  background: var(--brand-700);
  color: var(--white);
  font-size: 1.3rem;
  font-weight: 800;
  font-family: var(--font-display);
  flex-shrink: 0;
}

.sidebar { position: sticky; top: calc(var(--header-height) + 16px); }
.contact-price { text-align: center; margin-bottom: 16px; }
.contact-actions { display: grid; gap: 8px; }

.mobile-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 80;
  display: flex;
  gap: 8px;
  padding: 10px 14px;
  background: var(--white);
  border-top: 1px solid var(--ink-200);
  box-shadow: 0 -4px 18px rgba(15, 23, 32, 0.1);
}

@media (max-width: 1040px) {
  .layout { grid-template-columns: 1fr; }
  .sidebar { position: static; }
  .head-price { text-align: left; }
}

.report-link {
  display: block;
  margin-top: 10px;
  border: 0;
  background: none;
  padding: 0;
  color: var(--ink-500);
  text-decoration: underline;
  cursor: pointer;
}
.report-link:hover { color: var(--ink-700); }
</style>
