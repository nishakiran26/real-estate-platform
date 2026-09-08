<script setup>
import { onMounted, ref } from 'vue'
import { publicApi } from '@/services/api'
import { usePropertyStore } from '@/stores/property'

const properties = usePropertyStore()
const pages = ref([])
const year = new Date().getFullYear()

onMounted(async () => {
  // Footer links come from the CMS, so an administrator can publish a new page
  // without a frontend deploy.
  try {
    const res = await publicApi.pages()
    pages.value = res.data || []
  } catch {
    pages.value = []
  }
})
</script>

<template>
  <footer class="site-footer">
    <div class="container">
      <div class="footer-grid">
        <div>
          <div class="brand">
            <span class="brand-mark" aria-hidden="true">R</span>
            <strong>RERA Properties</strong>
          </div>
          <p class="small muted mt-1">
            Discover RERA-registered flats, apartments and new residential projects across India.
            Every listing is reviewed before it is published.
          </p>
        </div>

        <nav aria-labelledby="footer-explore">
          <h4 id="footer-explore">Explore</h4>
          <RouterLink :to="{ name: 'properties' }">All properties</RouterLink>
          <RouterLink :to="{ name: 'properties', query: { status: 'READY_TO_MOVE' } }">
            Ready to move
          </RouterLink>
          <RouterLink :to="{ name: 'properties', query: { status: 'UNDER_CONSTRUCTION' } }">
            Under construction
          </RouterLink>
          <RouterLink :to="{ name: 'properties', query: { rera_verified: 'true' } }">
            RERA verified projects
          </RouterLink>
          <RouterLink :to="{ name: 'compare' }">Compare properties</RouterLink>
        </nav>

        <nav aria-labelledby="footer-cities">
          <h4 id="footer-cities">Popular cities</h4>
          <RouterLink
            v-for="city in properties.cities.slice(0, 5)"
            :key="city.id"
            :to="{ name: 'city', params: { citySlug: city.slug } }"
          >
            Property in {{ city.name }}
          </RouterLink>
        </nav>

        <nav aria-labelledby="footer-company">
          <h4 id="footer-company">Company</h4>
          <RouterLink
            v-for="page in pages"
            :key="page.id"
            :to="{ name: 'static-page', params: { slug: page.slug } }"
          >
            {{ page.title }}
          </RouterLink>
          <RouterLink :to="{ name: 'register' }">List your project</RouterLink>
        </nav>
      </div>

      <div class="disclaimer">
        <strong>RERA disclaimer.</strong>
        Project information, including RERA registration details, is provided by the builder. While
        registration numbers are reviewed before a listing is published, buyers must independently
        verify every project and its regulatory approvals with the relevant RERA authority before
        making a purchase decision. This platform is not a party to any transaction between a buyer
        and a builder.
      </div>

      <div class="footer-base">
        <span class="tiny muted">© {{ year }} RERA Properties. All rights reserved.</span>
        <RouterLink class="tiny" :to="{ name: 'static-page', params: { slug: 'privacy-policy' } }">
          Privacy
        </RouterLink>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.site-footer {
  margin-top: 48px;
  background: var(--brand-900);
  color: var(--ink-300);
  padding: 44px 0 24px;
}

.footer-grid {
  display: grid;
  grid-template-columns: 1.6fr repeat(3, 1fr);
  gap: 32px;
}

.brand { display: flex; align-items: center; gap: 10px; color: var(--white); }
.brand-mark {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 9px;
  background: var(--brand-500);
  color: var(--white);
  font-weight: 800;
  font-family: var(--font-display);
}

h4 {
  color: var(--white);
  font-size: 0.82rem;
  text-transform: uppercase;
  letter-spacing: 0.07em;
  margin-bottom: 14px;
}

nav { display: flex; flex-direction: column; gap: 9px; }
nav a { color: var(--ink-300); font-size: 0.87rem; }
nav a:hover { color: var(--white); }

.disclaimer {
  margin-top: 36px;
  padding: 16px 18px;
  border-radius: var(--radius);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 0.8rem;
  line-height: 1.6;
  color: var(--ink-300);
}
.disclaimer strong { color: var(--white); }

.footer-base {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.09);
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.footer-base a { color: var(--ink-300); }
.footer-base a:hover { color: var(--white); }
.muted { color: var(--ink-400); }

@media (max-width: 860px) {
  .footer-grid { grid-template-columns: repeat(2, 1fr); gap: 26px; }
}
@media (max-width: 520px) {
  .footer-grid { grid-template-columns: 1fr; }
}
</style>
