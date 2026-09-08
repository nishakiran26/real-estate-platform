<script setup>
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import SiteHeader from '@/components/layout/SiteHeader.vue'
import SiteFooter from '@/components/layout/SiteFooter.vue'
import CompareTray from '@/components/property/CompareTray.vue'
import { usePropertyStore } from '@/stores/property'
import { useAuthStore } from '@/stores/auth'

const properties = usePropertyStore()
const auth = useAuthStore()

onMounted(() => {
  // Filter reference data is shared by every public page, so it is fetched once
  // for the whole session rather than per view.
  properties.loadReference()
  if (auth.isAuthenticated) properties.loadFavorites()
})
</script>

<template>
  <div class="shell">
    <SiteHeader />
    <main class="main">
      <RouterView />
    </main>
    <SiteFooter />
    <CompareTray />
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.main { flex: 1; }
</style>
