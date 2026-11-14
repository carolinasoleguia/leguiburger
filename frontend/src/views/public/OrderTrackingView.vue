<script setup>
import { onMounted, ref } from "vue";
import { useOrderStream } from "../../composables/useOrderStream.js";
import { notifyOrderChange } from "../../composables/useOrderSound.js";
import OrderSoundToggle from "../../components/OrderSoundToggle.vue";
import BrandLogo from "../../components/BrandLogo.vue";
import { money, when, statusLabels } from "../../composables/useWorkspace.js";
const token = new URLSearchParams(location.hash.slice(1)).get("token") || "";
history.replaceState(null, "", location.pathname);
const order = ref(null),
  error = ref(""),
  loading = ref(true);

async function load() {
  error.value = "";
  try {
    const response = await fetch("/api/public/order", {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
      referrerPolicy: "no-referrer",
    });
    if (!response.ok)
      throw new Error(
        "El enlace venció, fue revocado o no es válido. Pedí uno nuevo al local.",
      );
    const result = await response.json();
    notifyOrderChange(order.value, result);
    order.value = result;
  } catch (e) {
    order.value = null;
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
onMounted(load);
const { live } = useOrderStream({
  path: "/api/public/order/events",
  token: () => token,
  enabled: () =>
    !!order.value &&
    !(
      ["canceled", "rejected"].includes(order.value.status) ||
      (order.value.status === "delivered" &&
        Number(order.value.total_paid) >= Number(order.value.total_due))
    ),
  refresh: load,
});
</script>
<template>
  <main class="public-order">
    <BrandLogo />
    <section class="card">
      <p class="eyebrow">TU PEDIDO, EN CAMINO</p>
      <h1>{{ order?.business || "Seguimiento de tu pedido" }}</h1>
      <p v-if="loading" role="status">Consultando el estado…</p>
      <p v-if="error" role="alert">{{ error }}</p>
      <OrderSoundToggle v-if="order" />
      <p v-if="order" role="status">
        {{
          live
            ? "Seguimiento en vivo"
            : "Seguimiento con actualización automática"
        }}
      </p>
      <template v-if="order"
        ><p v-if="order.environment === 'sandbox'" class="inline-notice">
          Pedido de prueba. No representa una compra real.
        </p>
        <h2>{{ statusLabels[order.status] }}</h2>
        <p>{{ when(order.created_at) }}</p>
        <div v-for="(item, i) in order.items" :key="i" class="report-row">
          <span>{{ item.quantity }} × {{ item.name }}</span>
        </div>
        <div class="report-row">
          <span>Total</span
          ><strong>{{ money(order.total_due, order.currency) }}</strong>
        </div>
        <div class="report-row">
          <span>Cobrado y confirmado</span
          ><strong>{{ money(order.total_paid, order.currency) }}</strong>
        </div>
        <p>
          El estado del pago se actualiza después de su confirmación. Para
          completar o consultar un pago, comunicate con el local.
        </p></template
      ><button class="btn-secondary" :disabled="loading" @click="load">
        Actualizar
      </button>
      <p>
        Este enlace es privado. Conservá la pestaña abierta; al recargar, volvé
        a abrir el enlace original.
      </p>
    </section>
  </main>
</template>
