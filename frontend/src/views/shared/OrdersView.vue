<script setup>
import OrderSoundToggle from "../../components/OrderSoundToggle.vue";
import { useOrderStream } from "../../composables/useOrderStream.js";
import { playOrderSound } from "../../composables/useOrderSound.js";
import LoadingRows from "../../components/LoadingRows.vue";
import { computed, ref, watch } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import {
  useWorkspace,
  money,
  when,
  statusLabels,
} from "../../composables/useWorkspace.js";
const inbox = ref([]),
  pendingCount = ref(0);
const orders = ref([]),
  catalog = ref([]),
  customers = ref([]),
  offset = ref(0),
  catalogOffset = ref(0),
  catalogSearch = ref(""),
  selected = ref(null),
  detail = ref(null),
  payments = ref([]),
  cart = ref([]),
  product = ref(""),
  quantity = ref(1),
  customer = ref(""),
  customerSearch = ref(""),
  paymentAmount = ref(""),
  method = ref("cash"),
  reason = ref(""),
  share = ref(""),
  refundOriginal = ref(""),
  refundAmount = ref("");
const w = useWorkspace(async ({ request, tenant, current }) => {
  orders.value = [];
  catalog.value = [];
  if (!tenant) return;
  const [list, menu, people, pending] = await Promise.all([
    request(`/orders?limit=25&offset=${offset.value}`),
    request(
      `/pos/catalog?limit=25&offset=${catalogOffset.value}&search=${encodeURIComponent(catalogSearch.value)}`,
    ),
    request(
      `/crm/customers?limit=25&search=${encodeURIComponent(customerSearch.value)}`,
    ),
    request("/orders/inbox"),
  ]);
  if (current()) {
    orders.value = list.items || [];
    setInbox(pending, tenant);
    if (selected.value) await updateDetail(selected.value, tenant);
    catalog.value = menu.items || [];
    customers.value = people.items || [];
  }
});
let inboxTenant = "",
  known = null,
  dirty = false;
