<script setup>
import PropertiesView from './PropertiesView.vue'
</script>

<template>
  <!-- The queue is the property table pinned to the pending statuses; keeping
       one implementation means the two screens cannot drift apart. -->
  <PropertiesView queue-mode />
</template>
