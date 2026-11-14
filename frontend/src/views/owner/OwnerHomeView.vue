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
      subscriptions: {},
      businesses: {},
      activity_by_feature: [],
    },
);
const w = useWorkspace(async ({ request, current }) => {
  response.value = null;
  const data = await request(
    `/analytics/platform?from=${from.value}&to=${to.value}`,
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
const cards = computed(() => [
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
]);
const mainCards = computed(() => cards.value.slice(0, 4));
const secondaryCards = computed(() => cards.value.slice(4));
const days = computed(() => report.value.daily_new_businesses || []);
const stateRows = computed(() => [
  ["Activas", report.value.subscriptions?.active || 0],
  ["En prueba", report.value.subscriptions?.trialing || 0],
  ["Con atraso", report.value.subscriptions?.past_due || 0],
  ["Canceladas", report.value.subscriptions?.canceled_in_period || 0],
]);
const maxState = computed(() =>
  Math.max(1, ...stateRows.value.map((s) => s[1])),
);
</script>
<template>
  <section
    class="operations-page overview-page reference-overview"
    data-overview="platform"
  >
    <header class="section-header">
      <div class="section-title">
        <h1>Actividad de LeguiBurger</h1>
        <p>Restaurantes, suscripciones y cobros de la plataforma.</p>
      </div>
      <div class="page-heading-actions overview-context">
        <label
          >Moneda<select v-model="currency">
            <option v-for="c in currencies" :key="c">{{ c }}</option>
          </select></label
        ><router-link class="btn-secondary" to="/platform/health"
          >Estado del sistema</router-link
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
      label="Consultando la actividad de la plataforma…"
      view="overview"
      ><div class="metric-grid" aria-label="Indicadores del período">
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
              <h2>Nuevos restaurantes</h2>
              <p>Altas registradas en el período seleccionado</p>
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
            count
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
                  <th>Restaurantes nuevos</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="d in days" :key="d.day">
                  <td>
                    {{ formatDay(d.day, { day: "numeric", month: "long" }) }}
                  </td>
                  <td>{{ d.registered }}</td>
                </tr>
                <tr v-if="!days.length">
                  <td colspan="2">Sin altas registradas en este período.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
        <section class="card overview-states">
          <h2>Suscripciones</h2>
          <p class="overview-panel-description">
            Contratos de restaurantes en LeguiBurger
          </p>
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
          <p class="overview-platform-note">
            {{
              report.businesses?.unclassified
                ? `Hay ${report.businesses.unclassified} marcas sin clasificar. Sus cobros no se suman a producción.`
                : "Los cobros de membresías corresponden exclusivamente a la plataforma."
            }}
          </p>
        </section>
      </div>
      <div class="overview-secondary">
        <section class="card overview-products">
          <div class="overview-panel-heading">
            <div>
              <h2>Actividad de los restaurantes</h2>
              <p class="overview-panel-description">
                Uso registrado en la plataforma
              </p>
            </div>
          </div>
          <div class="overview-product-list">
            <div
              v-for="item in report.activity_by_feature"
              :key="item.event_name"
              class="report-row"
            >
              <span>{{ item.event_name }}</span
              ><strong>{{ item.events }}</strong>
            </div>
            <p v-if="!report.activity_by_feature?.length" class="no-data">
              Todavía no hay actividad registrada.
            </p>
          </div>
        </section>
        <section class="card overview-result">
          <h2>Balance de membresías</h2>
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
            <h2>Actividad reciente</h2>
            <p class="overview-panel-description">
              Eventos y negocios que los utilizan
            </p>
          </div>
        </div>
        <div class="card table-scroll overview-recent-table">
          <table class="data-table">
            <thead>
              <tr>
                <th>Función utilizada</th>
                <th>Eventos</th>
                <th>Negocios</th>
              </tr>
            </thead>
            <tbody>
              <LoadingRows v-if="w.loading.value" :columns="3" />
              <tr
                v-for="item in report.activity_by_feature"
                v-show="!w.loading.value"
                :key="item.event_name"
              >
                <td>{{ item.event_name }}</td>
                <td>{{ item.events }}</td>
                <td>{{ item.businesses }}</td>
              </tr>
              <tr
                v-if="!report.activity_by_feature?.length"
                v-show="!w.loading.value"
              >
                <td colspan="3">Sin actividad registrada en este período.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </StableContent>
  </section>
</template>
