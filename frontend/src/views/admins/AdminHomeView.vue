<script setup>
import { computed, ref } from "vue";
import LoadingRows from "../../components/LoadingRows.vue";
import AppIcon from "../../components/AppIcon.vue";
import StableContent from "../../components/StableContent.vue";
import OverviewAreaChart from "../../components/OverviewAreaChart.vue";
import OverviewDateRange from "../../components/OverviewDateRange.vue";
import {
  todayKey,
  shiftDay,
  formatDay,
} from "../../composables/calendarDates.js";
import {
  useWorkspace,
  money,
  when,
  statusLabels,
} from "../../composables/useWorkspace.js";
const response = ref(null),
  currency = ref("ARS"),
  showDaily = ref(false);
const to = ref(todayKey()),
  from = ref(shiftDay(to.value, -6)),
  selectedRange = ref(7);
const report = computed(
  () =>
    response.value || {
      money: [],
      metrics: {},
      top_products: [],
      recent_orders: [],
      order_states: [],
    },
);
const w = useWorkspace(async ({ request, tenant, current }) => {
  response.value = null;
  if (!tenant) return;
  const data = await request(
    `/analytics/restaurant?from=${from.value}&to=${to.value}`,
  );
  if (current()) response.value = data;
});
const amounts = computed(
  () => report.value.money?.find((x) => x.currency === currency.value) || {},
);
const currencies = computed(() => [
  ...new Set(["ARS", ...(report.value.money || []).map((x) => x.currency)]),
]);
const periodLabel = computed(
  () => `${formatDay(from.value)} – ${formatDay(to.value)}`,
);
function preset(count) {
  selectedRange.value = count;
  to.value = todayKey();
  from.value = shiftDay(to.value, -(count - 1));
  showDaily.value = false;
  w.refresh();
}
function customPeriod(range) {
  from.value = range.from;
  to.value = range.to;
  selectedRange.value = 0;
  showDaily.value = false;
  w.refresh();
}
const previous = computed(() =>
  report.value.previous_period?.sales?.find(
    (x) => x.currency === currency.value,
  ),
);
const comparison = computed(() =>
  !previous.value || Number(previous.value.amount) === 0
    ? "Sin base de comparación"
    : `${((Number(amounts.value.sales || 0) / Number(previous.value.amount) - 1) * 100).toFixed(1)}% frente al período anterior`,
);
const days = computed(() =>
  (report.value.daily_sales || []).filter((x) => x.currency === currency.value),
);
const cards = computed(() => [
  ["Ventas", money(amounts.value.sales ?? 0, currency.value), comparison.value],
  [
    "Pedidos",
    report.value?.metrics?.orders_created ?? 0,
    "Creados en el período",
  ],
  [
    "Ticket promedio",
    money(amounts.value.average_ticket, currency.value),
    "Ventas / pedidos con venta",
  ],
  [
    "Cobrado",
    money(amounts.value.collected ?? 0, currency.value),
    "Cobros menos devoluciones del período",
  ],
  [
    "Gastos operativos",
    money(amounts.value.operating_expenses ?? 0, currency.value),
    "Gastos vigentes, sin anulados",
  ],
  [
    "Resultado estimado",
    money(amounts.value.estimated_operating_result, currency.value),
    Number(amounts.value.missing_cost_lines) > 0
      ? "Faltan costos: no se estima rentabilidad"
      : "Margen conocido menos gastos registrados",
  ],
]);
const mainCards = computed(() => [
  cards.value[0],
  cards.value[1],
  cards.value[3],
  cards.value[2],
]);
const secondaryCards = computed(() => cards.value.slice(4));
const counts = computed(
  () =>
    new Map(
      (report.value.order_states || []).map((s) => [
        s.status,
        Number(s.count) || 0,
      ]),
    ),
);
const stateRows = computed(() => [
  ["Por confirmar", counts.value.get("pending") || 0],
  [
    "En curso",
    ["accepted", "preparing", "ready", "dispatched"].reduce(
      (n, s) => n + (counts.value.get(s) || 0),
      0,
    ),
  ],
  ["Entregados", counts.value.get("delivered") || 0],
  [
    "Cancelados / rechazados",
    (counts.value.get("canceled") || 0) + (counts.value.get("rejected") || 0),
  ],
]);
const maxState = computed(() =>
  Math.max(1, ...stateRows.value.map((s) => s[1])),
);
const products = computed(() =>
  (report.value.top_products || []).filter(
    (p) => p.currency === currency.value,
  ),
);
const maxUnits = computed(() =>
  Math.max(1, ...products.value.map((p) => Number(p.units))),
);
</script>
<template>
  <section
    class="operations-page overview-page reference-overview"
    data-overview="restaurant"
  >
    <header class="section-header">
      <div class="section-title">
        <h1>Actividad del negocio</h1>
        <p>Ventas, pedidos y cobros de tu hamburguesería.</p>
      </div>
      <div class="page-heading-actions overview-context">
        <label
          >Local<select v-model="w.tenant.value">
            <option v-for="t in w.tenants.value" :key="t.id" :value="t.id">
              {{ t.display_name || t.subdomain }}
            </option>
          </select></label
        ><label
          >Moneda<select v-model="currency">
            <option v-for="c in currencies" :key="c">{{ c }}</option>
          </select></label
        ><router-link class="btn-primary" to="/orders"
          ><AppIcon name="plus" :size="16" />Nuevo pedido</router-link
        >
      </div>
    </header>
    <div v-if="w.error.value" class="error-msg" role="alert">
      {{ w.error.value
      }}<button class="text-button" @click="w.refresh">Reintentar</button>
    </div>
    <StableContent
      v-if="w.loading.value || response"
      class="stable-overview"
      :loading="w.loading.value"
      label="Consultando la actividad del restaurante…"
      view="overview"
    >
      <div class="metric-grid" aria-label="Indicadores del período">
        <article v-for="(card, i) in mainCards" :key="card[0]" class="card">
          <div class="overview-metric-label">
            <h3>{{ card[0] }}</h3>
            <span class="overview-metric-icon"
              ><AppIcon
                :name="['receipt', 'activity', 'card', 'burger'][i]"
                :size="16"
            /></span>
          </div>
          <strong>{{ card[1] }}</strong
          ><small>{{ card[2] }}</small>
        </article>
      </div>
      <div class="report-grid overview-activity">
        <section
          class="card overview-sales"
          :class="{ 'is-expanded': showDaily }"
        >
          <div class="overview-chart-heading">
            <div>
              <h2>Ventas del negocio</h2>
              <p>Facturación diaria en {{ currency }}</p>
            </div>
            <div class="overview-range-controls">
              <div
                class="overview-range-presets"
                role="group"
                aria-label="Rango de tiempo del resumen"
              >
                <button
                  v-for="range in [
                    { days: 30, label: '1 mes' },
                    { days: 15, label: '15 días' },
                    { days: 7, label: '7 días' },
                  ]"
                  :key="range.days"
                  type="button"
                  :aria-pressed="selectedRange === range.days"
                  @click="preset(range.days)"
                >
                  {{ range.label }}
                </button>
              </div>
              <OverviewDateRange :from="from" :to="to" @change="customPeriod" />
            </div>
          </div>
          <OverviewAreaChart
            :data="days"
            :from="from"
            :to="to"
            :currency="currency"
            :loading="w.loading.value"
          />
          <div class="overview-chart-footer">
            <span>Todo el resumen · {{ periodLabel }}</span
            ><button
              class="text-button"
              :aria-expanded="showDaily"
              @click="showDaily = !showDaily"
            >
              {{ showDaily ? "Ocultar" : "Ver" }} detalle diario
            </button>
          </div>
          <div v-if="showDaily" class="overview-daily-detail">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Día</th>
                  <th>Ventas</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="d in days" :key="d.day">
                  <td>
                    {{ formatDay(d.day, { day: "numeric", month: "long" }) }}
                  </td>
                  <td>{{ money(d.total, currency) }}</td>
                </tr>
                <tr v-if="!days.length">
                  <td colspan="2">
                    Sin movimientos registrados en este período.
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
        <section class="card overview-states">
          <h2>Estado de pedidos</h2>
          <p class="overview-panel-description">Cómo avanza el servicio</p>
          <div class="overview-state-list">
            <div
              v-for="([label, value], i) in stateRows"
              :key="label"
              class="overview-state"
            >
              <div>
                <span>{{ label }}</span
                ><strong>{{ value }}</strong>
              </div>
              <div class="overview-state-track">
                <span
                  :class="'state-tone-' + i"
                  :style="{
                    width: `${value ? Math.max(4, (value / maxState) * 100) : 0}%`,
                  }"
                />
              </div>
            </div>
          </div>
          <div class="overview-customers">
            <div>
              <h3>Clientes atendidos</h3>
              <strong>{{
                Number(report.metrics?.new_customers || 0) +
                Number(report.metrics?.returning_customers || 0)
              }}</strong>
            </div>
            <p>Nuevos y recurrentes</p>
            <dl>
              <div>
                <dt>Nuevos</dt>
                <dd>{{ report.metrics?.new_customers || 0 }}</dd>
              </div>
              <div>
                <dt>Recurrentes</dt>
                <dd>{{ report.metrics?.returning_customers || 0 }}</dd>
              </div>
            </dl>
          </div>
        </section>
      </div>
      <div class="overview-secondary">
        <section class="card overview-products">
          <div class="overview-panel-heading">
            <div>
              <h2>Los más pedidos</h2>
              <p class="overview-panel-description">
                Los favoritos de tus clientes
              </p>
            </div>
            <router-link class="text-button" to="/admin/catalog"
              >Ver catálogo<AppIcon name="arrow" :size="14"
            /></router-link>
          </div>
          <div class="overview-product-list">
            <div
              v-for="p in products"
              :key="p.product_id"
              class="overview-product"
            >
              <div>
                <span>{{ p.product_name }}</span
                ><strong>{{ p.units }} u.</strong>
              </div>
              <div class="overview-product-track">
                <span
                  :style="{ width: `${(Number(p.units) / maxUnits) * 100}%` }"
                />
              </div>
            </div>
            <p v-if="!products.length" class="no-data">
              Tus productos aparecerán con las primeras ventas.
            </p>
          </div>
        </section>
        <section class="card overview-result">
          <h2>Resultado del período</h2>
          <div
            v-for="card in secondaryCards"
            :key="card[0]"
            class="overview-result-item"
          >
            <h3>{{ card[0] }}</h3>
            <strong>{{ card[1] }}</strong
            ><small>{{ card[2] }}</small>
          </div>
        </section>
      </div>
      <section class="overview-recent-section">
        <div class="overview-panel-heading">
          <div>
            <h2>Pedidos recientes</h2>
            <p class="overview-panel-description">
              Los últimos movimientos de tu local
            </p>
          </div>
          <router-link to="/orders" class="text-button"
            >Ver todos<AppIcon name="arrow" :size="14"
          /></router-link>
        </div>
        <div class="card table-scroll overview-recent-table">
          <table class="data-table">
            <thead>
              <tr>
                <th>Pedido</th>
                <th>Fecha</th>
                <th>Estado</th>
                <th>Total</th>
              </tr>
            </thead>
            <tbody>
              <LoadingRows v-if="w.loading.value" :columns="4" />
              <tr
                v-for="item in report.recent_orders"
                v-show="!w.loading.value"
                :key="item.id"
              >
                <td>#{{ item.id.slice(0, 8) }}</td>
                <td>{{ when(item.created_at) }}</td>
                <td>{{ statusLabels[item.status] }}</td>
                <td>{{ money(item.total_due, item.currency) }}</td>
              </tr>
              <tr
                v-if="!report.recent_orders?.length"
                v-show="!w.loading.value"
              >
                <td colspan="4">Todavía no hay pedidos en este período.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </StableContent>
    <p v-else-if="!w.error.value" class="no-data">
      Creá un local activo para empezar a ver tus números.
    </p>
  </section>
</template>
