<script setup>
import { confirmAction } from "../services/confirmation.js";
import ProductOptionsPicker from "./ProductOptionsPicker.vue";
import {
  optionsComplete,
  optionGroups,
  selectedOptions,
} from "../composables/productOptions.js";
import { useRoute } from "vue-router";
import { shopper, loadShopper } from "../composables/useShopper.js";
import { computed, onBeforeUnmount, ref } from "vue";
import { useOrderStream } from "../composables/useOrderStream.js";
import { notifyOrderChange } from "../composables/useOrderSound.js";
import OrderSoundToggle from "./OrderSoundToggle.vue";
import { money, statusLabels } from "../composables/useWorkspace.js";
const props = defineProps({
  tenant: { type: String, required: true },
  menu: Object,
  orderId: String,
  repeatOrderId: String,
});
const emit = defineEmits(["closed"]);
function closePanel() {
  panel.value = false;
  if (props.orderId) emit("closed");
}
const editingLine = ref(null);
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
const rememberContact = ref(false);
const couponCode = ref(String(useRoute().query.cupon || ""));
const recoveryMissing = ref(false);
const postalCode = ref(""),
  zone = ref(""),
  repeatChanges = ref([]),
  repeatNotice = ref("");
const storageKey = "legui-buyer:" + props.tenant;
let access = null,
  pendingBody = null,
  alive = true;
const controllers = new Set();
const checkingSession = ref(true);
let legacyAccess = null;
try {
  const saved = JSON.parse(sessionStorage.getItem(storageKey) || "null");
  if (saved?.tenant === props.tenant && saved.expires > Date.now())
    legacyAccess = saved;
} catch {}

