<script setup>
import { onMounted, reactive, ref } from 'vue'
import { adminApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { formatPriceRange } from '@/utils/format'

const ui = useUiStore()

const tab = ref('banners')
const banners = ref([])
const featured = ref([])
const pages = ref([])
const loading = ref(true)
const uploading = ref(false)

const bannerForm = reactive({
  id: null,
  title: '',
  subtitle: '',
  image_url: '',
  link_url: '',
  cta_text: '',
  cta_link: '',
  position: 0,
  is_active: true,
})

const pageForm = reactive({
  title: '',
  slug: '',
  content: '',
  meta_title: '',
  meta_description: '',
  is_published: true,
})

const featuredForm = reactive({ property_id: '', position: 0 })

async function loadAll() {
  loading.value = true
  try {
    const [b, f, p] = await Promise.all([
      adminApi.banners().catch(() => ({ data: [] })),
      adminApi.featured().catch(() => ({ data: [] })),
      adminApi.pages().catch(() => ({ data: [] })),
    ])
    banners.value = b.data || []
    featured.value = f.data || []
    pages.value = p.data || []
  } finally {
    loading.value = false
  }
}

function editBanner(banner) {
  Object.assign(bannerForm, banner)
  tab.value = 'banners'
}

function resetBanner() {
  Object.assign(bannerForm, {
    id: null, title: '', subtitle: '', image_url: '', link_url: '',
    cta_text: '', cta_link: '', position: 0, is_active: true,
  })
}

async function saveBanner() {
  try {
    const payload = { ...bannerForm, position: Number(bannerForm.position) || 0 }
    if (bannerForm.id) await adminApi.updateBanner(bannerForm.id, payload)
    else await adminApi.createBanner(payload)
    ui.success('Banner saved')
    resetBanner()
    await loadAll()
  } catch (err) {
    ui.error(err.message)
  }
}

async function deleteBanner(banner) {
  const confirmed = await ui.confirm({ title: `Delete "${banner.title}"?`, danger: true, confirmLabel: 'Delete' })
  if (!confirmed) return
  await adminApi.deleteBanner(banner.id)
  ui.success('Banner deleted')
  await loadAll()
}

async function uploadBannerImage(event) {
  const file = event.target.files?.[0]
  if (!file) return
  uploading.value = true
  try {
    const res = await adminApi.uploadAsset(file, 'assets/banners')
    bannerForm.image_url = res.data.url
    ui.success('Image uploaded')
  } catch (err) {
    ui.error(err.message)
  } finally {
    uploading.value = false
    event.target.value = ''
  }
}

async function addFeatured() {
  try {
    await adminApi.addFeatured({
      property_id: Number(featuredForm.property_id),
      position: Number(featuredForm.position) || 0,
    })
    ui.success('Property featured')
    featuredForm.property_id = ''
    await loadAll()
  } catch (err) {
    ui.error(err.message)
  }
}

async function removeFeatured(item) {
  const confirmed = await ui.confirm({ title: 'Remove from featured?', confirmLabel: 'Remove' })
  if (!confirmed) return
  await adminApi.removeFeatured(item.id)
  await loadAll()
}

function editPage(page) {
  Object.assign(pageForm, page)
  tab.value = 'pages'
}

function resetPage() {
  Object.assign(pageForm, {
    title: '', slug: '', content: '', meta_title: '', meta_description: '', is_published: true,
  })
}

async function savePage() {
  try {
    await adminApi.savePage({ ...pageForm })
    ui.success('Page saved')
    resetPage()
    await loadAll()
  } catch (err) {
    ui.error(err.message)
  }
}

async function deletePage(page) {
  const confirmed = await ui.confirm({ title: `Delete "${page.title}"?`, danger: true, confirmLabel: 'Delete' })
  if (!confirmed) return
  await adminApi.deletePage(page.id)
  ui.success('Page deleted')
  await loadAll()
}

onMounted(loadAll)
</script>

<template>
  <div>
    <div class="section-head">
      <div>
        <h2>Homepage content</h2>
        <p>Banners, featured projects and informational pages</p>
      </div>
    </div>

    <nav class="tabs">
      <button class="tab" :class="{ active: tab === 'banners' }" @click="tab = 'banners'">
        Banners
      </button>
      <button class="tab" :class="{ active: tab === 'featured' }" @click="tab = 'featured'">
        Featured projects
      </button>
      <button class="tab" :class="{ active: tab === 'pages' }" @click="tab = 'pages'">
        Static pages
      </button>
    </nav>

    <!-- Banners ----------------------------------------------------------- -->
    <div v-if="tab === 'banners'" class="layout">
      <section class="panel">
        <div class="panel-head">
          <h3>{{ bannerForm.id ? 'Edit banner' : 'New banner' }}</h3>
          <button v-if="bannerForm.id" class="btn btn-ghost btn-sm" @click="resetBanner">New</button>
        </div>

        <form @submit.prevent="saveBanner">
          <label class="field">
            <span class="label">Title</span>
            <input v-model="bannerForm.title" class="input" required />
          </label>
          <label class="field">
            <span class="label">Subtitle</span>
            <input v-model="bannerForm.subtitle" class="input" />
          </label>
          <label class="field">
            <span class="label">Image URL</span>
            <input v-model="bannerForm.image_url" class="input" required />
            <label class="btn btn-outline btn-sm mt-1">
              {{ uploading ? 'Uploading…' : 'Upload image' }}
              <input type="file" accept="image/*" hidden @change="uploadBannerImage" />
            </label>
          </label>
          <div class="form-grid">
            <label class="field">
              <span class="label">Button text</span>
              <input v-model="bannerForm.cta_text" class="input" />
            </label>
            <label class="field">
              <span class="label">Button link</span>
              <input v-model="bannerForm.cta_link" class="input" placeholder="/properties?city=pune" />
            </label>
            <label class="field">
              <span class="label">Position</span>
              <input v-model="bannerForm.position" class="input" type="number" />
            </label>
          </div>
          <label class="check">
            <input v-model="bannerForm.is_active" type="checkbox" />
            <span>Active on the homepage</span>
          </label>
          <button class="btn btn-primary btn-block" type="submit">Save banner</button>
        </form>
      </section>

      <section class="panel">
        <div class="panel-head"><h3>Current banners</h3></div>
        <ul class="content-list">
          <li v-for="banner in banners" :key="banner.id">
            <img v-if="banner.image_url" :src="banner.image_url" :alt="banner.title" class="thumb" />
            <div class="grow">
              <strong>{{ banner.title }}</strong>
              <div class="tiny muted">{{ banner.subtitle }}</div>
              <span class="badge" :class="banner.is_active ? 'badge-success' : 'badge-neutral'">
                {{ banner.is_active ? 'Active' : 'Hidden' }}
              </span>
            </div>
            <div class="row">
              <button class="btn btn-ghost btn-sm" @click="editBanner(banner)">Edit</button>
              <button class="btn btn-ghost btn-sm danger" @click="deleteBanner(banner)">Delete</button>
            </div>
          </li>
        </ul>
      </section>
    </div>

    <!-- Featured ---------------------------------------------------------- -->
    <div v-else-if="tab === 'featured'" class="layout">
      <section class="panel">
        <div class="panel-head"><h3>Feature a project</h3></div>
        <form @submit.prevent="addFeatured">
          <label class="field">
            <span class="label">Property ID</span>
            <input v-model="featuredForm.property_id" class="input" type="number" required />
            <span class="field-hint">
              Find the id in the property list, or use the Feature button there.
            </span>
          </label>
          <label class="field">
            <span class="label">Position</span>
            <input v-model="featuredForm.position" class="input" type="number" />
          </label>
          <button class="btn btn-primary btn-block" type="submit">Add to featured</button>
        </form>
      </section>

      <section class="panel">
        <div class="panel-head"><h3>Featured projects</h3></div>
        <ul class="content-list">
          <li v-for="item in featured" :key="item.id">
            <img
              v-if="item.property?.cover_image"
              :src="item.property.cover_image"
              :alt="item.property.name"
              class="thumb"
            />
            <div class="grow">
              <strong>{{ item.property?.name || `Property #${item.id}` }}</strong>
              <div class="tiny muted">
                {{ item.property?.locality }}, {{ item.property?.city }} ·
                {{ formatPriceRange(item.property?.min_price, item.property?.max_price) }}
              </div>
            </div>
            <button class="btn btn-ghost btn-sm danger" @click="removeFeatured(item)">Remove</button>
          </li>
        </ul>
        <p v-if="!featured.length" class="muted small mb-0">No featured projects yet.</p>
      </section>
    </div>

    <!-- Pages ------------------------------------------------------------- -->
    <div v-else class="layout">
      <section class="panel">
        <div class="panel-head">
          <h3>{{ pageForm.id ? 'Edit page' : 'New page' }}</h3>
          <button class="btn btn-ghost btn-sm" @click="resetPage">New</button>
        </div>

        <form @submit.prevent="savePage">
          <label class="field">
            <span class="label">Title</span>
            <input v-model="pageForm.title" class="input" required />
          </label>
          <label class="field">
            <span class="label">Slug</span>
            <input v-model="pageForm.slug" class="input" required placeholder="about-us" />
            <span class="field-hint">The page is served at /page/&lt;slug&gt;.</span>
          </label>
          <label class="field">
            <span class="label">Content (HTML)</span>
            <textarea v-model="pageForm.content" class="textarea" rows="10" required></textarea>
          </label>
          <label class="field">
            <span class="label">Meta title</span>
            <input v-model="pageForm.meta_title" class="input" />
          </label>
          <label class="field">
            <span class="label">Meta description</span>
            <textarea v-model="pageForm.meta_description" class="textarea" rows="2"></textarea>
          </label>
          <label class="check">
            <input v-model="pageForm.is_published" type="checkbox" />
            <span>Published</span>
          </label>
          <button class="btn btn-primary btn-block" type="submit">Save page</button>
        </form>
      </section>

      <section class="panel">
        <div class="panel-head"><h3>Pages</h3></div>
        <ul class="content-list">
          <li v-for="page in pages" :key="page.id">
            <div class="grow">
              <strong>{{ page.title }}</strong>
              <div class="tiny muted">/page/{{ page.slug }}</div>
            </div>
            <span class="badge" :class="page.is_published ? 'badge-success' : 'badge-neutral'">
              {{ page.is_published ? 'Published' : 'Draft' }}
            </span>
            <div class="row">
              <button class="btn btn-ghost btn-sm" @click="editPage(page)">Edit</button>
              <button class="btn btn-ghost btn-sm danger" @click="deletePage(page)">Delete</button>
            </div>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  padding: 4px;
  background: var(--ink-100);
  border-radius: var(--radius);
  width: fit-content;
}
.tab {
  padding: 8px 16px;
  border: 0;
  border-radius: var(--radius-sm);
  background: none;
  font-size: 0.86rem;
  font-weight: 600;
  color: var(--ink-600);
  cursor: pointer;
}
.tab.active { background: var(--white); color: var(--brand-800); box-shadow: var(--shadow-sm); }

.layout { display: grid; grid-template-columns: 380px 1fr; gap: 20px; align-items: start; }

.content-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 12px; }
.content-list li {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  flex-wrap: wrap;
}
.thumb { width: 76px; height: 50px; object-fit: cover; border-radius: var(--radius-sm); }
.danger { color: var(--danger-600); }

@media (max-width: 960px) {
  .layout { grid-template-columns: 1fr; }
}
</style>
