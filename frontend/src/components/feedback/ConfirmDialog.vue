<script setup>
import { watch } from 'vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

// Destructive actions run through ui.confirm(), so the wording and the escape
// hatch are consistent everywhere instead of being re-invented per view.
function onKeydown(event) {
  if (event.key === 'Escape') ui.resolveConfirm(false)
}

watch(
  () => ui.confirmState,
  (state) => {
    if (state) {
      document.addEventListener('keydown', onKeydown)
      document.body.style.overflow = 'hidden'
    } else {
      document.removeEventListener('keydown', onKeydown)
      document.body.style.overflow = ''
    }
  },
)
</script>

<template>
  <Transition name="fade">
    <div
      v-if="ui.confirmState"
      class="dialog-backdrop"
      role="dialog"
      aria-modal="true"
      @click.self="ui.resolveConfirm(false)"
    >
      <div class="dialog">
        <h3>{{ ui.confirmState.title }}</h3>
        <p v-if="ui.confirmState.message" class="muted">{{ ui.confirmState.message }}</p>
        <div class="dialog-actions">
          <button class="btn btn-outline" @click="ui.resolveConfirm(false)">
            {{ ui.confirmState.cancelLabel }}
          </button>
          <button
            class="btn"
            :class="ui.confirmState.danger ? 'btn-danger' : 'btn-primary'"
            @click="ui.resolveConfirm(true)"
          >
            {{ ui.confirmState.confirmLabel }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.dialog-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 32, 0.45);
  display: grid;
  place-items: center;
  padding: 20px;
  z-index: 1100;
}

.dialog {
  background: var(--white);
  border-radius: var(--radius-lg);
  padding: 26px;
  width: min(440px, 100%);
  box-shadow: var(--shadow-lg);
}

.dialog h3 { margin-top: 0; }
.dialog p { margin-bottom: 22px; }

.dialog-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
}

.fade-enter-active,
.fade-leave-active { transition: opacity 160ms ease; }
.fade-enter-from,
.fade-leave-to { opacity: 0; }
</style>