async function cancelPending() {
  if (
    busy.value ||
    !(await confirmAction(
      "El local todavía no confirmó tu pedido. ¿Querés cancelarlo?",
      "Cancelar pedido",
    ))
  )
    return;
  busy.value = true;
  error.value = "";
  try {
    const data = await request("/api/public/buyer-order/cancel", {}, true);
    notifyOrderChange(receipt.value, data);
    receipt.value = data;
  } catch (e) {
    error.value = e.message;
    if (e.status === 409) await refreshReceipt();
  } finally {
    busy.value = false;
  }
}
async function bootstrapBuyer(fresh = false) {
  checkingSession.value = true;
  if (props.orderId) {
    access = { key: "", paymentKey: crypto.randomUUID() };
    checkingSession.value = false;
    return true;
  }
  try {
    const run = async () => {
      const response = await fetch(
        `/api/public/menu/${props.tenant}/buyer-session`,
        {
          method: "POST",
          credentials: "same-origin",
          signal: AbortSignal.timeout(15000),
          headers: {
            "Content-Type": "application/json",
            ...(legacyAccess?.secret && !fresh
              ? { Authorization: `Bearer ${legacyAccess.secret}` }
              : {}),
          },
          body: JSON.stringify({
            ...(legacyAccess && !fresh ? { legacy_key: legacyAccess.key } : {}),
            ...(fresh ? { new: true } : {}),
          }),
        },
      );
      const data = await response.json();
      if (!response.ok)
        throw new Error(
          data.message || "No pudimos recuperar tu sesión de compra.",
        );
      return data;
    };
    const data = navigator.locks
      ? await navigator.locks.request(`legui-buyer:${props.tenant}`, run)
      : await run();
    if (!alive) return false;
    access = { key: data.creation_key, paymentKey: data.payment_key };
    legacyAccess = null;
    try {
      sessionStorage.removeItem(storageKey);
    } catch {}
    return true;
  } catch (e) {
    if (alive) error.value = e.message;
    return false;
  } finally {
    if (alive) checkingSession.value = false;
  }
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
          item.extras.reduce(
            (n, x) => n + cents(x.price) * BigInt(x.quantity || 1),
            0n,
          )) *
          BigInt(item.quantity),
      0n,
    ),
  ),
);
const locked = computed(
  () =>
    busy.value ||
    checkingSession.value ||
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
    ...(couponCode.value && shopper.profile
      ? { coupon_code: couponCode.value }
      : {}),
    items: cart.value.map((x) => ({
      product_id: x.id,
      quantity: x.quantity,
      extras: x.extras.map((e) => ({
        extra_id: e.id,
        quantity: e.quantity || 1,
      })),
      note: x.note,
    })),
    fulfillment: fulfillment.value,
    ...(fulfillment.value === "delivery"
      ? props.menu?.delivery_zones_enabled
        ? { delivery_zone_id: zone.value, postal_code: postalCode.value }
        : { shipping_method_id: shipping.value }
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
      credentials: "same-origin",
      signal: controller.signal,
      headers: {
        "Content-Type": "application/json",
        ...(authenticated
          ? {
              "X-Buyer-Tenant": props.tenant,
              ...(props.orderId ? { "X-Shopper-Order": props.orderId } : {}),
            }
          : {}),
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
const progressSteps = computed(() =>
  receipt.value?.fulfillment === "delivery"
    ? ["pending", "accepted", "preparing", "ready", "dispatched", "delivered"]
    : ["pending", "accepted", "preparing", "ready", "delivered"],
);
const canStartNew = computed(
  () =>
    !!receipt.value &&
    !unresolvedPayment.value &&
    (["canceled", "rejected"].includes(receipt.value.status) ||
      (receipt.value.status === "delivered" && !unpaid.value)),
);
function open(item) {
  if (item.available === false) {
    error.value =
      "Este producto no está disponible por ahora. Elegí otra opción de la carta.";
    panel.value = true;
    return;
  }
  if (
    !(props.menu?.ordering_ready ?? props.menu?.ordering_enabled) ||
    locked.value ||
    receipt.value
  ) {
    panel.value = true;
    return;
  }
  selected.value = item;
  editingLine.value = null;
  extras.value = [];
  quantity.value = 1;
  note.value = "";
  error.value = "";
}
const validOptions = computed(() =>
  optionsComplete(
    selected.value,
    extras.value,
    selected.value?.selection_extras || props.menu?.extras || [],
  ),
);
async function editLine(item, index) {
  if (locked.value || receipt.value) return;
  busy.value = true;
  error.value = "";
  try {
    const latest = await request(
      `/api/public/menu/${props.tenant}?product_id=${encodeURIComponent(item.id)}&limit=1`,
    );
    if (!alive || !panel.value) return;
    const current = latest.items?.find((x) => x.id === item.id);
    if (!current || !current.available)
      throw new Error(
        "Este producto ya no está disponible. Podés quitarlo y elegir otra opción de la carta.",
      );
    const kept = [];
    for (const group of optionGroups(current, latest.extras || [])) {
      let count = 0;
      for (const old of item.extras) {
        const quantity = old.quantity || 1;
        if (
          group.options.some((o) => o.id === old.id) &&
          count + quantity <= group.max
        ) {
          kept.push({ id: old.id, quantity });
          count += quantity;
        }
      }
    }
    panel.value = false;
    selected.value = { ...current, selection_extras: latest.extras || [] };
    editingLine.value = index;
    quantity.value = item.quantity;
    note.value = item.note || "";
    extras.value = kept;
    if (kept.length !== item.extras.length)
      error.value =
        "Algunas opciones cambiaron. Revisá las elecciones actuales antes de guardar.";
  } catch (e) {
    if (alive) error.value = e.message;
  } finally {
    if (alive) busy.value = false;
  }
}
function closeProduct() {
  const edited = editingLine.value !== null;
  selected.value = null;
  editingLine.value = null;
  if (edited) panel.value = true;
}
function add() {
  const edited = editingLine.value !== null;
  const previous =
    editingLine.value === null
      ? 0
      : cart.value[editingLine.value]?.quantity || 0;
  if (
    !selected.value ||
    !validOptions.value ||
    count.value - previous + quantity.value > 100 ||
    (editingLine.value === null && cart.value.length >= 30)
  ) {
    error.value = "Revisá las opciones y cantidades del producto.";
    return;
  }
  const line = {
    ...selected.value,
    quantity: quantity.value,
    extras: selectedOptions(
      selected.value,
      extras.value,
      selected.value?.selection_extras || props.menu?.extras || [],
    ),
    note: note.value.trim(),
  };
  if (editingLine.value === null) cart.value.push(line);
  else cart.value.splice(editingLine.value, 1, line);
  selected.value = null;
  editingLine.value = null;
  if (edited) panel.value = true;
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
    if (result.account_payment_key && access)
      access.paymentKey = result.account_payment_key;
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
    if (!access)
      throw new Error(
        "No hay sesión de compra. Usá Reintentar sesión antes de confirmar.",
      );
    if (!pendingBody)
      pendingBody = {
        ...basket(),
        quote_digest: quote.value.digest,
        ...(shopper.profile && rememberContact.value
          ? { remember_contact: true }
          : {}),
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
async function newOrder() {
  if (!canStartNew.value || busy.value) return;
  if (props.orderId) {
    window.location.assign("/catalogo/" + props.tenant);
    return;
  }
  if (!(await bootstrapBuyer(true))) return;
  receipt.value = null;
  recoveryMissing.value = true;
  quote.value = null;
  pendingBody = null;
  step.value = "cart";
  try {
    sessionStorage.removeItem(storageKey);
  } catch {}
  name.value = "";
  phone.value = "";
  email.value = "";
  address.value = "";
  if (shopper.profile) {
    name.value = shopper.profile.name;
    email.value = shopper.profile.email;
    phone.value = shopper.profile.phone || "";
    address.value = shopper.profile.delivery_address || "";
  }
  panel.value = false;
}
async function importRepeat() {
  if (!props.repeatOrderId || !alive) return;
  if (receipt.value) {
    if (canStartNew.value) await newOrder();
    else {
      panel.value = true;
      return;
    }
  }
  if (receipt.value || checkingSession.value || !access) return;
  busy.value = true;
  error.value = "";
  panel.value = true;
  try {
    const data = await request(
      `/api/shopper/orders/${encodeURIComponent(props.repeatOrderId)}/repeat`,
      {},
    );
    if (!alive) return;
    if (data.tenant_id !== props.tenant)
      throw new Error("El pedido corresponde a otro local.");
    cart.value = (data.items || []).map((x) => ({ ...x, id: x.product_id }));
    quote.value = null;
    pendingBody = null;
    fulfillment.value = "pickup";
    shipping.value = "";
    zone.value = "";
    postalCode.value = "";
    repeatChanges.value = data.changes || [];
    repeatNotice.value = data.notice;
    step.value = "cart";
  } catch (e) {
    if (alive) error.value = e.message;
  } finally {
    if (alive) busy.value = false;
  }
}
loadShopper()
  .then((p) => {
    if (p && alive) {
      if (!name.value) name.value = p.name;
      if (!email.value) email.value = p.email;
      if (!phone.value) phone.value = p.phone || "";
      if (!address.value) address.value = p.delivery_address || "";
    }
  })
  .catch(() => {});
bootstrapBuyer().then(async (ok) => {
  if (!ok || !alive) return;
  await refreshReceipt(true);
  if (alive && receipt.value) panel.value = true;
  if (props.repeatOrderId) await importRepeat();
});

const { live } = useOrderStream({
  path: "/api/public/buyer-order/events",
  token: () => "",
  tenant: () => props.tenant,
  order: () => props.orderId || "",
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
defineExpose({
  open,
  ready: computed(
    () =>
      !checkingSession.value &&
      !!access &&
      (!!receipt.value || recoveryMissing.value),
  ),
});
</script>
<template>
  <button
    v-if="
      !orderId && ((menu?.ordering_ready ?? menu?.ordering_enabled) || receipt)
    "
    class="buyer-cart-trigger btn-primary"
    @click="panel = true"
  >
    <span>{{ receipt ? "Ver mi pedido" : "Mi pedido" }}</span><strong>{{
      receipt ? "#" + receipt.id.slice(0, 8) : `${count} · ${money(estimated)}`
    }}</strong>
  </button>
  <Teleport to="body">
    <Transition name="modal"
      ><div v-if="selected" class="modal-overlay" @click.self="closeProduct">
        <section v-dialog-care class="modal-card buyer-dialog">
          <p class="eyebrow">A tu gusto</p>
          <h2>{{ selected.name }}</h2>
          <p>{{ selected.description }}</p>
          <form class="buyer-form" @submit.prevent="add">
            <ProductOptionsPicker
              v-model="extras"
              :product="selected"
              :extras="selected?.selection_extras || menu?.extras || []"
            />
            <label
              >Cantidad<input
                name="product-quantity"
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
            <button class="btn-primary" :disabled="!validOptions">
              {{ editingLine === null ? "Sumar al pedido" : "Guardar cambios" }}
            </button>
          </form>
          <button
            class="modal-close"
            aria-label="Cerrar producto"
            @click="closeProduct"
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
        @click.self="closePanel"
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
            @click="closePanel"
          >
            ×
          </button>
          <p v-if="error" role="alert" class="error-msg">{{ error }}</p>
          <p v-if="checkingSession" role="status">
            Recuperando tu pedido anterior…
          </p>
          <button
            v-if="!checkingSession && !access"
            type="button"
            class="btn-secondary"
            @click="bootstrapBuyer().then((ok) => ok && refreshReceipt(true))"
          >
            Reintentar sesión
          </button>
          <div
            v-if="!receipt && repeatNotice"
            class="inline-notice"
            role="status"
          >
            <strong>Volvé a disfrutarlo</strong>
            <p>{{ repeatNotice }}</p>
            <ul v-if="repeatChanges.length">
              <li v-for="(change, i) in repeatChanges" :key="i">
                {{ change }}
              </li>
            </ul>
          </div>
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
            <p v-if="receipt.ended_reason" class="inline-notice">
              {{ receipt.ended_reason }}
            </p>
            <p v-if="receipt.status === 'pending' && receipt.pending_deadline">
              El local puede confirmar hasta las
              {{
                new Date(receipt.pending_deadline).toLocaleTimeString("es-AR", {
                  hour: "2-digit",
                  minute: "2-digit",
                })
              }}. Si no confirma, el pedido se cancelará automáticamente.
            </p>
            <p
              v-if="
                receipt.prep_min &&
                !['canceled', 'rejected'].includes(receipt.status)
              "
            >
              Preparación estimada: {{ receipt.prep_min }}–{{
                receipt.prep_max
              }}
              min desde la confirmación.<template
                v-if="receipt.estimated_ready_at"
              >
                Listo aproximadamente a las
                {{
                  new Date(receipt.estimated_ready_at).toLocaleTimeString(
                    "es-AR",
                    { hour: "2-digit", minute: "2-digit" },
                  )
                }}.</template
              >
            </p>
            <button
              v-if="
                receipt.status === 'pending' &&
                Number(receipt.total_paid) === 0 &&
                !receipt.payment
              "
              class="btn-secondary"
              :disabled="busy"
              @click="cancelPending"
            >
              Cancelar pedido
            </button>
            <ol
              v-if="!['canceled', 'rejected'].includes(receipt.status)"
              class="buyer-progress"
              aria-label="Progreso del pedido"
            >
              <li
                v-for="(stage, index) in progressSteps"
                :key="stage"
                :class="{
                  done: index <= progressSteps.indexOf(receipt.status),
                }"
                :aria-current="stage === receipt.status ? 'step' : undefined"
              >
                <span>{{ index + 1 }}</span>{{ statusLabels[stage] }}
              </li>
            </ol>
            <p v-if="Number(receipt.discount) > 0" class="buyer-muted">
              Cupón aplicado: {{ money(receipt.discount) }} de descuento.
            </p>
            <div
              v-for="(item, index) in receipt.items"
              :key="index"
              class="buyer-line"
            >
              <strong>{{ item.quantity }} × {{ item.name }}</strong
              ><small v-for="x in item.extras" :key="x.name"
                >+ {{ x.quantity }} {{ x.group_name ? x.group_name + ": " : ""
                }}{{ x.name }}</small
              ><small v-if="item.note">{{ item.note }}</small>
            </div>
            <p>
              {{
                receipt.fulfillment === "delivery"
                  ? "Envío a domicilio"
                  : "Retiro en el local"
              }}
              · {{ receipt.location
              }}<template v-if="receipt.delivery_zone_name">
                · {{ receipt.delivery_zone_name }} · CP
                {{ receipt.delivery_postal_code }}</template
              >
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
              La confirmación se conserva en este navegador durante 24 horas.
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
              <button
                type="button"
                class="text-button"
                :disabled="locked"
                @click="editLine(item, index)"
              >
                Cambiar opciones
              </button>
              <small v-for="x in item.extras" :key="x.id"
                >+ {{ x.quantity || 1 }} {{ x.name }}</small
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
                ><label
                  v-if="
                    menu?.delivery_zones_enabled
                      ? menu?.delivery_zones?.length
                      : menu?.shipping_methods?.length
                  "
                  ><input
                    v-model="fulfillment"
                    type="radio"
                    value="delivery"
                    @change="
                      quote = null;
                      if (menu.delivery_zones_enabled)
                        zone ||= menu.delivery_zones[0]?.id;
                      else shipping ||= menu.shipping_methods[0].id;
                    "
                  />
                  Envío a domicilio</label
                ><label
                  v-if="
                    fulfillment === 'delivery' && !menu?.delivery_zones_enabled
                  "
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
                <template
                  v-if="
                    fulfillment === 'delivery' && menu?.delivery_zones_enabled
                  "
                >
                  <label
                    >Código postal<input
                      v-model="postalCode"
                      name="delivery-postal-code"
                      autocomplete="postal-code"
                      maxlength="12"
                      placeholder="1896 o B1896ABC"
                      @input="quote = null"
                  /></label>
                  <label
                    >Zona de entrega<select
                      v-model="zone"
                      @change="quote = null"
                    >
                      <option value="">Elegí tu zona</option>
                      <option
                        v-for="z in menu.delivery_zones"
                        :key="z.id"
                        :value="z.id"
                      >
                        {{ z.name }} · {{ money(z.price) }} · mínimo
                        {{ money(z.minimum) }}
                      </option>
                    </select></label
                  >
                  <p v-if="zone" class="buyer-muted">
                    {{
                      menu.delivery_zones.find((z) => z.id === zone)
                        ?.description
                    }}
                    · Códigos postales:
                    {{
                      menu.delivery_zones
                        .find((z) => z.id === zone)
                        ?.postal_codes?.join(", ")
                    }}
                  </p>
                </template>
              </fieldset>
              <label v-if="shopper.profile" class="buyer-coupon"
                >¿Tenés un cupón?<input
                  v-model="couponCode"
                  maxlength="32"
                  placeholder="Código de esta marca"
                  :disabled="locked"
                  @input="quote = null"
              /></label>
              <p v-else class="buyer-muted">
                Podés pedir sin cuenta.
                <router-link :to="`/mi-cuenta?local=${props.tenant}`"
                  >Ingresá para guardar tus compras y usar cupones.</router-link
                >
              </p>
              <p v-if="quote && Number(quote.discount) > 0" class="buyer-muted">
                Descuento aplicado: {{ money(quote.discount) }} sobre productos,
                sin extras ni envío.
              </p>
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
                :disabled="
                  busy ||
                  (fulfillment === 'delivery' &&
                    menu?.delivery_zones_enabled &&
                    (!zone || !postalCode.trim()))
                "
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
                  ><label v-if="shopper.profile" class="buyer-remember"
                    ><input v-model="rememberContact" type="checkbox" />Guardar
                    teléfono y dirección para próximas compras</label
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
                <p v-if="shopper.profile" class="buyer-muted">
                  Comprando como {{ shopper.profile.name }}. Este pedido se
                  vincula a tu cuenta.
                </p>
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
            <button type="button" class="btn-secondary" @click="closePanel">
              Seguir mirando la carta
            </button>
          </template>
        </section>
      </div></Transition
    >
  </Teleport>
</template>
<style scoped>
.buyer-progress {
  list-style: none;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin: 24px 0;
}
.buyer-progress li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.buyer-progress span {
  border: 1px solid var(--border);
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.buyer-progress .done {
  color: var(--text);
}
.buyer-progress .done span {
  background: var(--accent);
  color: var(--bg);
  border-color: var(--accent);
}
.buyer-form .buyer-remember {
  display: flex;
  align-items: center;
  gap: 12px;
  line-height: 1.5;
}
.buyer-remember input {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  accent-color: var(--accent);
}
.buyer-coupon {
  display: grid;
  gap: 8px;
  margin: 16px 0;
}
.buyer-coupon input {
  min-height: 44px;
  padding: 12px;
  border: 1px solid var(--control-border);
  border-radius: 8px;
  color: var(--text);
  background: var(--field);
}
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
.buyer-fulfillment :is(select, input:not([type="radio"])) {
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
.buyer-fulfillment input[type="radio"] {
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
