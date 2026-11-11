<template>
  <AppErrorBoundary><router-view /></AppErrorBoundary>
  <AppConfirmation />
</template>

<script setup>
import AppErrorBoundary from './components/AppErrorBoundary.vue';
import AppConfirmation from './components/AppConfirmation.vue';
</script>
