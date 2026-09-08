<script setup>
import { onMounted, ref } from 'vue'
import { publicApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'
import { useSeo } from '@/composables/useSeo'
import HeroSearch from '@/components/property/HeroSearch.vue'
import PropertyGrid from '@/components/property/PropertyGrid.vue'

const store = usePropertyStore()
const { setMeta } = useSeo()

const home = ref(null)
const loading = ref(true)
const recentlyViewed = ref([])

onMounted(async () => {
  setMeta({
    title: 'RERA Registered Flats and New Projects in Pune',
    description:
      'Discover RERA-registered flats, apartments and new residential projects in Pune, Mumbai and Bengaluru. Compare configurations, carpet area, price and possession dates.',
  })

  await store.loadReference()
  try {
    const res = await publicApi.homepage()
    home.value = res.data
  } finally {
    loading.value = false
  }

  // Works for anonymous visitors too: the API keys this on the session cookie.
  try {
    const res = await publicApi.recentlyViewed(4)
    recentlyViewed.value = res.data || []
  } catch {
    recentlyViewed.value = []
  }
})
</script>

<template>
  <div>
    <section class="hero">
      <div class="container hero-inner">
        <p class="eyebrow">RERA-registered projects only</p>
        <h1 class="display">
          Find a home you can verify,<br />
          not just a brochure.
        </h1>
        <p class="lede">
          Every project on this platform carries its RERA registration details and is reviewed
          before it is published. Compare carpet area, price and possession across Pune.
        </p>

        <HeroSearch class="hero-search-wrap" />

        <div class="quick-links">
          <span class="tiny">Popular:</span>
          <RouterLink
            v-for="locality in store.localities.filter((l) => l.is_popular).slice(0, 5)"
            :key="locality.id"
            :to="{ name: 'properties', query: { locality: locality.slug } }"
            class="quick-chip"
          >
            {{ locality.name }}
          </RouterLink>
        </div>
      </div>
    </section>

    <section v-if="home?.banners?.length" class="section-tight">
      <div class="container">
        <div class="banner-strip">
          <a
            v-for="banner in home.banners"
            :key="banner.id"
            class="promo-banner"
            :href="banner.cta_link || banner.link_url || '#'"
          >
            <img :src="banner.image_url" :alt="banner.title" loading="lazy" />
            <div class="promo-body">
              <h3>{{ banner.title }}</h3>
              <p class="small">{{ banner.subtitle }}</p>
              <span v-if="banner.cta_text" class="promo-cta">{{ banner.cta_text }} →</span>
            </div>
          </a>
        </div>
      </div>
    </section>

    <section class="section">
      <div class="container">
        <div class="section-head">
          <div>
            <h2>Featured projects</h2>
            <p>Handpicked RERA-registered developments</p>
          </div>
          <RouterLink class="btn btn-outline btn-sm" :to="{ name: 'properties' }">
            View all
          </RouterLink>
        </div>
        <PropertyGrid :properties="home?.featured || []" :loading="loading" :skeleton-count="3" />
      </div>
    </section>

    <section v-if="!loading && home?.new_launches?.length" class="section band">
      <div class="container">
        <div class="section-head">
          <div>
            <h2>Newly launched</h2>
            <p>Projects that opened for booking recently</p>
          </div>
          <RouterLink
            class="btn btn-outline btn-sm"
            :to="{ name: 'properties', query: { sort: 'newest' } }"
          >
            See all new launches
          </RouterLink>
        </div>
        <PropertyGrid :properties="home.new_launches" :skeleton-count="3" />
      </div>
    </section>

    <section v-if="!loading && home?.ready_to_move?.length" class="section">
      <div class="container">
        <div class="section-head">
          <div>
            <h2>Ready to move</h2>
            <p>Homes with possession available now</p>
          </div>
          <RouterLink
            class="btn btn-outline btn-sm"
            :to="{ name: 'properties', query: { status: 'READY_TO_MOVE' } }"
          >
            View all
          </RouterLink>
        </div>
        <PropertyGrid :properties="home.ready_to_move" :skeleton-count="3" />
      </div>
    </section>

    <!-- Rentals stand apart from every sale strip above. A monthly rent shown
         next to sale prices reads as a sale price, so the heading, the wording
         and the "View all" link all say rent before a single card is seen. -->
    <section v-if="!loading && home?.rentals?.length" class="section band">
      <div class="container">
        <div class="section-head">
          <div>
            <h2>Homes to rent</h2>
            <p>Available now, with monthly rent and deposit</p>
          </div>
          <RouterLink class="btn btn-outline btn-sm" :to="{ name: 'rent' }">
            View all rentals
          </RouterLink>
        </div>
        <PropertyGrid :properties="home.rentals" :skeleton-count="3" />
      </div>
    </section>

    <section v-if="store.cities.length" class="section">
      <div class="container">
        <div class="section-head">
          <div>
            <h2>Browse by city</h2>
            <p>Where buyers are searching right now</p>
          </div>
        </div>
        <div class="grid grid-4">
          <RouterLink
            v-for="city in store.cities"
            :key="city.id"
            class="city-tile card"
            :to="{ name: 'city', params: { citySlug: city.slug } }"
          >
            <strong>{{ city.name }}</strong>
            <span class="tiny muted">{{ city.state }}</span>
          </RouterLink>
        </div>
      </div>
    </section>

    <section v-if="!loading && home?.upcoming?.length" class="section band">
      <div class="container">
        <div class="section-head">
          <div>
            <h2>Upcoming projects</h2>
            <p>Register early interest before launch</p>
          </div>
        </div>
        <PropertyGrid :properties="home.upcoming" :skeleton-count="3" />
      </div>
    </section>

    <section v-if="recentlyViewed.length" class="section">
      <div class="container">
        <div class="section-head">
          <div>
            <h2>Pick up where you left off</h2>
            <p>Properties you viewed recently</p>
          </div>
        </div>
        <PropertyGrid :properties="recentlyViewed" compact />
      </div>
    </section>

    <section class="section">
      <div class="container">
        <div class="trust card">
          <div>
            <h3>Why the RERA number matters</h3>
            <p class="small muted mb-0">
              A registered project has filed its approvals, timelines and layout with the state
              authority. We display the registration number, authority and completion date on every
              listing so you can check it yourself before you commit.
            </p>
          </div>
          <RouterLink
            class="btn btn-primary"
            :to="{ name: 'static-page', params: { slug: 'rera-disclaimer' } }"
          >
            Read the disclaimer
          </RouterLink>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.hero {
  background:
    linear-gradient(180deg, rgba(8, 38, 42, 0.9) 0%, rgba(20, 81, 79, 0.86) 100%),
    url('https://images.unsplash.com/photo-1486406146926-c627a92ad1ab?w=1800&q=80&auto=format&fit=crop')
      center / cover no-repeat;
  color: var(--white);
  padding: 62px 0 48px;
}

.hero-inner { max-width: 900px; }

.eyebrow {
  display: inline-block;
  margin-bottom: 12px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.16);
  font-size: 0.76rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.hero h1 { color: var(--white); margin-bottom: 12px; }

.lede {
  max-width: 620px;
  color: rgba(255, 255, 255, 0.82);
  font-size: 1rem;
  margin-bottom: 26px;
}

.hero-search-wrap { margin-bottom: 16px; }

.quick-links { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; color: rgba(255, 255, 255, 0.7); }
.quick-chip {
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.12);
  color: var(--white);
  font-size: 0.8rem;
  font-weight: 600;
}
.quick-chip:hover { background: rgba(255, 255, 255, 0.22); color: var(--white); }

