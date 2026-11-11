<script setup>
import { useAuth } from "../../composables/useAuth.js";
import { defineAsyncComponent } from "vue";
const KitchenView = defineAsyncComponent(
  () => import("../shared/KitchenView.vue"),
);
const OrdersView = defineAsyncComponent(
  () => import("../shared/OrdersView.vue"),
);
const auth = useAuth();
</script>
<template>
  <OrdersView v-if="auth.userRole.value === 'cashier'" /><KitchenView v-else />
</template>
