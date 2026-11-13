<script setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  useId,
} from "vue";
import AppIcon from "./AppIcon.vue";
import { dateFor, formatDay, shiftDay } from "../composables/calendarDates.js";
const props = defineProps({ from: String, to: String });
const emit = defineEmits(["change"]);
const id = useId(),
  root = ref(null),
  trigger = ref(null),
  open = ref(false);
const month = ref(""),
  start = ref(""),
  end = ref(""),
  focus = ref(""),
  error = ref("");
const weekdays = ["lu", "ma", "mi", "ju", "vi", "sá", "do"];
const popupStyle = ref({});
function positionCalendar() {
  const box = trigger.value?.getBoundingClientRect();
  const popup = root.value?.querySelector(".overview-calendar");
  if (!box || !popup) return;
  const size = popup.getBoundingClientRect();
  popupStyle.value = {
    left: `${Math.max(8, Math.min(box.right - size.width, innerWidth - size.width - 8))}px`,
    top: `${Math.max(8, Math.min(box.bottom + 8, innerHeight - size.height - 8))}px`,
  };
}
const cells = computed(() => {
  if (!month.value) return [];
  const first = `${month.value}-01`,
    offset = (dateFor(first).getUTCDay() + 6) % 7;
  return Array.from({ length: 42 }, (_, i) => {
    const key = shiftDay(first, i - offset);
    return {
      key,
      day: dateFor(key).getUTCDate(),
      outside: !key.startsWith(month.value),
    };
  });
});
const label = computed(
  () => `${formatDay(props.from)} – ${formatDay(props.to)}`,
);
function show() {
  start.value = props.from;
  end.value = props.to;
  focus.value = props.to;
  month.value = props.to.slice(0, 7);
  error.value = "";
  open.value = true;
  nextTick(() => {
    positionCalendar();
    root.value?.querySelector('[tabindex="0"]')?.focus();
  });
}
function close() {
  open.value = false;
  trigger.value?.focus();
}
function choose(key) {
  error.value = "";
  focus.value = key;
  if (!start.value || end.value) {
    start.value = key;
    end.value = "";
  } else {
    const first = key < start.value ? key : start.value,
      last = key > start.value ? key : start.value;
    if ((dateFor(last) - dateFor(first)) / 86400000 >= 366) {
      error.value = "Elegí un período de hasta 366 días.";
      return;
    }
    start.value = first;
    end.value = last;
  }
}
function moveMonth(amount) {
  const date = dateFor(`${month.value}-01`);
  date.setUTCMonth(date.getUTCMonth() + amount);
  month.value = date.toISOString().slice(0, 7);
  focus.value = `${month.value}-01`;
}
async function navigate(event, key) {
  const shifts = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -7, ArrowDown: 7 };
  let next;
  if (event.key in shifts) next = shiftDay(key, shifts[event.key]);
  else if (event.key === "Home")
    next = shiftDay(key, -((dateFor(key).getUTCDay() + 6) % 7));
  else if (event.key === "End")
    next = shiftDay(key, 6 - ((dateFor(key).getUTCDay() + 6) % 7));
  else return;
  event.preventDefault();
  focus.value = next;
  month.value = next.slice(0, 7);
  await nextTick();
  root.value?.querySelector(`[data-date="${next}"]`)?.focus();
}
function apply() {
  emit("change", { from: start.value, to: end.value });
  close();
}
function outside(event) {
  if (open.value && !root.value?.contains(event.target)) open.value = false;
}
function escape(event) {
  if (open.value && event.key === "Escape") {
    event.preventDefault();
    event.stopPropagation();
    close();
  }
}
onMounted(() => {
  document.addEventListener("pointerdown", outside);
  document.addEventListener("keydown", escape, true);
});
onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", outside);
  document.removeEventListener("keydown", escape, true);
});
</script>
<template>
  <div ref="root" class="overview-date-range">
    <button
      ref="trigger"
      type="button"
      class="overview-calendar-trigger"
      :aria-label="'Elegir período: ' + label"
      aria-haspopup="dialog"
      :aria-expanded="open"
      :aria-controls="id"
      @click="open ? close() : show()"
    >
      <AppIcon name="calendar" :size="15" /><span>{{ label }}</span
      ><AppIcon name="chevron" :size="14" />
    </button>
    <section
      v-if="open"
      :id="id"
      role="dialog"
      aria-label="Elegir período del resumen"
      class="overview-calendar"
      :style="popupStyle"
    >
      <div class="calendar-heading">
        <button
          type="button"
          class="icon-button"
          aria-label="Mes anterior"
          @click="moveMonth(-1)"
        >
          <AppIcon name="arrow" :size="16" class="previous-month" />
        </button>
        <strong>{{
          formatDay(month + "-01", { month: "long", year: "numeric" })
        }}</strong>
        <button
          type="button"
          class="icon-button"
          aria-label="Mes siguiente"
          @click="moveMonth(1)"
        >
          <AppIcon name="arrow" :size="16" />
        </button>
      </div>
      <p class="calendar-instruction" role="status">
        {{
          end
            ? "Elegí un nuevo inicio o aplicá el período."
            : "Ahora elegí el último día del período."
        }}
      </p>
      <table
        class="calendar-grid"
        aria-label="Calendario de selección de período"
      >
        <thead>
          <tr>
            <th v-for="day in weekdays" :key="day" scope="col">{{ day }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="week in 6" :key="week">
            <td
              v-for="cell in cells.slice((week - 1) * 7, week * 7)"
              :key="cell.key"
            >
              <button
                type="button"
                :data-date="cell.key"
                :tabindex="focus === cell.key ? 0 : -1"
                :aria-label="
                  formatDay(cell.key, {
                    day: 'numeric',
                    month: 'long',
                    year: 'numeric',
                  })
                "
                :aria-pressed="cell.key === start || cell.key === end"
                :class="{
                  'is-outside': cell.outside,
                  'is-endpoint': cell.key === start || cell.key === end,
                  'is-between': end && cell.key > start && cell.key < end,
                }"
                @click="choose(cell.key)"
                @keydown="navigate($event, cell.key)"
              >
                {{ cell.day }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="calendar-selection">
        {{ start ? formatDay(start) : "Inicio" }} –
        {{ end ? formatDay(end) : "Fin" }}
      </p>
      <p v-if="error" role="alert" class="error-msg">{{ error }}</p>
      <div class="calendar-actions">
        <button type="button" class="btn-secondary" @click="close">
          Cancelar</button
        ><button
          type="button"
          class="btn-primary"
          :disabled="!start || !end || !!error"
          @click="apply"
        >
          Aplicar período
        </button>
      </div>
    </section>
  </div>
</template>
<style scoped>
.overview-date-range {
  position: relative;
}
.overview-calendar-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-muted);
  background: var(--surface);
  font-size: 12px;
}
.overview-calendar {
  position: fixed;
  z-index: 200;
  right: auto;
  top: 8px;
  max-height: calc(100dvh - 16px);
  overflow: auto;
  width: 322px;
  max-width: calc(100vw - 48px);
  background: var(--surface);
  border: 1px solid var(--control-border);
  border-radius: 12px;
  box-shadow: 0 12px 36px #0003;
  padding: 12px;
}
.calendar-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  text-transform: capitalize;
}
.calendar-heading strong {
  font-size: 13px;
}
.previous-month {
  transform: rotate(180deg);
}
.calendar-instruction,
.calendar-selection {
  font-size: 12px;
  line-height: 18px;
  color: var(--text-muted);
  margin: 8px 0;
  min-height: 36px;
}
.calendar-selection {
  min-height: 18px;
}
.calendar-grid {
  border-collapse: collapse;
  width: 100%;
  table-layout: fixed;
}
.calendar-grid th {
  font-size: 11px;
  color: var(--text-muted);
  padding-block: 6px;
  font-weight: 500;
}
.calendar-grid td {
  padding: 1px;
}
.calendar-grid button {
  width: 100%;
  height: 38px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  font-size: 12px;
}
.calendar-grid button:hover {
  background: var(--hover);
}
.calendar-grid .is-outside {
  color: var(--text-muted);
  opacity: 0.55;
}
.calendar-grid .is-between {
  background: var(--hover);
}
.calendar-grid .is-endpoint {
  background: var(--accent);
  color: var(--accent-ink);
  opacity: 1;
}
.calendar-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}
@media (max-width: 800px) {
  .calendar-grid button {
    height: 44px;
  }
  .overview-calendar-trigger {
    min-height: 44px;
    width: 44px;
    justify-content: center;
    padding: 0;
  }
  .overview-calendar-trigger > span,
  .overview-calendar-trigger > svg:last-child {
    display: none;
  }
}
</style>
