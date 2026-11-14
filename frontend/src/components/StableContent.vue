<script setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
const props = defineProps({
  loading: Boolean,
  busy: Boolean,
  label: { type: String, default: "Preparando tu espacio…" },
  view: { type: String, default: "auto" },
});
const route = useRoute();
const layout = computed(() =>
  props.view !== "auto"
    ? props.view
    : [
          "/notifications",
          "/onboarding",
          "/restaurants",
          "/platform/health",
        ].includes(route.path)
      ? "feed"
      : "page",
);
</script>
<template>
  <div
    class="stable-content"
    :class="[
      `stable-content--${layout}`,
      { 'is-loading': loading, 'is-ready': !loading },
    ]"
    :aria-busy="loading || busy"
  >
    <span
      v-if="loading"
      class="visually-hidden"
      role="status"
      aria-busy="true"
      data-dashboard-loading
      >{{ label }}</span
    >
    <div
      class="stable-content-body data-stage-content"
      :inert="loading || undefined"
      :aria-busy="busy"
    >
      <slot />
    </div>
  </div>
</template>
