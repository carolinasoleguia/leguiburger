<script setup>
// Resolve the account entry; authentication and the customer workspace are separate pages.
import { ref } from "vue";
import { useRoute } from "vue-router";
import { shopper } from "../../composables/useShopper.js";
import ShopperAccessView from "./ShopperAccessView.vue";
import ShopperDashboardView from "./ShopperDashboardView.vue";
const route = useRoute(),
  action = ref(
    ["verify", "reset"].includes(route.query.accion) ? route.query.accion : "",
  ),
  notice = ref(""),
  error = ref("");
const token = new URLSearchParams(location.hash.slice(1)).get("token") || "";
if (token) history.replaceState(null, "", location.pathname + location.search);
function complete(message = "", failure = "") {
  action.value = "";
  notice.value = message;
  error.value = failure;
}
</script>
<template>
  <ShopperAccessView
    v-if="!shopper.profile || action"
    :initial-action="action"
    :token="token"
    @complete="complete"
  /><ShopperDashboardView
    v-else
    :initial-notice="notice"
    :initial-error="error"
  />
</template>
