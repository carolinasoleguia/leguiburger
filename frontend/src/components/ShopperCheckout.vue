<script setup>
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import { shopper, shopperRequest } from "../composables/useShopper.js";
import OrderCheckout from "./OrderCheckout.vue";
const props = defineProps({
  tenant: String,
  menu: Object,
  orderId: String,
  repeatOrderId: String,
});
const emit = defineEmits(["closed"]),
  checkout = ref(null);
const couponCode = ref(String(useRoute().query.cupon || "")),
  rememberContact = ref(false);
const basketOptions = computed(() =>
  couponCode.value ? { coupon_code: couponCode.value } : {},
);
const orderOptions = computed(() =>
  rememberContact.value ? { remember_contact: true } : {},
);
const draftLoader = computed(() =>
  props.repeatOrderId
    ? () =>
        shopperRequest(
          `/api/shopper/orders/${encodeURIComponent(props.repeatOrderId)}/repeat`,
          {},
        )
    : undefined,
);
defineExpose({
  open: (item) => checkout.value?.open(item),
  ready: computed(() => !!checkout.value?.ready),
});
</script>
<template>
  <OrderCheckout
    ref="checkout"
    :tenant="tenant"
    :menu="menu"
    :order-id="orderId"
    :initial-contact="shopper.profile"
    :basket-options="basketOptions"
    :order-options="orderOptions"
    :load-draft="draftLoader"
    @closed="emit('closed')"
  >
    <template #benefits="{ locked, invalidate }"
      ><label class="buyer-coupon"
        >¿Tenés un cupón?<input
          v-model="couponCode"
          maxlength="32"
          placeholder="Código de esta marca"
          :disabled="locked"
          @input="invalidate" /></label
    ></template>
    <template #contact-options
      ><label class="buyer-remember"
        ><input v-model="rememberContact" type="checkbox" />Guardar teléfono y
        dirección para próximas compras</label
      ></template
    >
    <template #identity
      ><p class="buyer-muted">
        Comprando como {{ shopper.profile?.name }}. Este pedido se vincula a tu
        cuenta.
      </p></template
    >
  </OrderCheckout>
</template>
<style scoped>
.buyer-coupon {
  display: grid;
  gap: 8px;
  font-size: 13px;
}
.buyer-coupon input {
  width: 100%;
  min-height: 44px;
  padding: 11px 12px;
  background: var(--field);
  color: var(--text);
  border: 1px solid var(--control-border);
  border-radius: 8px;
  font: inherit;
}
.buyer-remember {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  line-height: 1.5;
}
.buyer-remember input {
  width: 18px;
  height: 18px;
  min-height: 18px;
  padding: 0;
  accent-color: var(--accent);
}
.buyer-muted {
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}
</style>
