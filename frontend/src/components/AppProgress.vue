<script setup>
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { pendingRequests, pendingNavigation } from "../services/activity.js";
const active = computed(
    () => pendingRequests.value > 0 || pendingNavigation.value,
  ),
  visible = ref(false);
let timer;
watch(
  active,
  (value) => {
    clearTimeout(timer);
    if (value) timer = setTimeout(() => (visible.value = true), 120);
    else visible.value = false;
  },
  { immediate: true },
);
onBeforeUnmount(() => clearTimeout(timer));
</script>
<template>
  <Transition name="progress"
    ><div
      v-if="visible"
      class="app-progress"
      role="progressbar"
      aria-label="Cargando contenido"
    >
      <span></span></div
  ></Transition>
</template>
