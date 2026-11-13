<script setup>
import { computed, onBeforeUnmount, onMounted, ref, useId, watch } from "vue";
import { datesBetween, formatDay } from "../composables/calendarDates.js";
import { money } from "../composables/useWorkspace.js";
const props = defineProps({
  data: Array,
  from: String,
  to: String,
  currency: { type: String, default: "ARS" },
  count: Boolean,
  loading: Boolean,
});
const root = ref(null),
  width = ref(640),
  active = ref(null),
  id = `overview-fill-${useId()}`;
const height = 250,
  left = computed(() => (props.count ? 32 : 58)),
  right = 12,
  top = 12,
  bottom = 30;
let observer;
onMounted(() => {
  observer = new ResizeObserver(([entry]) => {
    width.value = Math.max(200, entry.contentRect.width);
  });
  observer.observe(root.value);
});
onBeforeUnmount(() => observer?.disconnect());
const points = computed(() => {
  const values = new Map(
    (props.data || []).map((x) => [
      x.day,
      Number(props.count ? x.registered : x.total),
    ]),
  );
  return datesBetween(props.from, props.to).map((day) => ({
    day,
    value: Number.isFinite(values.get(day)) ? values.get(day) : 0,
  }));
});
const max = computed(() => Math.max(1, ...points.value.map((x) => x.value)));
const min = computed(() => Math.min(0, ...points.value.map((x) => x.value)));
const plotHeight = height - top - bottom;
const y = (value) =>
  top + ((max.value - value) / (max.value - min.value)) * plotHeight;
const positions = computed(() =>
  points.value.map((point, index) => ({
    ...point,
    x:
      left.value +
      (index / Math.max(1, points.value.length - 1)) *
        (width.value - left.value - right),
    y: y(point.value),
  })),
);
// Monotone cubic interpolation keeps the curve within neighboring values,
// including zero days between peaks, as in the reference area chart.
const line = computed(() => {
  const p = positions.value;
  if (!p.length) return "";
  if (p.length === 1) return `M${p[0].x},${p[0].y}`;
  const slopes = p.slice(1).map((v, i) => (v.y - p[i].y) / (v.x - p[i].x));
  const tangents = p.map((_, i) =>
    i === 0
      ? slopes[0]
      : i === p.length - 1
        ? slopes.at(-1)
        : slopes[i - 1] * slopes[i] <= 0
          ? 0
          : (slopes[i - 1] + slopes[i]) / 2,
  );
  slopes.forEach((slope, i) => {
    if (slope === 0) {
      tangents[i] = 0;
      tangents[i + 1] = 0;
      return;
    }
    const ratio = Math.hypot(tangents[i] / slope, tangents[i + 1] / slope);
    if (ratio > 3) {
      tangents[i] *= 3 / ratio;
      tangents[i + 1] *= 3 / ratio;
    }
  });
  return p.slice(1).reduce((path, point, i) => {
    const before = p[i],
      dx = (point.x - before.x) / 3;
    return `${path} C${before.x + dx},${before.y + tangents[i] * dx} ${point.x - dx},${point.y - tangents[i + 1] * dx} ${point.x},${point.y}`;
  }, `M${p[0].x},${p[0].y}`);
});
const area = computed(() =>
  !positions.value.length
    ? ""
    : `${line.value} L${positions.value.at(-1).x},${y(0)} L${positions.value[0].x},${y(0)} Z`,
);
const ticks = computed(() =>
  Array.from(
    { length: 5 },
    (_, i) => min.value + ((max.value - min.value) * i) / 4,
  ).filter((v) => !props.count || Number.isInteger(v)),
);
const labels = computed(() => {
  const p = positions.value,
    step = Math.max(1, Math.ceil(p.length / (width.value < 450 ? 4 : 7)));
  return p.filter(
    (_, i) =>
      i === 0 ||
      i === p.length - 1 ||
      (i % step === 0 && p.at(-1).x - p[i].x > 50),
  );
});
const selected = computed(() =>
  active.value === null ? null : positions.value[active.value],
);
const seriesKey = computed(
  () =>
    `${props.from}:${props.to}:${props.currency}:${points.value.map((p) => p.value).join()}`,
);
const empty = computed(() => !points.value.some((p) => p.value !== 0));
const formatValue = (value) =>
  props.count
    ? new Intl.NumberFormat("es-AR").format(value)
    : money(value, props.currency);
