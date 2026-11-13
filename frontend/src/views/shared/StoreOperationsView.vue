<script setup>
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
const emptySettings = () => ({
  timezone: "America/Argentina/Buenos_Aires",
  schedule_enabled: false,
  paused: false,
  pause_until: null,
  pending_minutes: 15,
  prep_min: 20,
  prep_max: 35,
  periods: [],
  receiving: false,
  max_active_orders: 0,
  queue_step_minutes: 0,
  max_queue_minutes: 60,
  active_count: 0,
});
const loaded = ref(false);
const form = ref(emptySettings()),
  pauseMinutes = ref(30);
const days = [
  "Domingo",
  "Lunes",
  "Martes",
  "Miércoles",
  "Jueves",
  "Viernes",
  "Sábado",
];
const w = useWorkspace(async ({ request, tenant, current }) => {
  loaded.value = false;
  if (!tenant) return;
  const data = await request("/ordering/settings");
  if (current()) {
    form.value = { ...emptySettings(), ...data };
    loaded.value = true;
  }
});
function save() {
  if (!loaded.value) return;
  return w.act(
    () =>
      w.request(
        "/ordering/settings",
        {
          timezone: form.value.timezone,
          schedule_enabled: form.value.schedule_enabled,
          paused: form.value.paused,
          pause_minutes: form.value.paused ? Number(pauseMinutes.value) : 0,
          pending_minutes: Number(form.value.pending_minutes),
          prep_min: Number(form.value.prep_min),
          prep_max: Number(form.value.prep_max),
          max_active_orders: Number(form.value.max_active_orders || 0),
          queue_step_minutes: Number(form.value.queue_step_minutes || 0),
          max_queue_minutes: Number(form.value.max_queue_minutes ?? 60),
          periods: form.value.periods,
        },
        "PATCH",
      ),
    "Horarios y recepción de pedidos actualizados.",
  );
}
function add() {
  form.value.periods.push({ day: 1, opens: "19:00", closes: "23:00" });
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Atención del local"
    description="Definí cuándo recibís pedidos y cuánto tarda tu cocina."
  >
    <div class="operation-actions">
      <router-link class="btn-secondary" to="/delivery-zones"
        >Zonas de delivery</router-link
      ><router-link class="btn-secondary" to="/menu-availability"
        >Disponibilidad de la carta</router-link
      >
    </div>
    <form v-form-care class="card operation-form" @submit.prevent="save">
      <h2>
        {{
          form.receiving
            ? "Tu local está recibiendo pedidos"
            : "Tu local no está recibiendo pedidos"
        }}
      </h2>
      <p>
        La carta permanece visible cuando el local está cerrado. Para recibir
        pedidos también debe estar publicada y habilitada.
      </p>
      <router-link class="text-button" :to="'/catalogo/' + w.tenant.value"
        >Ver mi carta</router-link
      >
      <fieldset :disabled="w.busy.value || !loaded">
        <legend>Recepción</legend>
        <label class="ordering-check"
          ><input v-model="form.paused" type="checkbox" /> Pausar nuevos
          pedidos</label
        >
        <label v-if="form.paused"
          >Duración de la pausa<select v-model.number="pauseMinutes">
            <option :value="15">15 minutos</option>
            <option :value="30">30 minutos</option>
            <option :value="60">1 hora</option>
            <option :value="120">2 horas</option>
            <option :value="0">Hasta que la reactive</option>
          </select></label
        >
        <p v-if="form.paused && form.pause_until">
          Pausa actual hasta {{ when(form.pause_until) }}. Guardar renueva su
          duración.
        </p>
        <div class="form-row">
          <label
            >Confirmar pedidos antes de (min)<input
              v-model.number="form.pending_minutes"
              type="number"
              min="5"
              max="60"
              required
          /></label>
          <label
            >Preparación mínima (min)<input
              v-model.number="form.prep_min"
              type="number"
              min="1"
              max="180"
              required
          /></label>
          <label
            >Preparación máxima (min)<input
              v-model.number="form.prep_max"
              type="number"
              :min="form.prep_min"
              max="240"
              required
          /></label>
        </div>
        <p>
          Los pedidos web nuevos que no confirmes a tiempo se cancelan, siempre
          que no tengan un cobro en curso. Estos plazos no modifican pedidos
          anteriores.
        </p>
      </fieldset>
      <fieldset :disabled="w.busy.value || !loaded">
        <legend>Capacidad de cocina</legend>
        <p>
          Hay {{ form.active_count || 0 }} pedidos pendientes, aceptados o en
          preparación. Los pedidos listos y despachados dejan lugar para nuevas
          compras.
        </p>
        <div class="form-row">
          <label
            >Pedidos simultáneos admitidos<input
              v-model.number="form.max_active_orders"
              type="number"
              min="0"
              max="1000"
              required /></label
          ><label
            >Minutos extra por pedido en cola<input
              v-model.number="form.queue_step_minutes"
              type="number"
              min="0"
              max="30"
              required /></label
          ><label
            >Tope de demora adicional (min)<input
              v-model.number="form.max_queue_minutes"
              type="number"
              min="0"
              max="180"
              required
          /></label>
        </div>
        <p>
          Cero pedidos admitidos significa sin límite. Al alcanzar el límite, se
          pausan nuevas compras y pedidos de mostrador; se reactivan al liberar
          lugar. Cero minutos por pedido desactiva la demora adicional.
        </p>
        <p v-if="form.capacity_full" role="status">
          La cocina está al máximo de su capacidad.
        </p>
        <p>
          Estimación actual: {{ form.estimated_prep_min ?? form.prep_min }}–{{
            form.estimated_prep_max ?? form.prep_max
          }}
          min. Los pedidos existentes mantienen el tiempo que se les informó.
        </p>
      </fieldset>
      <fieldset :disabled="w.busy.value || !loaded">
        <legend>Horarios semanales</legend>
        <label class="ordering-check"
          ><input v-model="form.schedule_enabled" type="checkbox" /> Recibir
          pedidos solo dentro de estos horarios</label
        >
        <label
          >Zona horaria<input
            v-model="form.timezone"
            required
            maxlength="80"
            placeholder="America/Argentina/Buenos_Aires"
        /></label>
        <div
          v-for="(period, i) in form.periods"
          :key="i"
          class="form-row ordering-period"
        >
          <label
            >Día<select v-model.number="period.day">
              <option v-for="(day, index) in days" :key="day" :value="index">
                {{ day }}
              </option>
            </select></label
          >
          <label
            >Apertura<input
              v-model="period.opens"
              required
              pattern="[0-2][0-9]:[0-5][0-9]"
              placeholder="19:00"
              maxlength="5"
          /></label>
          <label
            >Cierre<input
              v-model="period.closes"
              required
              pattern="([0-2][0-9]:[0-5][0-9])"
              placeholder="01:00"
              maxlength="5"
          /></label>
          <button
            type="button"
            class="btn-secondary"
            :aria-label="'Quitar horario ' + days[period.day]"
            @click="form.periods.splice(i, 1)"
          >
            Quitar
          </button>
        </div>
        <p v-if="!form.periods.length">Todavía no agregaste horarios.</p>
        <button
          type="button"
          class="btn-secondary"
          :disabled="form.periods.length >= 28"
          @click="add"
        >
          Agregar horario
        </button>
        <p>
          Podés agregar varios turnos por día. Un cierre anterior a la apertura
          termina al día siguiente. Usá 24:00 para medianoche.
        </p>
      </fieldset>
      <button
        class="btn-primary"
        :disabled="w.busy.value || !w.tenant.value || !loaded"
      >
        {{ w.busy.value ? "Guardando…" : "Guardar atención del local" }}
      </button>
    </form>
  </WorkspaceFrame>
</template>
<style scoped>
fieldset {
  border: 1px solid var(--border-color, var(--border));
  border-radius: 12px;
  padding: 1.2rem;
  display: grid;
  gap: 1rem;
  min-width: 0;
}
legend {
  font-weight: 700;
  padding: 0 0.5rem;
}
.ordering-check {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}
.ordering-check input {
  width: auto;
}
.ordering-period {
  align-items: end;
}
</style>
