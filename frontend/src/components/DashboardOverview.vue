<script setup>
import { computed, ref } from "vue";
import DashboardSkeleton from "./DashboardSkeleton.vue";
import {
  useWorkspace,
  money,
  statusLabels,
} from "../composables/useWorkspace.js";
defineProps({ role: String });
const report = ref(null),
  currency = ref("ARS"),
  showDaily = ref(false);
const day = (d) =>
  new Intl.DateTimeFormat("en-CA", {
    timeZone: "America/Argentina/Buenos_Aires",
  }).format(d);
const to = ref(day(new Date())),
  from = ref(day(new Date(Date.now() - 6 * 86400000)));
const w = useWorkspace(async ({ request, tenant, platform, current }) => {
  report.value = null;
  if (!platform && !tenant) return;
  const data = await request(
    `/analytics/${platform ? "platform" : "restaurant"}?from=${from.value}&to=${to.value}`,
  );
  if (current()) report.value = data;
});
const amounts = computed(
  () => report.value?.money?.find((x) => x.currency === currency.value) || {},
);
const currencies = computed(() => [
  ...new Set(["ARS", ...(report.value?.money || []).map((x) => x.currency)]),
]);
const previous = computed(() =>
  report.value?.previous_period?.sales?.find(
    (x) => x.currency === currency.value,
  ),
);
const comparison = computed(() =>
  !previous.value || Number(previous.value.amount) === 0
    ? "Sin base de comparación"
    : `${((Number(amounts.value.sales || 0) / Number(previous.value.amount) - 1) * 100).toFixed(1)}% frente al período anterior`,
);
const days = computed(() =>
  w.platform.value
    ? report.value?.daily_new_businesses || []
    : (report.value?.daily_sales || []).filter(
        (x) => x.currency === currency.value,
      ),
);
const max = computed(() =>
  Math.max(
    1,
    ...days.value.map((x) => Number(w.platform.value ? x.registered : x.total)),
  ),
);
const cards = computed(() =>
  w.platform.value
    ? [
        [
          "Cobros de membresías",
          money(amounts.value.collected ?? 0, currency.value),
          "Cobros netos de devoluciones del período",
        ],
        [
          "Ingreso mensual contratado",
          money(amounts.value.active_mrr ?? 0, currency.value),
          "Contratos vigentes; no equivale a dinero cobrado",
        ],
        [
          "Negocios nuevos",
          report.value?.businesses?.production_registered_in_period ?? 0,
          "Clasificados como producción",
        ],
        [
          "Suscripciones activas",
          report.value?.subscriptions?.active ?? 0,
          "Con período vigente",
        ],
        [
          "Saldo de membresías",
          money(amounts.value.outstanding ?? 0, currency.value),
          "Facturas pendientes",
        ],
        [
          "Negocios activos / 7 días",
          report.value?.businesses?.active_last_week ?? 0,
          "Actividad real registrada",
        ],
      ]
    : [
        [
          "Ventas",
          money(amounts.value.sales ?? 0, currency.value),
          comparison.value,
        ],
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
      ],
);
function preset(count) {
  to.value = day(new Date());
  from.value = day(new Date(Date.now() - (count - 1) * 86400000));
  w.refresh();
}
</script>
<template>
  <section class="operations-page">
    <header class="section-header">
      <div>
        <p class="eyebrow">
          {{
            w.platform.value
              ? "LEGUIBURGER COMO NEGOCIO"
              : "EL PULSO DE TU HAMBURGUESERÍA"
          }}
        </p>
        <h1>
          {{ w.platform.value ? "Cada negocio cuenta." : "Todo a punto." }}
        </h1>
        <p>
          Datos reales, en horario de Argentina. Los movimientos de prueba
          quedan fuera.
        </p>
      </div>
      <router-link v-if="!w.platform.value" class="btn-primary" to="/orders"
        >Abrir caja</router-link
      ><router-link v-else class="btn-secondary" to="/platform/health"
        >Estado del sistema</router-link
      >
    </header>
    <div class="workspace-filters">
      <label v-if="!w.platform.value"
        >Local<select v-model="w.tenant.value">
          <option v-for="t in w.tenants.value" :key="t.id" :value="t.id">
            {{ t.subdomain }}
          </option>
        </select></label
      ><label>Desde<input v-model="from" type="date" /></label
      ><label>Hasta<input v-model="to" type="date" /></label
      ><label
        >Moneda<select v-model="currency">
          <option v-for="c in currencies" :key="c">{{ c }}</option>
        </select></label
      ><button
        class="btn-secondary"
        :disabled="w.loading.value"
        @click="w.refresh"
      >
        Consultar</button
      ><button class="text-button" @click="preset(7)">7 días</button
      ><button class="text-button" @click="preset(30)">30 días</button>
    </div>
    <div v-if="w.error.value" class="error-msg" role="alert">
      {{ w.error.value }}
      <button class="text-button" @click="w.refresh">Reintentar</button>
    </div>
    <DashboardSkeleton v-if="w.loading.value" view="overview" label="Consultando la actividad…" />
    <template v-else-if="report"
      ><div class="metric-grid">
        <article v-for="card in cards" :key="card[0]" class="card">
          <h3>{{ card[0] }}</h3>
          <strong>{{ card[1] }}</strong
          ><small>{{ card[2] }}</small>
        </article>
      </div>
      <p
        v-if="w.platform.value && report.businesses?.unclassified"
        class="inline-notice"
      >
        Hay {{ report.businesses.unclassified }} marcas sin clasificar. Sus
        cobros no se incluyen en la facturación de producción.
      </p>
      <div class="report-grid">
        <section class="card">
          <h2>{{ w.platform.value ? "Nuevos negocios" : "Ventas por día" }}</h2>
          <p v-if="!days.length" class="no-data">
            Todavía no hay actividad para este período.
          </p>
          <template v-else
            ><div
              class="bar-chart"
              role="img"
              aria-label="Actividad por día. Consultá el detalle debajo."
            >
              <div
                v-for="d in days"
                :key="d.day"
                class="bar"
                :style="{
                  height: `${(Number(w.platform.value ? d.registered : d.total) / max) * 100}%`,
                }"
                :title="`${d.day}: ${w.platform.value ? d.registered : money(d.total, currency)}`"
              />
            </div>
            <div class="chart-labels">
              <span>{{ days[0]?.day }}</span
              ><span>{{ days.at(-1)?.day }}</span>
            </div>
            <button
              class="text-button"
              :aria-expanded="showDaily"
              @click="showDaily = !showDaily"
            >
              {{ showDaily ? "Ocultar" : "Ver" }} detalle diario
            </button>
            <table v-if="showDaily" class="data-table">
              <tbody>
                <tr v-for="d in days" :key="d.day">
                  <td>{{ d.day }}</td>
                  <td>
                    {{
                      w.platform.value ? d.registered : money(d.total, currency)
                    }}
                  </td>
                </tr>
              </tbody>
            </table></template
          >
        </section>
        <section class="card">
          <h2>{{ w.platform.value ? "Suscripciones" : "Los más pedidos" }}</h2>
          <template v-if="w.platform.value"
            ><div
              v-for="[label, value] in [
                ['Activas', report.subscriptions?.active],
                ['En prueba', report.subscriptions?.trialing],
                ['Con atraso', report.subscriptions?.past_due],
                [
                  'Canceladas en el período',
                  report.subscriptions?.canceled_in_period,
                ],
              ]"
              :key="label"
              class="report-row"
            >
              <span>{{ label }}</span
              ><strong>{{ value || 0 }}</strong>
            </div></template
          ><template v-else
            ><div
              v-for="p in report.top_products?.filter(
                (x) => x.currency === currency,
              )"
              :key="p.product_id"
              class="report-row"
            >
              <span>{{ p.product_name }}</span
              ><strong>{{ p.units }} u.</strong>
            </div>
            <p v-if="!report.top_products?.length" class="no-data">
              Tus productos aparecerán con las primeras ventas.
            </p></template
          >
        </section>
      </div>
      <section class="card table-scroll">
        <table class="data-table">
          <thead>
            <tr>
              <th>
                {{ w.platform.value ? "Función utilizada" : "Pedido reciente" }}
              </th>
              <th>{{ w.platform.value ? "Eventos" : "Estado" }}</th>
              <th>{{ w.platform.value ? "Negocios" : "Total" }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in w.platform.value
                ? report.activity_by_feature
                : report.recent_orders"
              :key="item.id || item.event_name"
            >
              <td>{{ item.event_name || item.id.slice(0, 8) }}</td>
              <td>
                {{ w.platform.value ? item.events : statusLabels[item.status] }}
              </td>
              <td>
                {{
                  w.platform.value
                    ? item.businesses
                    : money(item.total_due, item.currency)
                }}
              </td>
            </tr>
          </tbody>
        </table>
      </section></template
    >
    <p v-else-if="!w.error.value" class="no-data">
      Creá un local activo para empezar a ver tus números.
    </p>
    <div v-if="!w.platform.value" class="operation-actions">
      <router-link to="/inventory" class="btn-secondary"
        >Stock e insumos</router-link
      ><router-link to="/expenses" class="btn-secondary">Gastos</router-link
      ><router-link to="/onboarding" class="btn-secondary"
        >Preparar mi negocio</router-link
      >
    </div>
  </section>
</template>
