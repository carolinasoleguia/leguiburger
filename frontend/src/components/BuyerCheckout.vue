<script setup>
import { computed, onBeforeUnmount, ref } from "vue";
import { useOrderStream } from "../composables/useOrderStream.js";
import { notifyOrderChange } from "../composables/useOrderSound.js";
import OrderSoundToggle from "./OrderSoundToggle.vue";
import { money, statusLabels } from "../composables/useWorkspace.js";
const props = defineProps({
  tenant: { type: String, required: true },
  menu: Object,
});
const cart = ref([]),
  selected = ref(null),
  extras = ref([]),
  quantity = ref(1),
  note = ref("");
const panel = ref(false),
  step = ref("cart"),
  busy = ref(false),
  error = ref(""),
  quote = ref(null),
  receipt = ref(null);
const fulfillment = ref("pickup"),
  shipping = ref(""),
  name = ref(""),
  phone = ref(""),
  email = ref(""),
  address = ref(""),
  payment = ref("at_local");
const recoveryMissing = ref(false);
const storageKey = "legui-buyer:" + props.tenant;
let access = null,
  pendingBody = null,
  alive = true;
const controllers = new Set();
function saveAccess() {
  // Capabilities stay in this tab; contact details and card data are never persisted.
  sessionStorage.setItem(storageKey, JSON.stringify(access));
}
try {
  const saved = JSON.parse(sessionStorage.getItem(storageKey) || "null");
  if (
    saved?.tenant === props.tenant &&
    saved.expires > Date.now() &&
    /^[A-Za-z0-9_-]{43}$/.test(saved.secret)
  )
    access = saved;
  else sessionStorage.removeItem(storageKey);
} catch {
  /* Storage may be unavailable. Confirming requires it to work. */
}
function cents(raw) {
  const [whole, fraction = ""] = String(raw || "0").split(".");
  return BigInt(whole) * 100n + BigInt((fraction + "00").slice(0, 2));
}
function decimal(value) {
  return `${value / 100n}.${String(value % 100n).padStart(2, "0")}`;
}
const count = computed(() =>
  cart.value.reduce((sum, item) => sum + item.quantity, 0),
);
const estimated = computed(() =>
  decimal(
    cart.value.reduce(
      (sum, item) =>
        sum +
        (cents(item.price) +
          item.extras.reduce((n, x) => n + cents(x.price), 0n)) *
          BigInt(item.quantity),
      0n,
    ),
  ),
);
const locked = computed(
  () =>
    busy.value ||
    !!pendingBody ||
    (!!access && !receipt.value && !recoveryMissing.value),
);
const unpaid = computed(
  () =>
    receipt.value &&
    cents(receipt.value.total_paid) < cents(receipt.value.total_due),
);
const canPay = computed(
  () =>
    receipt.value?.online_available &&
    unpaid.value &&
    !["pending", "canceled", "rejected"].includes(receipt.value.status),
);
const unresolvedPayment = computed(() =>
  ["prepared", "submitting", "review"].includes(receipt.value?.payment?.state),
);
function basket() {
  return {
    items: cart.value.map((x) => ({
      product_id: x.id,
      quantity: x.quantity,
      extras: x.extras.map((e) => ({ extra_id: e.id, quantity: 1 })),
      note: x.note,
    })),
    fulfillment: fulfillment.value,
    ...(fulfillment.value === "delivery"
      ? { shipping_method_id: shipping.value }
      : {}),
  };
}
async function request(path, body, authenticated = false, key = "") {
  const controller = new AbortController();
  controllers.add(controller);
  const timeout = setTimeout(() => controller.abort(), 25000);
  try {
    const response = await fetch(path, {
      method: body === undefined ? "GET" : "POST",
      credentials: "omit",
      signal: controller.signal,
      headers: {
        "Content-Type": "application/json",
        ...(authenticated ? { Authorization: "Bearer " + access.secret } : {}),
        ...(key ? { "Idempotency-Key": key } : {}),
      },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    const result = await response.json();
    if (!response.ok) {
      const e = new Error(
        result.message ||
          "No pudimos completar la solicitud. Volvé a intentar.",
      );
      e.status = response.status;
      throw e;
    }
    return result;
  } finally {
    clearTimeout(timeout);
    controllers.delete(controller);
  }
}
const canStartNew = computed(
  () =>
    !!receipt.value &&
    !unresolvedPayment.value &&
    (["canceled", "rejected"].includes(receipt.value.status) ||
      (receipt.value.status === "delivered" && !unpaid.value)),
);
function open(item) {
  if (!props.menu?.ordering_enabled || locked.value || receipt.value) {
    panel.value = true;
    return;
  }
  selected.value = item;
  extras.value = [];
  quantity.value = 1;
  note.value = "";
  error.value = "";
}
function add() {
  if (
    !selected.value ||
    count.value + quantity.value > 100 ||
    cart.value.length >= 30
  ) {
    error.value = "Podés sumar hasta 100 productos en 30 selecciones.";
    return;
  }
  cart.value.push({
    ...selected.value,
    quantity: quantity.value,
    extras: (props.menu?.extras || []).filter((x) =>
      extras.value.includes(x.id),
    ),
    note: note.value.trim(),
  });
  selected.value = null;
  quote.value = null;
}
async function review() {
  if (locked.value) return;
  busy.value = true;
  error.value = "";
  quote.value = null;
  try {
    const result = await request(
      `/api/public/menu/${props.tenant}/quote`,
      basket(),
    );
    if (alive) {
      quote.value = result;
      step.value = "checkout";
    }
  } catch (e) {
    if (alive) error.value = e.message;
  } finally {
    if (alive) busy.value = false;
  }
}
async function refreshReceipt(silent = false) {
  if (!access) return;
  try {
    const result = await request("/api/public/buyer-order", undefined, true);
    if (!alive) return;
    if (result.tenant_id !== props.tenant)
      throw new Error("El pedido pertenece a otro local.");
    notifyOrderChange(receipt.value, result);
    receipt.value = result;
    cart.value = [];
    pendingBody = null;
    step.value = "receipt";
    error.value = "";
  } catch (e) {
    if (!alive) return;
    if (!silent)
      error.value =
        e.status === 404
          ? "No encontramos un pedido con esta confirmación. Podés reintentar el envío con los mismos datos."
          : "No pudimos consultar el pedido. Actualizá su estado antes de volver a comprar.";
    if (e.status === 404 && !pendingBody) {
      // Keep the creation key even if an earlier request is still in flight.
      // Rebuilding the basket can never create a second order for that key.
      recoveryMissing.value = true;
    }
  }
}
async function confirm() {
  if (busy.value || !quote.value || receipt.value) return;
  busy.value = true;
  error.value = "";
  try {
    if (!access) {
      const bytes = crypto.getRandomValues(new Uint8Array(32));
      access = {
        tenant: props.tenant,
        secret: btoa(String.fromCharCode(...bytes))
          .replaceAll("+", "-")
          .replaceAll("/", "_")
          .replaceAll("=", ""),
        key: crypto.randomUUID(),
        paymentKey: crypto.randomUUID(),
        expires: Date.now() + 86400000,
      };
    }
    saveAccess(); // Do not send if the tab cannot retain the receipt capability.
    if (!pendingBody)
      pendingBody = {
        ...basket(),
        quote_digest: quote.value.digest,
        name: name.value,
        phone: phone.value,
        email: email.value,
        address: fulfillment.value === "delivery" ? address.value : "",
        payment_method: payment.value,
      };
    const result = await request(
      `/api/public/menu/${props.tenant}/orders`,
      pendingBody,
      true,
      access.key,
    );
    if (alive) {
      notifyOrderChange(receipt.value, result);
      receipt.value = result;
      pendingBody = null;
      cart.value = [];
      step.value = "receipt";
    }
  } catch (e) {
    if (!alive) return;
    error.value = e.status
      ? e.message
      : "No recibimos la confirmación. Consultá el estado o reintentá: no se duplicará tu pedido.";
    if (e.status && e.status >= 400 && e.status < 500 && e.status !== 429) {
      pendingBody = null;
      recoveryMissing.value = true;
      if (e.status === 409) {
        quote.value = null;
        await refreshReceipt(true);
      }
    }
  } finally {
    if (alive) busy.value = false;
  }
}
async function pay() {
  if (busy.value || !canPay.value || unresolvedPayment.value) return;
  busy.value = true;
  error.value = "";
  try {
    const result = receipt.value.payment?.checkout_url
      ? receipt.value.payment
      : await request(
          "/api/public/buyer-order/mercadopago",
          {},
          true,
          access.paymentKey,
        );
    if (!alive) return;
    if (
      result.state === "failed" ||
      (result.state === "created" &&
        result.expires_at &&
        Date.parse(result.expires_at) <= Date.now())
    ) {
      access.paymentKey = crypto.randomUUID();
      saveAccess();
      await refreshReceipt();
      error.value =
        "Este intento no está disponible. Podés volver a iniciar el pago.";
      return;
    }
    if (result.state !== "created" || !result.checkout_url) {
      await refreshReceipt();
      return;
    }
    const url = new URL(result.checkout_url);
    if (
      url.protocol !== "https:" ||
      !/(^|\.)mercadopago\.(com|com\.ar)$/.test(url.hostname)
    )
      throw new Error("No pudimos validar el enlace de pago.");
    window.location.assign(url.href);
  } catch (e) {
    if (alive) error.value = e.message;
  } finally {
    if (alive) busy.value = false;
  }
}
function newOrder() {
  if (!canStartNew.value) return;
  access = null;
  receipt.value = null;
  quote.value = null;
  pendingBody = null;
  step.value = "cart";
  sessionStorage.removeItem(storageKey);
  name.value = "";
  phone.value = "";
  email.value = "";
  address.value = "";
  panel.value = false;
}
if (access) {
  panel.value = true;
  refreshReceipt();
}
const { live } = useOrderStream({
  path: "/api/public/buyer-order/events",
  token: () => access?.secret || "",
  enabled: () =>
    !!receipt.value &&
    !(
      ["canceled", "rejected"].includes(receipt.value.status) ||
      (receipt.value.status === "delivered" &&
        !unpaid.value &&
        !unresolvedPayment.value)
    ),
  refresh: () => refreshReceipt(true),
});
onBeforeUnmount(() => {
  alive = false;

  for (const controller of controllers) controller.abort();
});
defineExpose({ open });
</script>
<template>
  <button
    v-if="menu?.ordering_enabled || receipt || access"
    class="buyer-cart-trigger btn-primary"
    @click="panel = true"
  >
    <span>{{ receipt ? "Ver mi pedido" : "Mi pedido" }}</span><strong>{{
      receipt ? "#" + receipt.id.slice(0, 8) : `${count} · ${money(estimated)}`
    }}</strong>
  </button>
  <Teleport to="body">
    <Transition name="modal"
      ><div v-if="selected" class="modal-overlay" @click.self="selected = null">
        <section v-dialog-care class="modal-card buyer-dialog">
          <p class="eyebrow">A tu gusto</p>
          <h2>{{ selected.name }}</h2>
          <p>{{ selected.description }}</p>
          <form class="buyer-form" @submit.prevent="add">
            <fieldset v-if="menu?.extras?.length">
              <legend>¿Le sumamos algo?</legend>
              <label
                v-for="extra in menu.extras"
                :key="extra.id"
                class="buyer-extra"
                ><input
                  v-model="extras"
                  type="checkbox"
                  :value="extra.id"
                  :disabled="
                    !extra.available ||
                    (!extras.includes(extra.id) && extras.length >= 10)
                  "
                />{{ extra.name }} <span>{{ money(extra.price) }}</span></label
              >
            </fieldset>
            <label
              >Cantidad<input
                v-model.number="quantity"
                type="number"
                min="1"
                max="20"
                required
            /></label>
            <label
              >Preferencias para la cocina
              <span class="buyer-muted">(opcional)</span
              ><textarea
                v-model="note"
                rows="2"
                maxlength="300"
                placeholder="Por ejemplo, sin cebolla"
              />
            </label>
            <p v-if="error" role="alert" class="error-msg">{{ error }}</p>
            <button class="btn-primary">Sumar al pedido</button>
          </form>
          <button
            class="modal-close"
            aria-label="Cerrar producto"
            @click="selected = null"
          >
            ×
          </button>
        </section>
      </div></Transition
    >
    <Transition name="modal"
      ><div
        v-if="panel"
        class="modal-overlay buyer-overlay"
        @click.self="panel = false"
      >
        <section
          v-dialog-care
          class="modal-card buyer-dialog buyer-sheet"
          :aria-busy="busy"
        >
          <p class="eyebrow">
            {{
              menu?.location?.brand_name ||
              receipt?.business ||
              "Tu restaurante"
            }}
          </p>
          <h2>
            {{ receipt ? "Tu pedido" : "Lo bueno está por llegar" }}
          </h2>
          <button
            class="modal-close"
            aria-label="Cerrar pedido"
            @click="panel = false"
          >
            ×
          </button>
          <p v-if="error" role="alert" class="error-msg">{{ error }}</p>
          <template v-if="receipt">
            <p v-if="!canStartNew" class="inline-notice" role="status">
              {{
                receipt.status === "delivered"
                  ? "Tu pedido ya fue entregado. Completá el pago pendiente antes de hacer otro."
                  : "Ya tenés un pedido en curso en este local. Podrás hacer otro cuando lo recibas."
              }}
            </p>
            <OrderSoundToggle /><small role="status">{{
              live
                ? "Seguimiento en vivo"
                : "Seguimiento con actualización automática"
            }}</small>
            <div class="buyer-receipt-status" role="status">
              <small>Pedido #{{ receipt.id.slice(0, 8) }}</small>
              <h3>{{ statusLabels[receipt.status] || receipt.status }}</h3>
              <p>
                {{
                  receipt.status === "pending"
                    ? "El local debe aceptar tu pedido antes de prepararlo. Todavía no realizaste ningún pago."
                    : receipt.status === "rejected" ||
                        receipt.status === "canceled"
                      ? "El pedido no continuará. Si ya pagaste, contactá al local para consultar la devolución."
                      : "Seguí acá el avance de tu pedido."
                }}
              </p>
            </div>
            <div
              v-for="(item, index) in receipt.items"
              :key="index"
              class="buyer-line"
            >
              <strong>{{ item.quantity }} × {{ item.name }}</strong
              ><small v-for="x in item.extras" :key="x.name"
                >+ {{ x.quantity }} {{ x.name }}</small
              ><small v-if="item.note">{{ item.note }}</small>
            </div>
            <p>
              {{
                receipt.fulfillment === "delivery"
                  ? "Envío a domicilio"
                  : "Retiro en el local"
              }}
              · {{ receipt.location }}
            </p>
            <div class="buyer-total">
              <span>Total</span><strong>{{ money(receipt.total_due) }}</strong>
            </div>
            <p class="buyer-payment-state">
              {{ unpaid ? "Pendiente de pago" : "Pago registrado" }} ·
              {{ money(receipt.total_paid) }} recibido
            </p>
            <p
              v-if="
                receipt.payment_method === 'mercadopago' &&
                receipt.status === 'pending'
              "
            >
              Podrás pagar con Mercado Pago cuando el local lo acepte.
            </p>
            <p v-else-if="receipt.payment_method === 'at_local' && unpaid">
              Pagás al retirar o recibir. Coordiná el medio de pago con el
              local.
            </p>
            <p v-if="unresolvedPayment">
              Estamos verificando el intento de pago. No vuelvas a pagar
              mientras se confirma.
            </p>
            <div class="buyer-actions">
              <button
                v-if="canPay && !unresolvedPayment"
                class="btn-primary"
                :disabled="busy"
                @click="pay"
              >
                Pagar con Mercado Pago</button
              ><button
                class="btn-secondary"
                :disabled="busy"
                @click="refreshReceipt()"
              >
                Actualizar estado</button
              ><a
                v-if="receipt.tracking_path"
                class="text-link"
                :href="receipt.tracking_path"
                >Abrir seguimiento</a
              ><button
                v-if="canStartNew"
                class="btn-secondary"
                @click="newOrder"
              >
                Armar otro pedido
              </button>
            </div>
            <p class="buyer-muted">
              La confirmación se conserva en esta pestaña durante 24 horas.
              Guardá el enlace de seguimiento para consultarlo después.
            </p>
          </template>
          <template v-else-if="access && !pendingBody && !recoveryMissing"
            ><p>Consultando tu última confirmación…</p>
            <button class="btn-secondary" @click="refreshReceipt()">
              Consultar pedido
            </button></template
          >
          <template v-else>
            <p v-if="!cart.length" class="buyer-empty">
              Todavía no elegiste tu favorita. Explorá la carta y sumá algo
              rico.
            </p>
            <div v-for="(item, index) in cart" :key="index" class="buyer-line">
              <div class="buyer-line-heading">
                <strong>{{ item.quantity }} × {{ item.name }}</strong
                ><button
                  type="button"
                  class="text-button"
                  :aria-label="`Quitar ${item.name}`"
                  :disabled="locked"
                  @click="
                    cart.splice(index, 1);
                    quote = null;
                    step = 'cart';
                  "
                >
                  Quitar
                </button>
              </div>
              <small v-for="x in item.extras" :key="x.id">+ {{ x.name }}</small
              ><small v-if="item.note">{{ item.note }}</small>
            </div>
            <template v-if="cart.length">
              <fieldset :disabled="locked" class="buyer-fulfillment">
                <legend>¿Cómo lo disfrutás?</legend>
                <label
                  ><input
                    v-model="fulfillment"
                    type="radio"
                    value="pickup"
                    @change="quote = null"
                  />
                  Retiro en el local</label
                ><label v-if="menu?.shipping_methods?.length"
                  ><input
                    v-model="fulfillment"
                    type="radio"
                    value="delivery"
                    @change="
                      quote = null;
                      shipping ||= menu.shipping_methods[0].id;
                    "
                  />
                  Envío a domicilio</label
                ><label v-if="fulfillment === 'delivery'"
                  >Zona de envío<select
                    v-model="shipping"
                    @change="quote = null"
                  >
                    <option
                      v-for="method in menu.shipping_methods"
                      :key="method.id"
                      :value="method.id"
                    >
                      {{ method.name }} · {{ money(method.price) }}
                    </option>
                  </select></label
                >
              </fieldset>
              <div class="buyer-total">
                <span>{{
                  quote ? "Total confirmado" : "Subtotal estimado"
                }}</span><strong>{{ money(quote?.total || estimated) }}</strong>
              </div>
              <p v-if="quote && fulfillment === 'delivery'" class="buyer-muted">
                Incluye {{ money(quote.shipping) }} de envío ·
                {{ quote.shipping_name }}
              </p>
              <button
                v-if="!quote"
                class="btn-primary"
                :disabled="busy"
                @click="review"
              >
                {{ busy ? "Consultando precios…" : "Revisar pedido" }}
              </button>
              <form v-if="quote" class="buyer-form" @submit.prevent="confirm">
                <fieldset :disabled="locked">
                  <legend>Tus datos para la entrega</legend>
                  <label
                    >Nombre<input
                      v-model="name"
                      name="buyer-name"
                      autocomplete="name"
                      required
                      minlength="2"
                      maxlength="100" /></label
                  ><label
                    >Teléfono<input
                      v-model="phone"
                      name="buyer-phone"
                      type="tel"
                      autocomplete="tel"
                      required
                      minlength="6"
                      maxlength="30" /></label
                  ><label
                    >Correo <span class="buyer-muted">(opcional)</span
                    ><input
                      v-model="email"
                      name="buyer-email"
                      type="email"
                      autocomplete="email"
                      maxlength="254" /></label
                  ><label v-if="fulfillment === 'delivery'"
                    >Dirección y referencias<textarea
                      v-model="address"
                      name="buyer-address"
                      autocomplete="street-address"
                      required
                      minlength="8"
                      maxlength="500"
                      rows="2"
                    /></label
                  ><label
                    >¿Cómo vas a pagar?<select v-model="payment">
                      <option value="at_local">Al retirar o recibir</option>
                      <option
                        v-if="menu?.payment_options?.mercadopago"
                        value="mercadopago"
                      >
                        Mercado Pago, después de la aceptación
                      </option>
                    </select></label
                  >
                </fieldset>
                <p class="buyer-muted">
                  El local confirmará la disponibilidad al aceptar tu pedido.
                  Tus datos se comparten con este restaurante para coordinar la
                  entrega.
                </p>
                <button class="btn-primary" :disabled="busy">
                  {{
                    busy
                      ? "Enviando pedido…"
                      : pendingBody
                        ? "Reintentar la misma confirmación"
                        : "Confirmar pedido · " + money(quote.total)
                  }}
                </button>
                <button
                  v-if="pendingBody"
                  type="button"
                  class="btn-secondary"
                  :disabled="busy"
                  @click="refreshReceipt()"
                >
                  Consultar si se recibió
                </button>
              </form>
            </template>
            <button type="button" class="btn-secondary" @click="panel = false">
              Seguir mirando la carta
            </button>
          </template>
        </section>
      </div></Transition
    >
  </Teleport>
</template>
<style scoped>
.buyer-cart-trigger {
  position: fixed;
  z-index: 25;
  bottom: 24px;
  right: 24px;
  display: flex;
  gap: 32px;
  align-items: center;
  min-height: 56px;
  padding: 16px 24px;
  box-shadow: 0 8px 32px #0003;
  border-radius: 12px;
}
.buyer-overlay {
  justify-content: flex-end;
  padding: 16px;
}
.buyer-dialog {
  width: min(100%, 540px);
  padding: 32px;
  max-height: calc(100dvh - 32px);
  overflow-y: auto;
  color: var(--text);
  background: var(--surface);
}
.buyer-sheet {
  height: calc(100dvh - 32px);
  margin: 0;
  border-radius: 20px;
}
.buyer-dialog h2 {
  padding-right: 28px;
  font-size: 28px;
  margin-bottom: 24px;
}
.buyer-dialog p {
  line-height: 1.6;
}
.buyer-form,
.buyer-form fieldset,
.buyer-fulfillment {
  display: grid;
  gap: 18px;
}
.buyer-form fieldset,
.buyer-fulfillment {
  border: 0;
  padding: 0;
  margin: 24px 0;
  min-width: 0;
}
.buyer-form legend,
.buyer-fulfillment legend {
  font-weight: 650;
  margin-bottom: 16px;
}
.buyer-form label,
.buyer-fulfillment label {
  display: grid;
  gap: 8px;
  font-size: 14px;
}
.buyer-form :is(input:not([type="checkbox"]), textarea, select),
.buyer-fulfillment select {
  border: 1px solid var(--control-border);
  background: var(--field);
  color: var(--text);
  border-radius: 8px;
  padding: 12px;
  width: 100%;
  min-height: 44px;
}
.buyer-form .buyer-extra,
.buyer-fulfillment label:has([type="radio"]) {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
}
.buyer-extra input,
.buyer-fulfillment input {
  accent-color: var(--accent);
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}
.buyer-extra span {
  margin-left: auto;
}
.buyer-line {
  padding: 18px 0;
  border-bottom: 1px solid var(--border);
  overflow-wrap: anywhere;
}
.buyer-line-heading,
.buyer-total {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.buyer-line small {
  display: block;
  color: var(--text-muted);
  margin-top: 6px;
}
.buyer-total {
  font-size: 20px;
  margin: 24px 0;
  font-variant-numeric: tabular-nums;
}
.buyer-muted {
  color: var(--text-muted);
  font-size: 13px;
}
.buyer-empty {
  padding: 48px 0;
}
.buyer-receipt-status {
  padding: 24px;
  border: 1px solid var(--border);
  border-left: 4px solid var(--accent);
  border-radius: 12px;
  background: var(--surface-raised);
}
.buyer-receipt-status h3 {
  margin: 12px 0;
  font-size: 24px;
}
.buyer-actions {
  display: grid;
  gap: 12px;
  margin: 24px 0;
}
.buyer-sheet > .btn-secondary {
  margin-top: 16px;
}
@media (max-width: 600px) {
  .buyer-cart-trigger {
    right: 16px;
    left: 16px;
    bottom: 16px;
    justify-content: space-between;
  }
  .buyer-overlay {
    padding: 0;
  }
  .buyer-sheet {
    height: 100dvh;
    max-height: 100dvh;
    border-radius: 0;
  }
  .buyer-dialog {
    padding: 28px 20px;
  }
}
</style>
