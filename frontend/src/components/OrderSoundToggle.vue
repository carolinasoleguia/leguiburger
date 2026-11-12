<script setup>
import {
  soundEnabled,
  soundError,
  toggleOrderSound,
} from "../composables/useOrderSound.js";
</script>
<template>
  <div class="order-sound">
    <button
      type="button"
      class="btn-secondary"
      :aria-pressed="soundEnabled"
      @click="toggleOrderSound"
    >
      {{ soundEnabled ? "Silenciar avisos" : "Activar sonido" }}</button
    ><small role="status">{{
      soundError ||
      (soundEnabled
        ? "Sonará cuando cambie el estado o se registre un pago."
        : "Activá los avisos sonoros para este navegador.")
    }}</small>
  </div>
</template>
<style scoped>
.order-sound {
  display: grid;
  gap: 8px;
  margin: 16px 0;
}
small {
  color: var(--text-muted);
  line-height: 1.5;
}
</style>
