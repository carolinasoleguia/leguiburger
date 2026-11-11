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
  <AppProgress />
</template>

<script setup>
import AppErrorBoundary from "./components/AppErrorBoundary.vue";
import AppConfirmation from "./components/AppConfirmation.vue";
import AppProgress from "./components/AppProgress.vue";
</script>
