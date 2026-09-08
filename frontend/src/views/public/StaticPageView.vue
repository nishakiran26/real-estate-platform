<script setup>
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { publicApi } from '@/services/api'
import { useSeo } from '@/composables/useSeo'

const route = useRoute()
const { setMeta } = useSeo()

const page = ref(null)
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await publicApi.page(route.params.slug)
    page.value = res.data
    setMeta({
      title: page.value.meta_title || page.value.title,
      description: page.value.meta_description || '',
    })
  } catch (err) {
    error.value = err.message
    page.value = null
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => route.params.slug, load)
</script>

<template>
  <div class="container section">
    <div v-if="loading" class="skeleton" style="height: 280px"></div>
    <div v-else-if="error" class="alert alert-error">{{ error }}</div>

    <article v-else-if="page" class="page panel">
      <h1>{{ page.title }}</h1>
      <!-- Content is authored by administrators in the CMS. -->
      <div class="content" v-html="page.content"></div>
    </article>
  </div>
</template>

<style scoped>
.page { max-width: 820px; margin: 0 auto; padding: 32px; }
.page h1 { margin-bottom: 20px; }
.content { color: var(--ink-700); line-height: 1.7; }
.content :deep(h2) { margin-top: 26px; font-size: 1.2rem; }
.content :deep(ul) { padding-left: 20px; }
.content :deep(a) { text-decoration: underline; }
</style>
