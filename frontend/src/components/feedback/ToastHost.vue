<script setup>
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

const icons = {
  success: '✓',
  error: '!',
  info: 'i',
}
</script>

<template>
  <div class="toast-host" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="toast in ui.toasts" :key="toast.id" class="toast" :class="`toast-${toast.type}`">
        <span class="toast-icon" aria-hidden="true">{{ icons[toast.type] || 'i' }}</span>
        <span class="toast-message">{{ toast.message }}</span>
        <button class="toast-close" aria-label="Dismiss" @click="ui.dismiss(toast.id)">×</button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: min(380px, calc(100vw - 40px));
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding: 13px 15px;
  border-radius: var(--radius);
  background: var(--white);
  border: 1px solid var(--ink-200);
  border-left: 4px solid var(--ink-400);
  box-shadow: var(--shadow-lg);
  font-size: 0.9rem;
}

.toast-success { border-left-color: var(--success-600); }
.toast-error { border-left-color: var(--danger-600); }
.toast-info { border-left-color: var(--info-600); }

.toast-icon {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--white);
  background: var(--ink-500);
}
.toast-success .toast-icon { background: var(--success-600); }
.toast-error .toast-icon { background: var(--danger-600); }
.toast-info .toast-icon { background: var(--info-600); }

.toast-message { flex: 1; color: var(--ink-800); }

.toast-close {
  border: 0;
  background: none;
  color: var(--ink-400);
  font-size: 1.2rem;
  line-height: 1;
  cursor: pointer;
  padding: 0 2px;
}
.toast-close:hover { color: var(--ink-700); }

.toast-enter-active,
.toast-leave-active {
  transition: all 200ms ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(24px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(24px);
}

@media (max-width: 640px) {
  .toast-host { left: 16px; right: 16px; bottom: 16px; max-width: none; }
}
</style>
