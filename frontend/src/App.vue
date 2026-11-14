<template>
  <AppErrorBoundary
    ><router-view v-slot="{ Component, route }"
      ><Transition name="screen" mode="out-in"
        ><component
          :is="Component"
          :key="
            route.matched[0]?.path || route.path
          " /></Transition></router-view
  ></AppErrorBoundary>
  <AppConfirmation />
</template>

<script setup>
import AppErrorBoundary from "./components/AppErrorBoundary.vue";
import AppConfirmation from "./components/AppConfirmation.vue";
</script>