function axisValue(value) {
  const formatted = new Intl.NumberFormat("es-AR", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);
  return props.count ? formatted : `$ ${formatted}`;
}
function pointer(event) {
  const box = root.value.getBoundingClientRect();
  active.value = Math.max(
    0,
    Math.min(
      points.value.length - 1,
      Math.round(
        ((event.clientX - box.left - left.value) /
          (width.value - left.value - right)) *
          (points.value.length - 1),
      ),
    ),
  );
}
function keydown(event) {
  if (!["ArrowLeft", "ArrowRight", "Home", "End", "Escape"].includes(event.key))
    return;
  event.preventDefault();
  if (event.key === "Escape") active.value = null;
  else if (event.key === "Home") active.value = 0;
  else if (event.key === "End") active.value = points.value.length - 1;
  else
    active.value = Math.max(
      0,
      Math.min(
        points.value.length - 1,
        (active.value ?? -1) + (event.key === "ArrowRight" ? 1 : -1),
      ),
    );
}
watch(seriesKey, () => {
  active.value = null;
});
</script>
<template>
  <div
    ref="root"
    class="overview-area-chart"
    :class="{ 'chart-loading': loading }"
    tabindex="0"
    role="img"
    :aria-label="`${count ? 'Nuevos restaurantes' : 'Ventas diarias en ' + currency}. Del ${formatDay(from)} al ${formatDay(to)}. Usá las flechas para consultar cada día o abrí el detalle diario.`"
    @pointermove="pointer"
    @pointerdown="pointer"
    @pointerleave="if ($event.pointerType !== 'touch') active = null;"
    @blur="active = null"
    @keydown="keydown"
  >
    <svg
      :viewBox="`0 0 ${width} ${height}`"
      width="100%"
      :height="height"
      aria-hidden="true"
    >
      <defs>
        <linearGradient :id="id" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--accent)" stop-opacity=".34" />
          <stop offset="100%" stop-color="var(--accent)" stop-opacity=".025" />
        </linearGradient>
      </defs>
      <g v-for="tick in ticks" :key="tick" class="chart-axis">
        <line :x1="left" :x2="width - right" :y1="y(tick)" :y2="y(tick)" />
        <text :x="left - 8" :y="y(tick) + 4" text-anchor="end">
          {{ axisValue(tick) }}
        </text>
      </g>
      <g :key="seriesKey" class="chart-series">
        <path :d="area" :fill="`url(#${id})`" />
        <path
          class="chart-line"
          :d="line"
          fill="none"
          stroke="var(--accent)"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </g>
      <g v-for="point in labels" :key="point.day" class="chart-axis">
        <text
          :x="point.x"
          :y="height - 6"
          :text-anchor="
            point.day === from ? 'start' : point.day === to ? 'end' : 'middle'
          "
        >
          {{ formatDay(point.day) }}
        </text>
      </g>
      <g v-if="selected && !loading" class="chart-selection">
        <line
          :x1="selected.x"
          :x2="selected.x"
          :y1="top"
          :y2="height - bottom"
          stroke="var(--control-border)"
          stroke-dasharray="3 5"
        />
        <circle
          :cx="selected.x"
          :cy="selected.y"
          r="4"
          fill="var(--accent)"
          stroke="var(--surface)"
          stroke-width="2"
        />
      </g>
    </svg>
    <div
      v-if="selected && !loading"
      class="chart-tooltip"
      role="status"
      :style="{ left: `clamp(6px, ${selected.x}px, calc(100% - 172px))` }"
    >
      <span>{{
        formatDay(selected.day, { day: "numeric", month: "long" })
      }}</span><strong
        >{{ count ? "Restaurantes" : "Ventas" }}:
        {{ formatValue(selected.value) }}</strong
      >
    </div>
    <p v-if="empty && !loading" class="chart-empty">
      {{
        count
          ? "Todavía no hay nuevos restaurantes en este período."
          : "Todavía no hay ventas registradas en este período."
      }}
    </p>
  </div>
</template>
<style scoped>
.overview-area-chart {
  position: relative;
  width: 100%;
  height: 250px;
  min-width: 0;
  touch-action: pan-y;
  border-radius: 8px;
}
svg {
  display: block;
  overflow: visible;
}
.chart-axis line {
  stroke: var(--border);
  stroke-dasharray: 3 5;
}
.chart-axis text {
  fill: var(--text-muted);
  font-size: 11px;
  font-family: inherit;
}
.chart-series {
  animation: area-appear 420ms ease-out both;
  transform-origin: center bottom;
}
.chart-tooltip {
  position: absolute;
  top: 8px;
  z-index: 2;
  pointer-events: none;
  background: var(--surface);
  border: 1px solid var(--border);
  box-shadow: 0 6px 20px #0002;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 12px;
  max-width: 170px;
}
.chart-tooltip span,
.chart-tooltip strong {
  display: block;
}
.chart-tooltip span {
  color: var(--text-muted);
  margin-bottom: 4px;
}
.chart-tooltip strong {
  font-weight: 550;
}
.chart-empty {
  position: absolute;
  inset: 65px 20px auto 60px;
  text-align: center;
  color: var(--text-muted);
  font-size: 12px;
  pointer-events: none;
}
.chart-loading svg {
  opacity: 0.15;
}
.chart-loading::after {
  content: "";
  position: absolute;
  inset: 10px 12px 30px 58px;
  background: var(--surface-raised);
  border-radius: 8px;
  animation: skeleton-pulse 1.3s ease infinite;
}
@keyframes area-appear {
  from {
    opacity: 0.3;
    transform: scaleY(0.9);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
</style>
