<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import { computed, onMounted, ref, watch } from "vue";
import DashboardShell from "../../components/DashboardShell.vue";
import StableContent from "../../components/StableContent.vue";
import { useAuth } from "../../composables/useAuth.js";
import { useMembership } from "../../composables/useMembership.js";
import { apiFetch, getJSON } from "../../services/api.js";
const auth = useAuth();
const membership = useMembership(),
  terminals = ref([]);
const suspended = ref(false);
const config = ref({}),
  connections = ref([]),
  transactions = ref([]),
  reviews = ref([]),
  tenants = ref([]),
  plans = ref([]);
const billing = ref({ invoices: [], subscriptions: [] }),
  brand = ref(auth.user.value?.brand_id || ""),
  tenant = ref(auth.tenantId.value || ""),
  email = ref(auth.user.value?.email || ""),
  plan = ref("");
const link = ref(""),
  error = ref(""),
  notice = ref(""),
  busy = ref(false),
  loading = ref(true);
const tab = ref("transactions"),
  reviewNote = ref(""),
  reviewSelection = ref(""),
  resolution = ref("reprocess");
const visibleTenants = computed(() =>
  tenants.value.filter((item) => !brand.value || item.brand_id === brand.value),
);
const openReviews = computed(() =>
  reviews.value.filter((item) => !item.resolved_at),
);
const labels = {
  approved: "Aprobado",
  pending: "Pendiente",
  in_process: "En proceso",
  rejected: "Rechazado",
  cancelled: "Cancelado",
  refunded: "Devuelto",
  charged_back: "Contracargo",
  created: "Link listo",
  prepared: "Preparado",
  submitting: "Enviando",
  review: "En revisión",
  failed: "No confirmado",
  connected: "Conectada",
  disconnected: "Desconectada",
  connecting: "Conectando",
  needs_reconnect: "Requiere reconexión",
  incomplete: "Pendiente de pago",
  trialing: "Prueba",
  active: "Activa",
  past_due: "Pago vencido",
  paused: "Pausada",
  canceled: "Cancelada",
};
const reviewLabels = {
  AMOUNT_MISMATCH: "El importe no coincide",
  CHARGEBACK: "Contracargo",
  ALLOCATION_REQUIRES_REVIEW: "El saldo necesita revisión",
  UNOWNED_REFERENCE: "Referencia sin pedido asociado",
  INVALID_PROVIDER_PROOF: "No se pudo verificar el pago",
  REFUND_TOTAL_MISMATCH: "La devolución necesita conciliación",
  CONNECTION_CHANGED: "Cambió la cuenta de cobro",
  OPERATION_OUTCOME_UNKNOWN: "Resultado pendiente de confirmación",
  OPEN_REVIEW: "Hay una revisión pendiente",
};
const money = (value, currency = "ARS") =>
  value === null || value === undefined
    ? "Por confirmar"
    : new Intl.NumberFormat("es-AR", { style: "currency", currency }).format(
        Number(value),
      );
const when = (value) =>
  value ? new Date(value).toLocaleString("es-AR") : "Por confirmar";
