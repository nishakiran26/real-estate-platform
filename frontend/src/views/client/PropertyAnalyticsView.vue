<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { clientApi } from '@/services/api'
import StatTile from '@/components/common/StatTile.vue'

const route = useRoute()

const stats = ref(null)
const property = ref(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    const [analytics, detail] = await Promise.all([
      clientApi.propertyAnalytics(route.params.id),
      clientApi.property(route.params.id),
    ])
    stats.value = analytics.data
    property.value = detail.data
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-else-if="loading" class="skeleton" style="height: 200px"></div>

    <template v-else>
      <div class="section-head">
        <div>
          <h2>{{ property?.name }}</h2>
          <p>{{ property?.locality?.name }}, {{ property?.city?.name }}</p>
        </div>
        <div class="row">
          <RouterLink
            class="btn btn-outline btn-sm"
            :to="{ name: 'client-property-edit', params: { id: route.params.id } }"
          >
            Edit listing
          </RouterLink>
          <RouterLink
            v-if="property?.slug"
            class="btn btn-ghost btn-sm"
            :to="{ name: 'property-detail', params: { slug: property.slug } }"
          >
            View public page
          </RouterLink>
        </div>
      </div>

      <div class="grid grid-4">
        <StatTile label="Views" :value="stats?.views || 0" />
        <StatTile label="Unique visitors" :value="stats?.unique_visitors || 0" />
        <StatTile label="Inquiries" :value="stats?.inquiries || 0" />
        <StatTile
          label="Conversion"
          :value="(stats?.conversion_rate || 0).toFixed(2)"
          suffix="%"
          meta="Inquiries per view"
        />
      </div>

      <div class="grid grid-4 mt-2">
        <StatTile label="WhatsApp clicks" :value="stats?.whatsapp_clicks || 0" />
        <StatTile label="Phone clicks" :value="stats?.phone_clicks || 0" />
        <StatTile label="Brochure downloads" :value="stats?.brochure_downloads || 0" />
        <StatTile label="Saved by buyers" :value="stats?.favorites || 0" />
      </div>

      <section class="panel mt-3">
        <div class="panel-head"><h3>What these numbers mean</h3></div>
        <ul class="notes">
          <li>
            <strong>Views</strong> counts opens of the listing page. Repeat opens by the same
            visitor within 30 minutes are counted once.
          </li>
          <li>
            <strong>Unique visitors</strong> counts distinct people, whether or not they are logged
            in.
          </li>
          <li>
            <strong>Conversion</strong> is inquiries divided by views. WhatsApp and phone clicks are
            tracked separately because they often replace a form submission.
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>

<style scoped>
.notes { margin: 0; padding-left: 20px; display: grid; gap: 9px; color: var(--ink-600); font-size: 0.9rem; }
</style>
