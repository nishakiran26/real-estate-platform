<script setup>
import { computed, ref, watch } from 'vue'
import { PLACEHOLDER_IMAGE } from '@/utils/constants'
import { humanise } from '@/utils/format'

const props = defineProps({
  media: { type: Array, default: () => [] },
  alt: { type: String, default: 'Property image' },
})

const active = ref(0)
const lightbox = ref(false)

const images = computed(() =>
  props.media.length
    ? props.media
    : [{ id: 0, file_url: PLACEHOLDER_IMAGE, media_type: 'PROPERTY_IMAGE' }],
)

const current = computed(() => images.value[active.value] || images.value[0])

watch(() => props.media, () => (active.value = 0))

function step(delta) {
  const total = images.value.length
  active.value = (active.value + delta + total) % total
}

function onKey(event) {
  if (event.key === 'ArrowRight') step(1)
  if (event.key === 'ArrowLeft') step(-1)
  if (event.key === 'Escape') lightbox.value = false
}
</script>

<template>
  <div class="gallery" @keydown="onKey">
    <div class="stage">
      <img :src="current.file_url" :alt="alt" @click="lightbox = true" />
      <span class="type-chip">{{ humanise(current.media_type) }}</span>

      <template v-if="images.length > 1">
        <button class="nav prev" aria-label="Previous image" @click="step(-1)">‹</button>
        <button class="nav next" aria-label="Next image" @click="step(1)">›</button>
        <span class="counter">{{ active + 1 }} / {{ images.length }}</span>
      </template>
    </div>

    <div v-if="images.length > 1" class="thumbs">
      <button
        v-for="(image, index) in images"
        :key="image.id || index"
        class="thumb"
        :class="{ active: index === active }"
        :aria-label="`View image ${index + 1}`"
        @click="active = index"
      >
        <img :src="image.file_url" :alt="`${alt} thumbnail ${index + 1}`" loading="lazy" />
      </button>
    </div>

    <Teleport to="body">
      <div v-if="lightbox" class="lightbox" @click.self="lightbox = false">
        <button class="lightbox-close" aria-label="Close" @click="lightbox = false">×</button>
        <button v-if="images.length > 1" class="nav prev" aria-label="Previous" @click="step(-1)">‹</button>
        <img :src="current.file_url" :alt="alt" />
        <button v-if="images.length > 1" class="nav next" aria-label="Next" @click="step(1)">›</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.stage {
  position: relative;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--ink-100);
  aspect-ratio: 16 / 9;
}
.stage img { width: 100%; height: 100%; object-fit: cover; cursor: zoom-in; }

.type-chip {
  position: absolute;
  top: 12px;
  left: 12px;
  padding: 4px 11px;
  border-radius: var(--radius-full);
  background: rgba(8, 38, 42, 0.8);
  color: var(--white);
  font-size: 0.74rem;
  font-weight: 600;
}

.counter {
  position: absolute;
  bottom: 12px;
  right: 12px;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: rgba(8, 38, 42, 0.75);
  color: var(--white);
  font-size: 0.76rem;
  font-variant-numeric: tabular-nums;
}

.nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 38px;
  height: 38px;
  border-radius: 50%;
  border: 0;
  background: rgba(255, 255, 255, 0.9);
  color: var(--ink-800);
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  display: grid;
  place-items: center;
}
.nav:hover { background: var(--white); }
.prev { left: 12px; }
.next { right: 12px; }

.thumbs {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
}
.thumb {
  flex: 0 0 92px;
  height: 62px;
  padding: 0;
  border: 2px solid transparent;
  border-radius: var(--radius);
  overflow: hidden;
  cursor: pointer;
  background: none;
  opacity: 0.65;
  transition: all var(--transition);
}
.thumb img { width: 100%; height: 100%; object-fit: cover; }
.thumb:hover { opacity: 1; }
.thumb.active { border-color: var(--brand-600); opacity: 1; }

.lightbox {
  position: fixed;
  inset: 0;
  z-index: 1200;
  background: rgba(8, 15, 20, 0.94);
  display: grid;
  place-items: center;
  padding: 40px;
}
.lightbox img { max-width: 92vw; max-height: 88vh; object-fit: contain; border-radius: var(--radius); }
.lightbox-close {
  position: absolute;
  top: 18px;
  right: 22px;
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.14);
  color: var(--white);
  font-size: 1.6rem;
  cursor: pointer;
}
</style>
