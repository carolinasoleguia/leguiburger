<script setup>
import { computed, ref } from "vue";
import OrderCheckout from "./OrderCheckout.vue";
const props = defineProps({ tenant: String, menu: Object, orderId: String });
const emit = defineEmits(["closed"]),
  checkout = ref(null);
defineExpose({
  open: (item) => checkout.value?.open(item),
  ready: computed(() => !!checkout.value?.ready),
});
</script>
<template>
  <OrderCheckout ref="checkout" v-bind="props" @closed="emit('closed')">
    <template #benefits
      ><p class="buyer-muted">
        Podés pedir sin cuenta.
        <router-link :to="`/mi-cuenta?local=${props.tenant}`"
          >Ingresá para guardar tus compras y usar cupones.</router-link
        >
      </p></template
    >
  </OrderCheckout>
</template>

<style scoped>
.buyer-muted {
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}
</style>
