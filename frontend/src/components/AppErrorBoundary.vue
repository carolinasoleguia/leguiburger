<script setup>
import { onErrorCaptured, ref, watch } from "vue";
import { useRoute } from "vue-router";
import StatePanel from "./StatePanel.vue";
import { reportClientError } from "../services/monitoring.js";
const failed = ref(false),
  route = useRoute();
onErrorCaptured(() => {
  failed.value = true;
  reportClientError("render");
  return false;
});
watch(
  () => route.fullPath,
  () => {
    failed.value = false;
  },
);
</script>
<template>
  <main v-if="failed" class="recovery-page">
    <StatePanel
      kind="error"
      title="No pudimos mostrar esta pantalla"
      description="Tus cambios guardados siguen disponibles. Podés volver a intentar."
      :retry="() => (failed = false)"
    /><router-link class="text-link" to="/">Volver al inicio</router-link>
  </main>
  <slot v-else />
</template>