const pendingKeys = new Map();
const purchasedLocations = ref(1);
const selectedPrice = computed(() =>
  plans.value.find((item) => item.id === plan.value),
);
const membershipTotal = computed(
  () =>
    Number(selectedPrice.value?.amount || 0) +
    (selectedPrice.value?.membership_tier === "premium"
      ? Math.max(0, Number(purchasedLocations.value) - 1) *
        Number(selectedPrice.value.extra_location_amount || 0)
      : 0),
);
watch(plan, () => {
  purchasedLocations.value =
    selectedPrice.value?.membership_tier === "premium"
      ? Math.max(
          1,
          visibleTenants.value.filter((item) => item.active !== false).length,
        )
      : 1;
});
async function mutate(path, body = {}, method = "POST") {
  const operation = `${method}:${path}:${JSON.stringify(body)}`;
  if (!pendingKeys.has(operation))
    pendingKeys.set(operation, crypto.randomUUID());
  const response = await apiFetch(path, {
    method,
    headers: {
      "Idempotency-Key": pendingKeys.get(operation),
      ...(tenant.value ? { "X-Tenant-ID": tenant.value } : {}),
    },
    body: method === "DELETE" ? undefined : JSON.stringify(body),
  });
  const result = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new Error(result.message || "No se pudo completar la operación");
  pendingKeys.delete(operation);
  return result;
}
async function execute(fn) {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  notice.value = "";
  try {
    await fn();
    await refreshData();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
async function refreshData() {
  await membership.refresh();
  const state = await getJSON("/business");
  suspended.value =
    !!state.items?.[0]?.suspended_at ||
    membership.access.value?.operational === false;
  if (suspended.value) tab.value = "membership";
  const [cfg, planList, account] = await Promise.all([
    getJSON("/payments/mercadopago/config"),
    getJSON("/billing/plans"),
    getJSON("/billing/my-account"),
  ]);
  config.value = cfg;
  plans.value = planList.items || [];
  billing.value = account;
  if (!plan.value) plan.value = plans.value[0]?.id || "";
  terminals.value = (await getJSON("/billing/terminals")).items || [];
  if (!suspended.value) {
    const rows = await Promise.all([
      getJSON("/integrations/mercadopago/connections"),
      getJSON("/payments/mercadopago/transactions"),
      getJSON("/payments/mercadopago/reviews"),
    ]);
    [connections.value, transactions.value, reviews.value] = rows.map(
      (item) => item.items || [],
    );
  }
}
async function connect() {
  await execute(async () => {
    const created = await mutate("/integrations/mercadopago/connections", {
      brand_id: brand.value,
      tenant_id: tenant.value,
      environment: config.value.environment,
    });
    const result = await mutate(
      `/integrations/mercadopago/connections/${created.id}/authorize`,
    );
    const url = new URL(result.authorization_url);
    if (url.protocol !== "https:" || url.hostname !== "auth.mercadopago.com")
      throw new Error("No se pudo verificar el enlace de autorización");
    window.location.assign(url.href);
  });
}
function showCheckout(result) {
  if (result.checkout_url) {
    link.value = result.checkout_url;
    notice.value = "Link listo para abonar tu membresía.";
  } else
    notice.value =
      "La operación quedó pendiente de confirmación. Consultá su estado antes de generar otro cobro.";
}
function payInvoice(item) {
  return execute(async () =>
    showCheckout(await mutate(`/billing/invoices/${item.id}/checkout`, {})),
  );
}
function startSubscription() {
  return execute(async () => {
    const price = plans.value.find((item) => item.id === plan.value);
    const result = await mutate("/billing/subscribe", {
      price_id: plan.value,
      brand_id: brand.value,
      tenant_id: price?.billing_unit === "tenant" ? tenant.value : "",
      payer_email: email.value,
      purchased_locations: Number(purchasedLocations.value),
    });
    showCheckout(
      await mutate(
        `/billing/subscriptions/${result.subscription_id}/checkout`,
        { payer_email: email.value },
      ),
    );
  });
}
function releaseTerminal(item) {
  return execute(async () => {
    await mutate(`/billing/terminals/${item.id}`, {}, "DELETE");
    notice.value = "Terminal desconectada. El cupo está disponible.";
  });
}
function continueSubscription(item) {
  return execute(async () =>
    showCheckout(
      await mutate(`/billing/subscriptions/${item.id}/checkout`, {
        payer_email: email.value,
      }),
    ),
  );
}
function cancelSubscription(item) {
  return execute(async () => {
    const result = await mutate(
      `/billing/subscriptions/${item.id}/cancel-renewal`,
    );
    notice.value =
      result.state === "completed"
        ? "La renovación quedó cancelada. Conservás el acceso durante el período ya pagado."
        : "La cancelación necesita confirmación. Revisá su estado.";
  });
}
const refundPayment = ref(""),
  refundAmount = ref("");
function refund() {
  return execute(async () => {
    const result = await mutate(
      `/payments/mercadopago/transactions/${refundPayment.value}/refunds`,
      { amount: String(refundAmount.value) },
    );
    notice.value =
      result.state === "completed"
        ? "Devolución confirmada y registrada."
        : "Devolución pendiente de conciliación.";
    refundPayment.value = "";
    refundAmount.value = "";
  });
}
function resolveReview() {
  return execute(async () => {
    await mutate(
      `/payments/mercadopago/reviews/${reviewSelection.value}/resolve`,
      { resolution: resolution.value, note: reviewNote.value },
    );
    reviewSelection.value = "";
    reviewNote.value = "";
    notice.value =
      "Resolución registrada. Reprocesar vuelve a verificar el proveedor; no acredita dinero manualmente.";
  });
}
function reconcile(item) {
  return execute(async () => {
    await mutate("/payments/mercadopago/reconcile", { connection_id: item.id });
    notice.value =
      "Conciliación consultada. Actualizá para ver nuevos movimientos.";
  });
}
function disconnect(item) {
  return execute(async () => {
    await mutate(
      `/integrations/mercadopago/connections/${item.id}`,
      {},
      "DELETE",
    );
    notice.value = "Cuenta desconectada.";
  });
}
watch(brand, () => {
  if (!visibleTenants.value.some((item) => item.id === tenant.value))
    tenant.value = visibleTenants.value[0]?.id || "";
});
watch([brand, tenant], () => {
  if (!loading.value) execute(async () => {});
});
onMounted(async () => {
  try {
    tenants.value = ((await getJSON("/tenants")) || []).map((item) => ({
      ...item,
      brand_id: item.brand_id || item.brand?.id,
    }));
    if (!tenant.value) tenant.value = visibleTenants.value[0]?.id || "";
    await refreshData();
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <DashboardShell role="admin"
    ><section class="payments-page">
      <div class="section-header">
        <div class="section-title">
          <h2>Finanzas del negocio</h2>
          <p>
            Consultá los ingresos de tus locales, conectá tu cuenta y administrá
            tu plan.
          </p>
        </div>
        <button
          class="btn-secondary"
          :disabled="busy || loading"
          @click="execute(async () => {})"
        >
          Actualizar
        </button>
      </div>
      <StableContent
        :loading="loading"
        :busy="busy"
        label="Cargando cobros y membresías…"
      >
        <div class="payment-intro">
          <div class="card setup-notice">
            <strong v-if="loading"
              >Consultando la configuración de cobros…</strong
            ><strong v-else-if="!config.enabled"
              >Mercado Pago todavía no está habilitado.</strong
            ><strong v-else>Gestioná tus cobros en un mismo lugar.</strong>
            <p v-if="!config.enabled">
              La administración de LeguiBurger debe completar la configuración
              de cobros. Tus registros existentes siguen disponibles.
            </p>
            <p v-else>
              Revisá los movimientos de tus locales y la cuenta que recibe tus
              ventas.
            </p>
          </div>
          <p class="test-mode">
            {{
              loading
                ? "Consultando el entorno de cobros…"
                : config.environment === "sandbox"
                  ? "Modo de prueba · Estos movimientos no acreditan dinero real."
                  : "Movimientos del entorno configurado para tu negocio."
            }}
          </p>
        </div>
        <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
        <p v-if="notice" class="status-notice" role="status">{{ notice }}</p>
        <p v-if="suspended" class="error-msg">
          Tu restaurante está suspendido. Podés consultar tu membresía y
          contactar a LeguiBurger para reactivar la operación.
        </p>
        <div v-if="!suspended" class="payment-filters">
          <div class="input-group">
            <label for="payments-tenant">Local</label
            ><select id="payments-tenant" v-model="tenant">
              <option value="">Toda la marca</option>
              <option
                v-for="item in visibleTenants"
                :key="item.id"
                :value="item.id"
              >
                {{ item.subdomain }}
              </option>
            </select>
          </div>
        </div>
        <nav class="payment-tabs" aria-label="Secciones de cobros">
          <button
            v-for="item in suspended
              ? ['membership']
              : ['transactions', 'connections', 'membership', 'reviews']"
            :key="item"
            class="btn-secondary"
            :aria-pressed="tab === item"
            @click="tab = item"
          >
            {{
              {
                transactions: "Movimientos",
                connections: "Cuentas conectadas",
                membership: "Mi membresía",
                reviews: `Revisiones (${openReviews.length})`,
              }[item]
            }}
          </button>
        </nav>
        <div v-if="link" class="card share-checkout">
          <label for="checkout-link">Link de pago</label
          ><input
            id="checkout-link"
            :value="link"
            readonly
            @focus="$event.target.select()"
          /><a :href="link" target="_blank" rel="noopener noreferrer"
            >Abrir checkout</a
          >
        </div>
        <template v-if="tab === 'transactions'"
          ><div class="card payment-table">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Pago</th>
                  <th>Fecha</th>
                  <th>Estado</th>
                  <th>Bruto</th>
                  <th>Comisiones</th>
                  <th>Neto informado</th>
                  <th>Disponibilidad</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <LoadingRows v-if="loading" :columns="8" />
                <tr
                  v-show="!loading"
                  v-for="item in transactions"
                  :key="item.id"
                >
                  <td>
                    {{ item.payment_id
                    }}<small
                      >{{
                        item.scope === "platform" ? "Membresía" : "Pedido"
                      }}
                      · {{ item.environment }}</small
                    >
                  </td>
                  <td>{{ when(item.paid_at) }}</td>
                  <td>
                    {{ labels[item.provider_status] || item.provider_status
                    }}<small v-if="Number(item.refunded_amount) > 0"
                      >Devuelto:
                      {{ money(item.refunded_amount, item.currency) }}</small
                    ><small v-if="item.under_review">En revisión</small
                    ><small v-else-if="!item.allocated"
                      >Sin aplicar al saldo</small
                    >
                  </td>
                  <td>{{ money(item.amount, item.currency) }}</td>
                  <td>{{ money(item.fee_amount, item.currency) }}</td>
                  <td>{{ money(item.net_received, item.currency) }}</td>
                  <td>{{ when(item.available_at) }}</td>
                  <td>
                    <button
                      v-if="
                        item.allocated &&
                        !item.under_review &&
                        item.scope === 'business' &&
                        Number(item.amount) > Number(item.refunded_amount)
                      "
                      class="btn-secondary"
                      @click="
                        refundPayment = item.id;
                        refundAmount = '';
                      "
                    >
                      Devolver
                    </button>
                  </td>
                </tr>
                <tr v-show="!loading" v-if="!transactions.length">
                  <td colspan="8">Todavía no hay pagos verificados.</td>
                </tr>
              </tbody>
            </table>
          </div>
          <form
            v-form-care
            v-if="refundPayment"
            class="card action-form"
            @submit.prevent="refund"
          >
            <h3>Devolución parcial o total</h3>
            <p>
              El importe se devuelve desde la cuenta receptora original y se
              registra después de confirmarlo.
            </p>
            <div class="input-group">
              <label for="refund-amount">Importe a devolver</label
              ><input
                id="refund-amount"
                v-model="refundAmount"
                type="text"
                inputmode="decimal"
                pattern="[0-9]+([.][0-9]{1,2})?"
                required
              />
            </div>
            <button class="btn-danger" :disabled="busy || !config.enabled">
              Confirmar devolución</button
            ><button
              class="btn-secondary"
              type="button"
              @click="refundPayment = ''"
            >
              Cancelar
            </button>
          </form></template
        >
        <template v-if="tab === 'connections'"
          ><div class="card action-form">
            <h3>Tu cuenta de cobro</h3>
            <p>
              Conectá Mercado Pago para que el dinero de los pedidos llegue a tu
              negocio. Elegí un local o toda la marca antes de continuar.
            </p>
            <button
              class="btn-primary"
              :disabled="busy || !config.oauth_configured || !brand"
              @click="connect"
            >
              Conectar Mercado Pago
            </button>
          </div>
          <div
            v-for="item in connections"
            :key="item.id"
            class="card connection-row"
          >
            <div>
              <strong>{{
                item.scope === "platform"
                  ? "Cuenta de LeguiBurger"
                  : item.tenant_id
                    ? "Cuenta de local"
                    : "Cuenta de marca"
              }}</strong>
              <p>
                {{ labels[item.state] || item.state }} · {{ item.environment
                }}<span v-if="item.collector_id">
                  · Cuenta {{ item.collector_id }}</span
                >
              </p>
            </div>
            <div>
              <button
                v-if="item.state === 'connected'"
                class="btn-secondary"
                :disabled="busy || !config.enabled"
                @click="reconcile(item)"
              >
                Conciliar</button
              ><button
                v-if="
                  item.scope === 'business' && item.state !== 'disconnected'
                "
                class="btn-secondary"
                :disabled="busy"
                @click="disconnect(item)"
              >
                Desconectar
              </button>
            </div>
          </div></template
        >
        <template v-if="tab === 'membership'"
          ><form
            v-form-care
            class="card action-form"
            @submit.prevent="startSubscription"
          >
            <h3>Elegí tu membresía</h3>
            <p>
              La membresía se activa cuando Mercado Pago confirma el cobro. La
              prueba gratuita dura siete días y no se renueva.
            </p>
            <div class="input-group">
              <label for="membership-plan">Plan</label
              ><select id="membership-plan" v-model="plan" required>
                <option v-for="item in plans" :key="item.id" :value="item.id">
                  {{ item.name }} · {{ money(item.amount, item.currency) }} cada
                  {{ item.interval_months }} mes(es) ·
                  {{
                    item.billing_unit === "tenant" ? "por local" : "por marca"
                  }}
                </option>
              </select>
            </div>
            <div
              v-if="selectedPrice?.membership_tier === 'premium'"
              class="input-group"
            >
              <label for="membership-locations">Locales a contratar</label>
              <input
                id="membership-locations"
                v-model="purchasedLocations"
                type="number"
                min="1"
                max="10000"
                step="1"
                required
              />
              <small
                >Un local incluido. Cada adicional suma
                {{
                  money(
                    selectedPrice.extra_location_amount,
                    selectedPrice.currency,
                  )
                }}
                al mes. Hasta 5 terminales por local.</small
              >
            </div>
            <p v-else-if="selectedPrice?.membership_tier === 'standard'">
              Incluye 1 local y hasta 2 terminales.
            </p>
            <p v-if="selectedPrice" class="membership-total" aria-live="polite">
              Total:
              <strong>{{
                money(membershipTotal, selectedPrice.currency)
              }}</strong>
              {{
                selectedPrice.interval_months === 1
                  ? "por mes"
                  : `cada ${selectedPrice.interval_months} meses`
              }}. El importe se confirma al continuar.
            </p>
            <div class="input-group">
              <label for="membership-email">Correo del pagador</label
              ><input
                id="membership-email"
                v-model="email"
                type="email"
                required
              />
            </div>
            <button
              class="btn-primary"
              :disabled="busy || !config.platform_configured || !plan"
            >
              Continuar con Mercado Pago
            </button>
            <p v-if="!plans.length">Todavía no hay planes publicados.</p>
          </form>
          <section class="card action-form">
            <h3>Terminales conectadas</h3>
            <p>
              Las pestañas de un mismo navegador comparten sesión. Cerrar sesión
              libera la terminal; tras 30 minutos sin actividad, también queda
              libre.
            </p>
            <div
              v-for="item in terminals"
              :key="item.id"
              class="connection-row"
            >
              <div>
                <strong>{{ item.location }}</strong>
                <p>
                  {{ item.first_name }} {{ item.last_name }} · Última actividad
                  {{ when(item.last_seen_at) }}
                </p>
                <small>{{ item.user_agent }}</small>
              </div>
              <button
                type="button"
                class="btn-secondary"
                :disabled="busy"
                @click="releaseTerminal(item)"
              >
                Desconectar terminal
              </button>
            </div>
            <p v-if="!terminals.length">
              No hay terminales operativas conectadas.
            </p>
          </section>
          <div
            v-for="item in billing.subscriptions || []"
            :key="item.id"
            class="card connection-row"
          >
            <div>
              <strong>{{ labels[item.status] || item.status }}</strong>
              <p>
                {{
                  money(
                    Number(item.contracted_amount) * item.quantity,
                    item.currency,
                  )
                }}
                · Período hasta {{ when(item.current_period_end) }}
              </p>
              <small v-if="item.purchased_locations > 1"
                >{{ item.purchased_locations }} locales contratados · incluye el
                cargo de los adicionales.</small
              >
              <small v-if="item.cancel_at_period_end"
                >La renovación está cancelada.</small
              >
            </div>
            <button
              v-if="!item.provider_reference && item.status !== 'canceled'"
              class="btn-primary"
              :disabled="busy || !config.platform_configured"
              @click="continueSubscription(item)"
            >
              Completar contratación</button
            ><button
              v-else-if="
                item.provider_reference &&
                item.status !== 'canceled' &&
                !item.cancel_at_period_end
              "
              class="btn-secondary"
              :disabled="busy || !config.platform_configured"
              @click="cancelSubscription(item)"
            >
              Cancelar renovación
            </button>
          </div>
          <div class="card payment-table">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Factura de membresía</th>
                  <th>Importe</th>
                  <th>Saldo</th>
                  <th>Estado</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <LoadingRows v-if="loading" :columns="5" />
                <tr
                  v-show="!loading"
                  v-for="item in billing.invoices || []"
                  :key="item.id"
                >
                  <td>{{ item.description || item.id.slice(0, 8) }}</td>
                  <td>{{ money(item.amount_total, item.currency) }}</td>
                  <td>{{ money(item.balance_due, item.currency) }}</td>
                  <td>{{ item.status }}</td>
                  <td>
                    <button
                      v-if="
                        item.status === 'open' && Number(item.balance_due) > 0
                      "
                      class="btn-primary"
                      :disabled="busy || !config.platform_configured"
                      @click="payInvoice(item)"
                    >
                      Pagar
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
        <template v-if="tab === 'reviews'"
          ><div class="card payment-table">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Recurso</th>
                  <th>Motivo</th>
                  <th>Última observación</th>
                  <th>Estado</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <LoadingRows v-if="loading" :columns="5" />
                <tr v-show="!loading" v-for="item in reviews" :key="item.id">
                  <td>{{ item.resource_id }}</td>
                  <td>
                    {{
                      reviewLabels[item.reason] || "Requiere revisión del cobro"
                    }}
                  </td>
                  <td>{{ when(item.last_seen_at) }}</td>
                  <td>{{ item.resolved_at ? "Resuelta" : "Pendiente" }}</td>
                  <td>
                    <button
                      v-if="!item.resolved_at"
                      class="btn-secondary"
                      @click="reviewSelection = item.id"
                    >
                      Revisar
                    </button>
                  </td>
                </tr>
                <tr v-show="!loading" v-if="!reviews.length">
                  <td colspan="5">No hay inconsistencias registradas.</td>
                </tr>
              </tbody>
            </table>
          </div>
          <form
            v-form-care
            v-if="reviewSelection"
            class="card action-form"
            @submit.prevent="resolveReview"
          >
            <h3>Registrar resolución</h3>
            <div class="input-group">
              <label for="review-resolution">Acción</label
              ><select id="review-resolution" v-model="resolution">
                <option value="reprocess">
                  Volver a verificar Mercado Pago
                </option>
                <option value="acknowledged">
                  Cerrar revisión con una nota
                </option>
              </select>
            </div>
            <div class="input-group">
              <label for="review-note">Motivo de la resolución</label
              ><textarea
                id="review-note"
                v-model="reviewNote"
                minlength="10"
                maxlength="500"
                required
              />
            </div>
            <button class="btn-primary" :disabled="busy">
              Guardar resolución
            </button>
          </form></template
        >
      </StableContent>
    </section></DashboardShell
  >
</template>

<style scoped src="../../styles/payments.css"></style>
