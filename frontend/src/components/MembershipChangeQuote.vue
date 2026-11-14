<script setup>
import { computed, ref, watch } from "vue";
import { apiFetch } from "../services/api.js";

const props = defineProps({
  subscription: { type: Object, required: true },
  plans: { type: Array, default: () => [] },
});
const locations = ref(
    Math.max(1, Number(props.subscription.purchased_locations || 1)) +
      (props.subscription.membership_tier === "premium" ? 1 : 0),
  ),
  price = ref(""),
  quote = ref(null),
  error = ref(""),
  busy = ref(false);
const keys = new Map();
const options = computed(() => {
  const available = props.plans.filter(
    (p) =>
      p.membership_tier === "premium" &&
      p.currency === props.subscription.currency,
  );
  if (
    props.subscription.membership_tier === "premium" &&
    !available.some((p) => p.id === props.subscription.plan_price_id)
  ) {
    return [
      {
        id: props.subscription.plan_price_id,
        name: "Conservar mi tarifa Premium contratada",
      },
      ...available,
    ];
  }
  return available;
});
watch(
  options,
  (items) => {
    if (!items.some((p) => p.id === price.value))
      price.value =
        items.find((p) => p.id === props.subscription.plan_price_id)?.id ||
        items[0]?.id ||
        "";
  },
  { immediate: true },
);
watch([locations, price], () => {
  quote.value = null;
  error.value = "";
});
watch(
  () => props.subscription.version,
  () => {
    quote.value = null;
    keys.clear();
  },
);
const money = (value, currency = "ARS") =>
  new Intl.NumberFormat("es-AR", { style: "currency", currency }).format(
    Number(value),
  );
async function preview() {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  quote.value = null;
  const body = {
      price_id: price.value,
      purchased_locations: Number(locations.value),
    },
    digest = JSON.stringify(body);
  const sourceVersion = props.subscription.version;
  if (!keys.has(digest)) keys.set(digest, crypto.randomUUID());
  try {
    const response = await apiFetch(
      `/billing/subscriptions/${props.subscription.id}/change-quote`,
      {
        method: "POST",
        headers: { "Idempotency-Key": keys.get(digest) },
        body: digest,
      },
    );
    const data = await response.json().catch(() => ({}));
    if (!response.ok)
      throw new Error(
        data.message || "No se pudo cotizar la ampliación. Podés reintentar.",
      );
    keys.delete(digest);
    if (props.subscription.version === sourceVersion) quote.value = data;
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <section class="card change-quote" :aria-busy="busy">
    <h3>Calcular una ampliación</h3>
    <p>
      Revisá el total mensual para más tiendas o para pasar de Clásico a
      Premium. La cotización no modifica tu membresía ni genera cargos.
    </p>
    <form @submit.prevent="preview" class="quote-fields">
      <div class="input-group">
        <label :for="`change-price-${subscription.id}`">Tarifa Premium</label
        ><select
          :id="`change-price-${subscription.id}`"
          v-model="price"
          :disabled="busy"
          required
        >
          <option v-for="item in options" :key="item.id" :value="item.id">
            {{ item.name }}
          </option>
        </select>
      </div>
      <div class="input-group">
        <label :for="`change-stores-${subscription.id}`"
          >Total de tiendas a contratar</label
        ><input
          :id="`change-stores-${subscription.id}`"
          v-model="locations"
          type="number"
          :min="subscription.purchased_locations || 1"
          max="10000"
          step="1"
          :disabled="busy"
          required
        />
      </div>
      <button type="submit" class="btn-secondary" :disabled="busy || !price">
        <span v-if="busy" class="spinner" aria-hidden="true"></span
        >{{ busy ? "Calculando…" : "Cotizar ampliación" }}
      </button>
    </form>
    <p v-if="!options.length">
      Todavía no hay una tarifa Premium disponible para ampliar esta membresía.
    </p>
    <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
    <div v-if="quote" class="quote-result" role="status">
      <div>
        <span>Mensual actual · {{ quote.source_locations }} tienda(s)</span
        ><strong>{{ money(quote.source_amount, quote.currency) }}</strong>
      </div>
      <div>
        <span>Mensual cotizado · {{ quote.purchased_locations }} tienda(s)</span
        ><strong>{{ money(quote.amount, quote.currency) }}</strong>
      </div>
      <p>
        Tu período actual está pagado hasta
        {{ new Date(quote.source_period_end).toLocaleString("es-AR") }}.
        Cotización válida hasta
        {{ new Date(quote.expires_at).toLocaleString("es-AR") }}.
      </p>
      <p>
        Se conserva tu cupo actual. La aplicación y el cobro del cambio
        requieren una confirmación posterior.
      </p>
    </div>
  </section>
</template>
<style scoped>
.change-quote {
  padding: 1.2rem;
  margin-block: 1rem;
}
.change-quote h3 {
  margin: 0 0 0.5rem;
}
.change-quote p {
  font-size: 0.85rem;
  color: var(--text-muted);
  line-height: 1.55;
}
.quote-fields {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr) auto;
  gap: 0.8rem;
  align-items: end;
}
.quote-fields .input-group {
  margin: 0;
  min-width: 0;
}
.quote-fields button {
  min-height: 42px;
  display: flex;
  gap: 0.4rem;
  align-items: center;
  justify-content: center;
}
.quote-result {
  margin-top: 1rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  border-top: 1px solid var(--border);
  padding-top: 1rem;
}
.quote-result span {
  display: block;
  font-size: 0.8rem;
  color: var(--text-muted);
}
.quote-result strong {
  font-size: 1.35rem;
}
.quote-result p {
  grid-column: 1/-1;
  margin: 0;
}
@media (max-width: 720px) {
  .quote-fields,
  .quote-result {
    grid-template-columns: 1fr;
  }
  .quote-result p {
    grid-column: auto;
  }
}
</style>
