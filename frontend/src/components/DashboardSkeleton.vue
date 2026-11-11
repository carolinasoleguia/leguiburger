<script setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import AppSpinner from "./AppSpinner.vue";
const props = defineProps({
  view: { type: String, default: "auto" },
  label: { type: String, default: "Preparando tu espacio…" },
  compact: Boolean,
});
const route = useRoute();
const layout = computed(() => {
  if (props.view !== "auto") return props.view;
  const path = route.path;
  if (["/admin", "/owner"].includes(path)) return "overview";
  if (path === "/kitchen" || path.includes("/employee")) return "kitchen";
  if (path === "/care") return "support";
  if (path.includes("catalog")) return "catalog";
  if (
    [
      "/team",
      "/locations",
      "/restaurants",
      "/notifications",
      "/onboarding",
    ].includes(path)
  )
    return "cards";
  if (path.includes("/account/") || path.includes("invitation")) return "form";
  return "table";
});
const metrics = [
  "Ventas",
  "Pedidos",
  "Ticket promedio",
  "Cobrado",
  "Gastos operativos",
  "Resultado estimado",
];
</script>
<template>
  <div
    class="dashboard-skeleton"
    :class="[
      `dashboard-skeleton--${layout}`,
      { 'dashboard-skeleton--compact': compact },
    ]"
    role="status"
    aria-live="polite"
    aria-busy="true"
    data-dashboard-loading
  >
    <span class="visually-hidden">{{ label }}</span>
    <template v-if="layout === 'overview'">
      <div class="metric-grid" aria-hidden="true">
        <article
          v-for="(metric, i) in metrics"
          :key="metric"
          class="card skeleton-metric"
        >
          <span
            class="skeleton-bone skeleton-label"
            :style="{ '--bone-width': `${48 + (i % 3) * 12}%` }"
          ></span
          ><span class="skeleton-bone skeleton-number"></span
          ><span class="skeleton-bone skeleton-caption"></span>
        </article>
      </div>
      <div class="report-grid" aria-hidden="true">
        <section v-for="panel in 2" :key="panel" class="card skeleton-panel">
          <span class="skeleton-bone skeleton-title"></span>
          <div v-if="panel === 1" class="skeleton-chart">
            <span
              v-for="bar in 10"
              :key="bar"
              class="skeleton-bone"
              :style="{
                height: `${[35, 62, 48, 80, 52, 94, 67, 82, 60, 75][bar - 1]}%`,
              }"
            ></span>
          </div>
          <div v-else class="skeleton-panel-data">
            <AppSpinner :size="30" />
          </div>
        </section>
      </div>
    </template>
    <template
      v-else-if="
        layout === 'cards' || layout === 'catalog' || layout === 'kitchen'
      "
    >
      <div class="skeleton-toolbar" aria-hidden="true">
        <span class="skeleton-bone skeleton-field"></span
        ><span class="skeleton-bone skeleton-action"></span>
      </div>
      <div
        class="skeleton-card-grid"
        :class="{ 'skeleton-card-grid--kitchen': layout === 'kitchen' }"
        aria-hidden="true"
      >
        <article
          v-for="item in compact ? 3 : 6"
          :key="item"
          class="card skeleton-card"
        >
          <span
            v-if="layout === 'catalog'"
            class="skeleton-bone skeleton-image"
          ></span>
          <div class="skeleton-card-heading">
            <span class="skeleton-bone skeleton-avatar"></span
            ><span class="skeleton-bone skeleton-title"></span>
          </div>
          <span class="skeleton-bone skeleton-line"></span
          ><span
            class="skeleton-bone skeleton-line skeleton-line--short"
          ></span>
          <div class="skeleton-card-footer">
            <span class="skeleton-bone skeleton-label"></span
            ><span class="skeleton-bone skeleton-action"></span>
          </div>
        </article>
      </div>
    </template>
    <template v-else-if="layout === 'support' || layout === 'form'">
      <div class="skeleton-form-grid" aria-hidden="true">
        <section v-for="panel in 2" :key="panel" class="card skeleton-panel">
          <span class="skeleton-bone skeleton-title"></span>
          <div v-for="field in 3" :key="field" class="skeleton-form-field">
            <span class="skeleton-bone skeleton-label"></span
            ><span
              class="skeleton-bone skeleton-field"
              :class="{
                'skeleton-textarea': field === 3 && layout === 'support',
              }"
            ></span>
          </div>
          <span class="skeleton-bone skeleton-action"></span>
        </section>
      </div>
    </template>
    <template v-else>
      <div class="card skeleton-panel" aria-hidden="true">
        <div class="skeleton-toolbar">
          <span class="skeleton-bone skeleton-field"></span
          ><span class="skeleton-bone skeleton-action"></span>
        </div>
        <div class="skeleton-table">
          <div
            v-for="row in compact ? 4 : 7"
            :key="row"
            class="skeleton-table-row"
          >
            <span
              v-for="column in 4"
              :key="column"
              class="skeleton-bone"
              :style="{
                '--bone-width': `${column === 1 ? 78 : 40 + ((row + column) % 4) * 12}%`,
              }"
            ></span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
