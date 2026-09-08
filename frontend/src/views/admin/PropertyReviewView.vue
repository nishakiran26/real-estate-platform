<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { adminApi } from '@/services/api'
import { useUiStore } from '@/stores/ui'
import { RERA_STATUS_VARIANT, STATUS_VARIANT } from '@/utils/constants'
import {
  formatArea, formatDate, formatDateTime, formatNumber, formatPriceRange, humanise,
} from '@/utils/format'
import ImageGallery from '@/components/property/ImageGallery.vue'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()

const property = ref(null)
const history = ref([])
const loading = ref(true)
const error = ref('')
const working = ref(false)

const rejectOpen = ref(false)
const rejectReason = ref('')
const publishOnApprove = ref(true)

async function load() {
  loading.value = true
  try {
    const [detail, trail] = await Promise.all([
      adminApi.property(route.params.id),
      adminApi.statusHistory(route.params.id).catch(() => ({ data: [] })),
    ])
    property.value = detail.data
    history.value = trail.data || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

async function startReview() {
  working.value = true
  try {
    await adminApi.startReview(route.params.id)
    ui.success('Moved to under review')
    await load()
  } catch (err) {
    ui.error(err.message)
  } finally {
    working.value = false
  }
}

async function approve() {
  const confirmed = await ui.confirm({
    title: `Approve ${property.value.name}?`,
    message: publishOnApprove.value
      ? 'The listing will be published and visible to buyers immediately.'
      : 'The listing will be approved but not yet published.',
    confirmLabel: 'Approve',
  })
  if (!confirmed) return

  working.value = true
  try {
    await adminApi.approveProperty(route.params.id, {
      remarks: '',
      publish: publishOnApprove.value,
    })
    ui.success('Property approved')
    await load()
  } catch (err) {
    ui.error(err.message)
  } finally {
    working.value = false
  }
}

async function reject() {
  if (rejectReason.value.trim().length < 5) {
    ui.error('Give the builder a reason they can act on')
    return
  }
  working.value = true
  try {
    await adminApi.rejectProperty(route.params.id, { reason: rejectReason.value.trim() })
    ui.success('Property rejected and the builder notified')
    rejectOpen.value = false
    rejectReason.value = ''
    await load()
  } catch (err) {
    ui.error(err.message)
  } finally {
    working.value = false
  }
}

async function setReraStatus(status) {
  working.value = true
  try {
    await adminApi.verifyRera(route.params.id, { status })
    ui.success('RERA verification updated')
    await load()
  } catch (err) {
    ui.error(err.message)
  } finally {
    working.value = false
  }
}

async function toggleFeatured() {
  working.value = true
  try {
    await adminApi.setFeatured(route.params.id, !property.value.is_featured)
    await load()
  } catch (err) {
    ui.error(err.message)
  } finally {
    working.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-else-if="loading" class="skeleton" style="height: 300px"></div>

    <template v-else-if="property">
      <div class="section-head">
        <div>
          <div class="row-wrap mb-1">
            <span class="badge" :class="`badge-${STATUS_VARIANT[property.property_status] || 'neutral'}`">
              {{ humanise(property.property_status) }}
            </span>
            <span v-if="property.is_featured" class="badge badge-featured">Featured</span>
            <span v-if="property.is_rera_verified" class="badge badge-rera">RERA verified</span>
          </div>
          <h2 class="mb-0">{{ property.name }}</h2>
          <p class="muted small mb-0">
            {{ property.locality?.name }}, {{ property.city?.name }} · by
            {{ property.builder?.company_name || property.builder_name }}
          </p>
        </div>
        <RouterLink class="btn btn-ghost" :to="{ name: 'admin-approvals' }">Back to queue</RouterLink>
      </div>

      <div class="review-layout">
        <div class="stack">
          <ImageGallery :media="property.media" :alt="property.name" />

          <section class="panel">
            <div class="panel-head"><h3>Project details</h3></div>
            <div class="detail-grid">
              <div><span class="stat-label">Type</span><strong>{{ humanise(property.property_type) }}</strong></div>
              <div><span class="stat-label">Construction</span><strong>{{ humanise(property.construction_status) }}</strong></div>
              <div><span class="stat-label">Price band</span><strong>{{ formatPriceRange(property.min_price, property.max_price) }}</strong></div>
              <div><span class="stat-label">Area</span><strong>{{ formatArea(property.min_area) }} - {{ formatArea(property.max_area) }}</strong></div>
              <div><span class="stat-label">Possession</span><strong>{{ formatDate(property.possession_date) }}</strong></div>
              <div><span class="stat-label">Pincode</span><strong>{{ property.pincode || '-' }}</strong></div>
            </div>
            <p class="small muted mt-2 mb-0">{{ property.description }}</p>
          </section>

          <section v-if="property.configurations?.length" class="panel">
            <div class="panel-head"><h3>Configurations</h3></div>
            <div class="table-wrap">
              <table class="table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th class="num">Carpet</th>
                    <th class="num">Price</th>
                    <th class="num">Units</th>
                    <th>Availability</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="config in property.configurations" :key="config.id">
                    <td><strong>{{ config.name }}</strong></td>
                    <td class="num">{{ formatArea(config.carpet_area) }}</td>
                    <td class="num">{{ formatPriceRange(config.price, config.price) }}</td>
                    <td class="num">{{ config.number_of_units }}</td>
                    <td>{{ humanise(config.availability) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section v-if="property.documents?.length" class="panel">
            <div class="panel-head"><h3>Documents</h3></div>
            <ul class="doc-list">
              <li v-for="doc in property.documents" :key="doc.id">
                <span class="grow">
                  <strong>{{ doc.name }}</strong>
                  <span class="tiny muted"> · {{ humanise(doc.doc_type) }}</span>
                </span>
                <a class="btn btn-outline btn-sm" :href="doc.file_url" target="_blank" rel="noopener">
                  Open
                </a>
              </li>
            </ul>
          </section>

          <section v-if="history.length" class="panel">
            <div class="panel-head"><h3>Approval history</h3></div>
            <ul class="history">
              <li v-for="entry in history" :key="entry.id">
                <span class="badge" :class="`badge-${STATUS_VARIANT[entry.to_status] || 'neutral'}`">
                  {{ humanise(entry.from_status) }} → {{ humanise(entry.to_status) }}
                </span>
                <span class="tiny muted">{{ formatDateTime(entry.created_at) }}</span>
                <span v-if="entry.remarks" class="small grow">{{ entry.remarks }}</span>
              </li>
            </ul>
          </section>
        </div>

        <aside class="side stack">
          <section class="panel">
            <div class="panel-head"><h3>Decision</h3></div>

            <div v-if="property.rejection_reason" class="alert alert-error mb-2 tiny">
              Previously rejected: {{ property.rejection_reason }}
            </div>

            <div class="stack">
              <button
                v-if="property.property_status === 'SUBMITTED'"
                class="btn btn-outline btn-block"
                :disabled="working"
                @click="startReview"
              >
                Start review
              </button>

              <label class="check">
                <input v-model="publishOnApprove" type="checkbox" />
                <span class="tiny">Publish immediately on approval</span>
              </label>

              <button class="btn btn-primary btn-block" :disabled="working" @click="approve">
                Approve listing
              </button>

              <button
                class="btn btn-danger btn-block"
                :disabled="working"
                @click="rejectOpen = !rejectOpen"
              >
                Reject listing
              </button>

              <div v-if="rejectOpen">
                <label class="field mb-1">
                  <span class="label">Reason for rejection</span>
                  <textarea
                    v-model="rejectReason"
                    class="textarea"
                    rows="4"
                    placeholder="Tell the builder exactly what to correct, for example: the RERA number does not match the project name."
                  ></textarea>
                </label>
                <button class="btn btn-danger btn-block" :disabled="working" @click="reject">
                  Confirm rejection
                </button>
              </div>

              <hr class="divider" />

              <button class="btn btn-outline btn-block" :disabled="working" @click="toggleFeatured">
                {{ property.is_featured ? 'Remove from featured' : 'Mark as featured' }}
              </button>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head"><h3>RERA verification</h3></div>

            <div v-if="property.rera_details?.registration_number" class="stack">
              <div>
                <span class="stat-label">Registration number</span>
                <strong class="mono">{{ property.rera_details.registration_number }}</strong>
              </div>
              <div>
                <span class="stat-label">Authority</span>
                <strong>{{ property.rera_details.authority || '-' }}</strong>
              </div>
              <div>
                <span class="stat-label">Registered</span>
                <strong>{{ formatDate(property.rera_details.registration_date) }}</strong>
              </div>
              <div>
                <span class="stat-label">Completion</span>
                <strong>{{ formatDate(property.rera_details.completion_date) }}</strong>
              </div>
              <div>
                <span class="stat-label">Current status</span>
                <span
                  class="badge"
                  :class="`badge-${RERA_STATUS_VARIANT[property.rera_details.verification_status] || 'neutral'}`"
                >
                  {{ humanise(property.rera_details.verification_status) }}
                </span>
              </div>

              <div class="row-wrap">
                <button class="btn btn-primary btn-sm" :disabled="working" @click="setReraStatus('VERIFIED')">
                  Mark verified
                </button>
                <button class="btn btn-outline btn-sm" :disabled="working" @click="setReraStatus('REJECTED')">
                  Reject RERA
                </button>
                <button
                  class="btn btn-ghost btn-sm"
                  :disabled="working"
                  @click="setReraStatus('PENDING_VERIFICATION')"
                >
                  Reset to pending
                </button>
              </div>
            </div>

            <p v-else class="muted small mb-0">
              No RERA details were supplied. Reject the listing and ask the builder to add them.
            </p>
          </section>

          <section class="panel">
            <div class="panel-head"><h3>Engagement</h3></div>
            <div class="mini-stats">
              <div><span class="stat-label">Views</span><strong>{{ formatNumber(property.view_count) }}</strong></div>
              <div><span class="stat-label">Inquiries</span><strong>{{ formatNumber(property.inquiry_count) }}</strong></div>
              <div><span class="stat-label">Saved</span><strong>{{ formatNumber(property.favorite_count) }}</strong></div>
            </div>
          </section>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.review-layout { display: grid; grid-template-columns: 1fr 340px; gap: 20px; align-items: start; }
.side { position: sticky; top: 78px; }

.detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 16px; }
.detail-grid > div { display: flex; flex-direction: column; gap: 3px; }

.mini-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.mini-stats > div { display: flex; flex-direction: column; gap: 2px; }

.mono { font-family: ui-monospace, 'SFMono-Regular', Menlo, monospace; font-size: 0.9rem; }

.doc-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
.doc-list li { display: flex; align-items: center; gap: 10px; padding: 10px 12px; border: 1px solid var(--ink-200); border-radius: var(--radius); }

.history { list-style: none; margin: 0; padding: 0; display: grid; gap: 10px; }
.history li { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }

@media (max-width: 1040px) {
  .review-layout { grid-template-columns: 1fr; }
  .side { position: static; }
}
</style>
