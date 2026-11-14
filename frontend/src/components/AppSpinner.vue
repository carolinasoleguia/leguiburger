<script setup>
defineProps({
  size: { type: Number, default: 24 },
  label: { type: String, default: "Cargando…" },
});
</script>
<template>
  <span class="app-spinner-wrap" role="status"
    ><svg
      class="app-spinner"
      :width="size"
      :height="size"
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
    >
      <path
        d="M20.9 13a9 9 0 1 1-9.9-9.9"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
      /></svg
    ><span class="visually-hidden">{{ label }}</span></span
  >
</template>
