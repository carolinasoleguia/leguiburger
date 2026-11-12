<script setup>
import AppSpinner from "./AppSpinner.vue";
defineProps({
  kind: { type: String, default: "empty" },
  title: String,
  description: String,
  retry: Function,
  view: { type: String, default: "auto" },
});
</script>
<template>
  <div
    v-if="kind === 'loading'"
    class="stable-loading-placeholder"
    data-dashboard-loading
    aria-busy="true"
  >
    <AppSpinner :size="28" :label="title || 'Cargando datos…'" />
  </div>
  <div
    v-else
    class="state-panel"
    :class="`state-panel--${kind}`"
    :role="kind === 'error' ? 'alert' : 'status'"
    :aria-busy="kind === 'loading'"
  >
    <span
      v-if="kind === 'loading'"
      class="state-spinner"
      aria-hidden="true"
    ></span>
    <div>
      <strong>{{
        title ||
        (kind === "loading" ? "Preparando tu espacio…" : "Todavía no hay datos")
      }}</strong>
      <p v-if="description">{{ description }}</p>
    </div>
    <button v-if="retry" class="btn-secondary" @click="retry">
      Reintentar
    </button>
  </div>
</template>
