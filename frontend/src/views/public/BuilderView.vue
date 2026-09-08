<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { publicApi } from '@/services/api'
import { useSeo } from '@/composables/useSeo'
import PropertyGrid from '@/components/property/PropertyGrid.vue'

const route = useRoute()
const { setMeta } = useSeo()

const builder = ref(null)
const projects = ref([])
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    const res = await publicApi.builder(route.params.id)
    builder.value = res.data
    setMeta({
      title: `${builder.value.company_name} projects`,
      description:
        builder.value.description ||
        `RERA-registered projects by ${builder.value.company_name}.`,
    })

    // The public search is the single source of a builder's live listings.
    const list = await publicApi.properties({
      builder: builder.value.company_name,
      per_page: 24,
    })
    projects.value = list.data || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="container section">
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <template v-else-if="builder">
      <header class="builder-head panel">
        <div class="logo" aria-hidden="true">{{ builder.company_name.charAt(0) }}</div>
        <div class="grow">
          <h1>{{ builder.company_name }}</h1>
          <p class="muted">{{ builder.description }}</p>
          <div class="row-wrap tiny muted">
            <span>{{ builder.total_properties }} project(s) listed</span>
            <span v-if="builder.rera_reg_no">RERA {{ builder.rera_reg_no }}</span>
            <a v-if="builder.website" :href="builder.website" target="_blank" rel="noopener">
              Visit website
            </a>
          </div>
        </div>
      </header>

      <div class="section-head mt-3">
        <div>
          <h2>Projects by {{ builder.company_name }}</h2>
          <p>Live RERA-registered listings</p>
        </div>
      </div>

      <PropertyGrid
        :properties="projects"
        :loading="loading"
        empty-title="No published projects"
        empty-message="This builder has no live listings at the moment."
      />
    </template>
  </div>
</template>

<style scoped>
.builder-head { display: flex; align-items: center; gap: 20px; flex-wrap: wrap; }
.builder-head h1 { font-size: 1.5rem; margin-bottom: 6px; }

.logo {
  width: 70px;
  height: 70px;
  border-radius: var(--radius-lg);
  display: grid;
  place-items: center;
  background: var(--brand-700);
  color: var(--white);
  font-size: 1.8rem;
  font-weight: 800;
  font-family: var(--font-display);
  flex-shrink: 0;
}
</style>
