<script setup>
import { ref } from "vue";
import RestaurantMenu from "../../components/RestaurantMenu.vue";
import GuestCheckout from "../../components/GuestCheckout.vue";
defineProps({ sessionReady: { type: Boolean, default: true } });
const checkout = ref(null);
</script>
<template>
  <RestaurantMenu
    data-experience="guest"
    :checkout-ready="!!checkout?.ready"
    ordering-notice="Elegí tus favoritas. Pedí acá, sin crear una cuenta."
    @choose="checkout?.open($event)"
  >
    <template #account-link="{ tenant }"
      ><router-link
        class="text-link"
        :to="`/mi-cuenta?local=${encodeURIComponent(tenant)}`"
        >Mi cuenta</router-link
      ></template
    >
    <template #checkout="{ tenant, menu }"
      ><GuestCheckout
        v-if="tenant && sessionReady"
        :key="tenant"
        ref="checkout"
        :tenant="tenant"
        :menu="menu"
    /></template>
  </RestaurantMenu>
</template>
