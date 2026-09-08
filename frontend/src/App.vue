<script setup>
import { RouterView } from 'vue-router'
import ToastHost from '@/components/feedback/ToastHost.vue'
import ConfirmDialog from '@/components/feedback/ConfirmDialog.vue'

// The shell mounts the singletons that any view can reach through the ui store,
// so no view has to render its own toast or dialog host.
</script>

<template>
  <RouterView v-slot="{ Component }">
    <Transition name="page" mode="out-in">
      <component :is="Component" />
    </Transition>
  </RouterView>

  <ToastHost />
  <ConfirmDialog />
</template>

<style>
.page-enter-active,
.page-leave-active {
  transition: opacity 140ms ease;
}
.page-enter-from,
.page-leave-to {
  opacity: 0;
}
</style>
