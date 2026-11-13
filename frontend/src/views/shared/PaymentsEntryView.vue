<script setup>
import { computed, defineAsyncComponent } from "vue";
import { useAuth } from "../../composables/useAuth.js";
const auth = useAuth();
const AdminPayments = defineAsyncComponent(
  () => import("../admins/AdminPaymentsView.vue"),
);
const OperationalPayments = defineAsyncComponent(
  () => import("./PaymentsView.vue"),
);
const experience = computed(() =>
  auth.userRole.value === "admin" ? AdminPayments : OperationalPayments,
);
</script>
<template><component :is="experience" /></template>
