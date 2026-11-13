<script setup>
import RestaurantDirectory from "../../components/RestaurantDirectory.vue";
import { shopper } from "../../composables/useShopper.js";
</script>
<template>
  <RestaurantDirectory
    ><template #intro
      ><p class="directory-session" role="status">
        Estás comprando como <strong>{{ shopper.profile?.name }}</strong>. Tus
        compras se guardan en Mis pedidos y podés usar tus cupones.
      </p></template
    ></RestaurantDirectory
  >
</template>
<style scoped>
.directory-session {
  border-left: 3px solid var(--accent);
  padding: 14px 18px;
  background: var(--surface);
  border-radius: 8px;
}
</style>