.band { background: var(--white); border-block: 1px solid var(--ink-200); }

.banner-strip { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px; }

.promo-banner {
  position: relative;
  border-radius: var(--radius-lg);
  overflow: hidden;
  min-height: 150px;
  display: flex;
  align-items: flex-end;
  color: var(--white);
}
.promo-banner img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
.promo-banner::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(8, 38, 42, 0.86) 25%, rgba(8, 38, 42, 0.3) 100%);
}
.promo-body { position: relative; z-index: 1; padding: 18px 20px; }
.promo-body h3 { color: var(--white); margin-bottom: 4px; font-size: 1.05rem; }
.promo-body p { color: rgba(255, 255, 255, 0.8); margin-bottom: 6px; }
.promo-cta { font-size: 0.84rem; font-weight: 700; }

.city-tile {
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  transition: all var(--transition);
}
.city-tile:hover { border-color: var(--brand-500); transform: translateY(-2px); }
.city-tile strong { color: var(--ink-900); }

.trust {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 22px;
  padding: 24px;
  background: var(--brand-50);
  border-color: var(--brand-100);
  flex-wrap: wrap;
}
.trust h3 { margin-bottom: 6px; }
.trust p { max-width: 640px; }

@media (max-width: 640px) {
  .hero { padding: 36px 0 30px; }
}
</style>
