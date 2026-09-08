<script setup>
import { computed } from 'vue'
import { formatNumber } from '@/utils/format'

const props = defineProps({
  title: { type: String, default: '' },
  series: { type: Array, default: () => [] }, // [{ date, count }]
  color: { type: String, default: 'var(--brand-600)' },
  height: { type: Number, default: 150 },
})

/**
 * A dependency-free sparkline. The dashboards need a shape and a total, not an
 * interactive charting library, so the SVG is generated inline and stays
 * readable at any width.
 */
const WIDTH = 600

const points = computed(() => {
  const data = props.series
  if (!data.length) return []
  const max = Math.max(...data.map((point) => point.count), 1)
  const stepX = data.length > 1 ? WIDTH / (data.length - 1) : 0
  return data.map((point, index) => ({
    x: index * stepX,
    y: props.height - (point.count / max) * (props.height - 14) - 4,
    ...point,
  }))
})

const linePath = computed(() =>
  points.value.map((point, i) => `${i === 0 ? 'M' : 'L'}${point.x.toFixed(1)},${point.y.toFixed(1)}`).join(' '),
)

const areaPath = computed(() => {
  if (!points.value.length) return ''
  const last = points.value[points.value.length - 1]
  return `${linePath.value} L${last.x.toFixed(1)},${props.height} L0,${props.height} Z`
})

const total = computed(() => props.series.reduce((sum, point) => sum + (point.count || 0), 0))
const peak = computed(() => Math.max(0, ...props.series.map((point) => point.count || 0)))
const gradientId = `grad-${Math.random().toString(36).slice(2, 9)}`
</script>

<template>
  <div class="chart">
    <div class="chart-head">
      <div>
        <span class="stat-label">{{ title }}</span>
        <div class="chart-total">{{ formatNumber(total) }}</div>
      </div>
      <span class="tiny muted">Peak {{ formatNumber(peak) }}</span>
    </div>

    <svg
      v-if="points.length"
      class="chart-svg"
      :viewBox="`0 0 ${600} ${height}`"
      preserveAspectRatio="none"
      role="img"
      :aria-label="`${title}: ${total} total`"
    >
      <defs>
        <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" :stop-color="color" stop-opacity="0.28" />
          <stop offset="100%" :stop-color="color" stop-opacity="0" />
        </linearGradient>
      </defs>
      <path :d="areaPath" :fill="`url(#${gradientId})`" />
      <path
        :d="linePath"
        fill="none"
        :stroke="color"
        stroke-width="2"
        stroke-linejoin="round"
        stroke-linecap="round"
        vector-effect="non-scaling-stroke"
      />
    </svg>

    <p v-else class="tiny muted center chart-empty">No activity in this period yet.</p>

    <div v-if="series.length" class="axis tiny muted">
      <span>{{ series[0].date }}</span>
      <span>{{ series[series.length - 1].date }}</span>
    </div>
  </div>
</template>

<style scoped>
.chart {
  background: var(--white);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-lg);
  padding: 16px 18px;
}

.chart-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
.chart-total { font-size: 1.4rem; font-weight: 800; color: var(--ink-900); margin-top: 2px; }

.chart-svg { width: 100%; height: 150px; margin-top: 8px; display: block; }
.chart-empty { padding: 44px 0; margin: 0; }

.axis { display: flex; justify-content: space-between; margin-top: 4px; }
</style>
