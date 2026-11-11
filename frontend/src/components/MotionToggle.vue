<script setup>
import AppIcon from "./AppIcon.vue";
import { useMotion } from "../composables/useMotion.js";
const { reduced, toggle } = useMotion();
</script>
<template>
  <button
    type="button"
    class="icon-button motion-toggle"
    :aria-label="reduced ? 'Activar animaciones' : 'Pausar animaciones'"
    :title="reduced ? 'Activar animaciones' : 'Pausar animaciones'"
    :aria-pressed="!reduced"
    @click="toggle"
  >
    <AppIcon :name="reduced ? 'play' : 'zap'" :size="18" />
  </button>
</template>
