<script setup>
// Entry resolution only. Each shopping experience owns its own page and actions.
import { onMounted } from "vue";
import { shopper, loadShopper } from "../../composables/useShopper.js";
import GuestCatalogView from "./GuestCatalogView.vue";
import ShopperCatalogView from "./ShopperCatalogView.vue";
onMounted(() => {
  if (!shopper.ready) loadShopper().catch(() => {});
});
</script>
<template>
  <ShopperCatalogView v-if="shopper.profile" /><GuestCatalogView
    v-else
    :session-ready="shopper.ready"
  />
</template>