function setInbox(data, tenant) {
  if (inboxTenant !== tenant) {
    known = null;
    inboxTenant = tenant;
  }
  const items = data.items || [];
  if (known && items.some((x) => !known.has(x.id))) playOrderSound();
  known = new Set(items.map((x) => x.id));
  inbox.value = items;
  pendingCount.value = data.pending_count || 0;
}
async function updateDetail(id, tenant) {
  const [order, paid] = await Promise.all([
    w.request(`/orders/${id}`),
    w.request(`/orders/${id}/payments`),
  ]);
  if (w.tenant.value === tenant && selected.value === id) {
    detail.value = order;
    payments.value = paid.items || [];
  }
}
async function refreshInbox() {
  if (w.busy.value || w.loading.value) {
    dirty = true;
    return;
  }
  const tenant = w.tenant.value;
  try {
    const [list, pending] = await Promise.all([
      w.request(`/orders?limit=25&offset=${offset.value}`),
      w.request("/orders/inbox"),
    ]);
    if (w.tenant.value !== tenant) return;
    if (w.busy.value || w.loading.value) {
      dirty = true;
      return;
    }
    orders.value = list.items || [];
    setInbox(pending, tenant);
    if (selected.value) await updateDetail(selected.value, tenant);
  } catch (e) {
    w.error.value = e.message;
  }
}
const { live } = useOrderStream({
  path: "/api/orders/events",
  token: () => w.auth.state.token,
  tenant: () => w.tenant.value,
  headers: () => ({ "X-Tenant-ID": w.tenant.value }),
  enabled: () => !!w.tenant.value,
  refresh: refreshInbox,
});
watch([w.busy, w.loading], () => {
  if (dirty && !w.busy.value && !w.loading.value) {
    dirty = false;
    refreshInbox();
  }
});
const transitions = {
  pending: ["accepted", "rejected", "canceled"],
  accepted: ["preparing", "canceled"],
  preparing: ["ready", "canceled"],
  ready: ["dispatched", "delivered", "canceled"],
  dispatched: ["delivered", "canceled"],
};
const total = computed(() =>
  cart.value.reduce((sum, x) => sum + Number(x.price) * x.quantity, 0),
);
function add() {
  const item = catalog.value.find((x) => x.id === product.value);
  if (!item || quantity.value < 1) return;
  const existing = cart.value.find((x) => x.product_id === item.id);
  if (existing) existing.quantity += quantity.value;
  else
    cart.value.push({
      product_id: item.id,
      name: item.name,
      price: item.price,
      quantity: quantity.value,
    });
}
function create() {
  return w.act(async () => {
    await w.request("/orders", {
      channel: "counter",
      environment: "production",
      ...(customer.value ? { customer_id: customer.value } : {}),
      items: cart.value.map((x) => ({
        product_id: x.product_id,
        quantity: x.quantity,
      })),
    });
    cart.value = [];
    customer.value = "";
  }, "Pedido creado. Aceptalo para registrar la venta y descontar el stock.");
}
async function open(item) {
  w.error.value = "";
  selected.value = item.id;
  share.value = "";
  w.busy.value = true;
  try {
    [detail.value, payments.value] = await Promise.all([
      w.request(`/orders/${item.id}`),
      w.request(`/orders/${item.id}/payments`).then((x) => x.items || []),
    ]);
  } catch (e) {
    w.error.value = e.message;
  } finally {
    w.busy.value = false;
  }
}
function status(next) {
  return w.act(
    () =>
      w.request(
        `/orders/${detail.value.id}/status`,
        {
          status: next,
          expected_version: detail.value.version,
          ...(["canceled", "rejected"].includes(next)
            ? { reason: reason.value }
            : {}),
        },
        "PATCH",
      ),
    "Estado actualizado.",
  );
}
function refund() {
  return w.act(async () => {
    await w.request(`/orders/${detail.value.id}/payments`, {
      kind: "refund",
      amount: String(refundAmount.value),
      method:
        payments.value.find((x) => x.id === refundOriginal.value)?.method ||
        "other",
      original_payment_id: refundOriginal.value,
    });
    refundOriginal.value = "";
    refundAmount.value = "";
  }, "Devolución manual registrada.");
}
function pay() {
  return w.act(async () => {
    await w.request(`/orders/${detail.value.id}/payments`, {
      kind: "charge",
      amount: String(paymentAmount.value),
      method: method.value,
    });
    paymentAmount.value = "";
  }, "Cobro registrado.");
}
async function link() {
  w.busy.value = true;
  w.error.value = "";
  try {
    const data = await w.request(`/orders/${detail.value.id}/link`, {});
    share.value = window.location.origin + data.path;
  } catch (e) {
    w.error.value = e.message;
  } finally {
    w.busy.value = false;
  }
}
function revokeLink() {
  return w.act(
    () => w.request(`/orders/${detail.value.id}/link`, {}, "DELETE"),
    "Todos los enlaces del pedido fueron revocados.",
  );
}
watch(w.tenant, () => {
  selected.value = null;
  detail.value = null;
  payments.value = [];
  share.value = "";
  inbox.value = [];
  pendingCount.value = 0;
  known = null;
  offset.value = 0;
  catalogOffset.value = 0;
  cart.value = [];
  customer.value = "";
});
function page(n) {
  offset.value = n;
  w.refresh();
}
function menuPage(n) {
  catalogOffset.value = n;
  w.refresh();
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Caja y pedidos"
    description="Tomá pedidos, registrá cobros y acompañá cada entrega."
    ><div class="operation-actions">
      <router-link class="btn-secondary" to="/cash"
        >Turnos y arqueo de caja</router-link
      >
    </div>
    <section
      class="card operation-form live-inbox"
      aria-labelledby="inbox-title"
    >
      <div class="operation-actions">
        <h2 id="inbox-title">
          Por confirmar <span class="badge">{{ pendingCount }}</span>
        </h2>
        <OrderSoundToggle new-orders />
      </div>
      <p role="status">
        {{
          live
            ? "Pedidos en vivo"
            : "Actualización automática · reconectando el canal en vivo"
        }}
      </p>
      <p v-if="!inbox.length" class="no-data">
        No hay pedidos esperando confirmación.
      </p>
      <article v-for="item in inbox" :key="item.id" class="report-row">
        <div>
          <strong
            >#{{ item.id.slice(0, 8) }} ·
            {{ item.buyer_name || "Mostrador" }}</strong
          ><small class="inbox-deadline"
            >{{ money(item.total_due) }} · {{ when(item.created_at)
            }}<template v-if="item.pending_deadline">
              · Confirmar antes de {{ when(item.pending_deadline) }}</template
            ></small
          >
        </div>
        <button
          class="btn-primary"
          :disabled="w.busy.value"
          @click="open(item)"
        >
          Revisar y confirmar
        </button>
      </article>
      <small v-if="pendingCount > 100">Mostrando los 100 más antiguos.</small>
    </section>
    <form v-form-care class="card operation-form" @submit.prevent="create">
      <h2>Nuevo pedido de mostrador</h2>
      <div class="form-row">
        <label
          >Buscar en toda la carta<input
            v-model="catalogSearch"
            type="search"
            maxlength="100" /></label
        ><button
          type="button"
          class="btn-secondary"
          :disabled="w.busy.value"
          @click="
            catalogOffset = 0;
            w.refresh();
          "
        >
          Buscar producto
        </button>
      </div>
      <div class="form-row">
        <label
          >Hamburguesa / producto<select v-model="product">
            <option value="">Elegí de la carta</option>
            <option v-for="item in catalog" :key="item.id" :value="item.id">
              {{ item.name }} · {{ money(item.price) }}
            </option>
          </select></label
        ><label
          >Cantidad<input
            v-model.number="quantity"
            type="number"
            min="1"
            max="500" /></label
        ><button
          class="btn-secondary"
          type="button"
          :disabled="!product || w.busy.value"
          @click="add"
        >
          Sumar al pedido
        </button>
      </div>
      <ListPager
        :offset="catalogOffset"
        :count="catalog.length"
        :disabled="w.busy.value"
        @change="menuPage"
      />
      <div v-for="(line, i) in cart" :key="line.product_id" class="report-row">
        <span>{{ line.quantity }} × {{ line.name }}</span
        ><span
          >{{ money(Number(line.price) * line.quantity) }}
          <button
            type="button"
            class="text-button"
            :aria-label="`Quitar ${line.name}`"
            @click="cart.splice(i, 1)"
          >
            Quitar
          </button></span
        >
      </div>
      <p>
        Total estimado: <strong>{{ money(total) }}</strong>. El servidor
        confirma los precios al crear.
      </p>
      <div class="form-row">
        <label
          >Buscar cliente<input v-model="customerSearch" type="search" /></label
        ><button class="btn-secondary" type="button" @click="w.refresh">
          Buscar</button
        ><label
          >Cliente (opcional)<select v-model="customer">
            <option value="">Consumidor final</option>
            <option v-for="item in customers" :key="item.id" :value="item.id">
              {{ item.first_name }} {{ item.last_name }} · {{ item.email }}
            </option>
          </select></label
        >
      </div>
      <button
        class="btn-primary"
        :disabled="w.busy.value || !cart.length || !w.tenant.value"
      >
        Crear pedido
      </button>
    </form>
    <div class="card table-scroll">
      <table class="data-table">
        <thead>
          <tr>
            <th>Pedido</th>
            <th>Fecha</th>
            <th>Estado</th>
            <th>Total</th>
            <th>Cobrado</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <LoadingRows v-if="w.loading.value" :columns="6" />
          <tr v-show="!w.loading.value" v-for="item in orders" :key="item.id">
            <td>
              #{{ item.id.slice(0, 8)
              }}<small v-if="item.channel === 'web'"> · Online</small>
            </td>
            <td>{{ when(item.created_at) }}</td>
            <td>{{ statusLabels[item.status] }}</td>
            <td>{{ money(item.total_due, item.currency) }}</td>
            <td>{{ money(item.total_paid, item.currency) }}</td>
            <td>
              <button class="btn-secondary" @click="open(item)">
                Ver pedido
              </button>
            </td>
          </tr>
          <tr v-show="!w.loading.value" v-if="!orders.length">
            <td colspan="6">Los pedidos de este local aparecerán acá.</td>
          </tr>
        </tbody>
      </table>
    </div>
    <ListPager
      :offset="offset"
      :count="orders.length"
      :disabled="w.busy.value"
      @change="page"
    />
    <section v-if="detail" class="card operation-form">
      <h2>
        Pedido #{{ detail.id.slice(0, 8) }} · {{ statusLabels[detail.status] }}
      </h2>
      <div class="operation-actions">
        <router-link
          class="btn-secondary"
          target="_blank"
          rel="noopener"
          :to="`/orders/${detail.id}/print?tenant=${w.tenant.value}&kind=receipt`"
          >Imprimir comprobante</router-link
        ><router-link
          class="btn-secondary"
          target="_blank"
          rel="noopener"
          :to="`/orders/${detail.id}/print?tenant=${w.tenant.value}&kind=kitchen`"
          >Imprimir comanda</router-link
        >
      </div>
      <section v-if="detail.buyer_name" aria-label="Datos del comprador">
        <h3>{{ detail.buyer_name }}</h3>
        <p>
          Teléfono: {{ detail.buyer_phone
          }}<span v-if="detail.buyer_email"> · {{ detail.buyer_email }}</span>
        </p>
        <p>
          {{
            detail.fulfillment === "delivery"
              ? "Envío a domicilio"
              : "Retiro en el local"
          }}<span v-if="detail.shipping_address">
            · {{ detail.shipping_address }}</span
          >
        </p>
        <p>
          Pago elegido:
          {{
            detail.buyer_payment_method === "mercadopago"
              ? "Mercado Pago después de la aceptación"
              : "Al retirar o recibir"
          }}
        </p>
      </section>
      <div v-for="line in detail.items" :key="line.id" class="report-row">
        <span
          >{{ line.quantity }} × {{ line.product_name
          }}<small v-for="extra in line.extras" :key="extra.id">
            + {{ extra.quantity }} {{ extra.extra_name }}</small
          ><small v-if="line.preparation_note"
            >Preferencias: {{ line.preparation_note }}</small
          ></span
        ><span>{{ money(line.final_price_unit, detail.currency) }} c/u</span>
      </div>
      <p>
        Total {{ money(detail.total_due, detail.currency) }} · Cobrado
        {{ money(detail.total_paid, detail.currency) }}
      </p>
      <label v-if="transitions[detail.status]?.length"
        >Motivo para rechazar o cancelar<input v-model="reason" maxlength="500"
      /></label>
      <div class="operation-actions">
        <button
          v-for="next in transitions[detail.status]"
          :key="next"
          class="btn-secondary"
          :disabled="
            w.busy.value ||
            (['canceled', 'rejected'].includes(next) &&
              reason.trim().length < 3)
          "
          @click="status(next)"
        >
          {{ statusLabels[next] }}
        </button>
      </div>
      <form
        v-form-care
        v-if="!['canceled', 'rejected'].includes(detail.status)"
        class="operation-form"
        @submit.prevent="pay"
      >
        <h3>Registrar un cobro manual</h3>
        <p>
          Sólo registrá dinero que ya recibiste. Para cobrar online, usá Mercado
          Pago.
        </p>
        <div class="form-row">
          <label
            >Importe<input
              v-model="paymentAmount"
              type="text"
              inputmode="decimal"
              pattern="[0-9]+([.][0-9]{1,2})?"
              required /></label
          ><label
            >Medio<select v-model="method">
              <option value="cash">Efectivo</option>
              <option value="transfer">Transferencia</option>
              <option value="card">Tarjeta</option>
              <option value="other">Otro</option>
            </select></label
          >
        </div>
        <button class="btn-primary" :disabled="w.busy.value">
          Registrar cobro recibido
        </button>
      </form>
      <div v-for="p in payments" :key="p.id" class="report-row">
        <span
          >{{ p.kind === "refund" ? "Devolución" : "Cobro" }} · {{ p.method }} ·
          {{ p.status }}</span
        ><strong>{{ money(p.amount, p.currency) }}</strong>
      </div>
      <form
        v-form-care
        v-if="
          payments.some(
            (x) =>
              x.kind === 'charge' &&
              x.status === 'succeeded' &&
              x.provider !== 'mercadopago',
          )
        "
        class="operation-form"
        @submit.prevent="refund"
      >
        <h3>Registrar una devolución manual</h3>
        <p>
          Registrá el dinero que ya devolviste. Las devoluciones de Mercado Pago
          se realizan desde Cobros.
        </p>
        <div class="form-row">
          <label
            >Cobro original<select v-model="refundOriginal" required>
              <option value="">Elegí el cobro</option>
              <option
                v-for="p in payments.filter(
                  (x) =>
                    x.kind === 'charge' &&
                    x.status === 'succeeded' &&
                    x.provider !== 'mercadopago',
                )"
                :key="p.id"
                :value="p.id"
              >
                {{ money(p.amount, p.currency) }} ? {{ p.method }}
              </option>
            </select></label
          ><label
            >Importe devuelto<input
              v-model="refundAmount"
              type="text"
              inputmode="decimal"
              pattern="[0-9]+([.][0-9]{1,2})?"
              required
          /></label>
        </div>
        <button class="btn-danger" :disabled="w.busy.value">
          Registrar devolución realizada
        </button>
      </form>
      <div class="operation-actions">
        <router-link class="btn-secondary" to="/payments"
          >Cobrar con Mercado Pago</router-link
        ><button class="btn-secondary" :disabled="w.busy.value" @click="link">
          Crear seguimiento para el cliente</button
        ><button
          class="text-button"
          :disabled="w.busy.value"
          @click="revokeLink"
        >
          Revocar seguimientos
        </button>
      </div>
      <label v-if="share"
        >Enlace privado, válido por 7 días<input
          :value="share"
          readonly
          @focus="$event.target.select()"
      /></label>
    </section>
  </WorkspaceFrame>
</template>

<style scoped>
.live-inbox .operation-actions {
  justify-content: space-between;
}
.inbox-deadline {
  display: block;
  margin-top: 0.4rem;
  color: var(--text-muted);
}
.live-inbox .report-row {
  gap: 1rem;
  flex-wrap: wrap;
}
</style>
