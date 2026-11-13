<script setup>
import { ref, watch, onBeforeUnmount } from "vue";
import { useRoute } from "vue-router";
import CatalogLandingView from "./CatalogLandingView.vue";
const route = useRoute(),
  tenant = ref(""),
  loading = ref(true),
  error = ref("");
let controller;
async function resolveMenu() {
  controller?.abort();
  const current = (controller = new AbortController());
  tenant.value = "";
  loading.value = true;
  error.value = "";
  try {
    const response = await fetch(
      `/api/public/restaurants/${encodeURIComponent(route.params.brand)}/${encodeURIComponent(route.params.local)}`,
      { signal: current.signal, credentials: "omit" },
    );
    if (!response.ok)
      throw new Error(
        response.status === 404
          ? "Esta carta no está disponible."
          : "No pudimos cargar la carta. Volvé a intentar.",
      );
    const data = await response.json();
    if (controller === current) tenant.value = data.tenant_id;
  } catch (e) {
    if (e.name !== "AbortError" && controller === current)
      error.value = e.message;
  } finally {
    if (controller === current) loading.value = false;
  }
}
watch(() => [route.params.brand, route.params.local], resolveMenu, {
  immediate: true,
});
onBeforeUnmount(() => controller?.abort());
</script>
<template>
  <CatalogLandingView v-if="tenant" :key="tenant" :tenant-id="tenant" />
  <main v-else class="public-menu-main" aria-live="polite">
    <p v-if="loading" role="status">Buscando la carta…</p>
    <template v-else
      ><h1>{{ error }}</h1>
      <button class="btn-secondary" @click="resolveMenu">Reintentar</button>
      <router-link to="/restaurantes"
        >Explorar restaurantes</router-link
      ></template
    >
  </main>
</template>

<style scoped>
.public-menu-main {
  max-width: 960px;
  margin: 0 auto;
  padding: 48px 24px;
  min-height: 60vh;
}
</style>
