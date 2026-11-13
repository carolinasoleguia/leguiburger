<script setup>
import { onMounted } from "vue";
import { shopper, loadShopper } from "../../composables/useShopper.js";
import GuestDirectoryView from "./GuestDirectoryView.vue";
import ShopperDirectoryView from "./ShopperDirectoryView.vue";
onMounted(() => {
  if (!shopper.ready) loadShopper().catch(() => {});
});
</script>
<template>
  <ShopperDirectoryView v-if="shopper.profile" /><GuestDirectoryView v-else />
</template>
