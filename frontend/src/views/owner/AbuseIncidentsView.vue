<script setup>
import { onMounted, ref, watch } from "vue";
import DashboardShell from "../../components/DashboardShell.vue";
import PageHeading from "../../components/PageHeading.vue";
import StableContent from "../../components/StableContent.vue";
import LoadingRows from "../../components/LoadingRows.vue";
import ListPager from "../../components/ListPager.vue";
import { apiFetch, getJSON } from "../../services/api.js";
const data = ref({ items: [], total: 0, pending: 0, blocked: 0 }),
  loading = ref(true),
  busy = ref(false),
  error = ref(""),
  notice = ref("");
const filter = ref("open"),
  offset = ref(0),
  selected = ref(null),
  action = ref("resolve"),
  minutes = ref(15),
  note = ref("");
let revision = 0;
const keys = new Map();
const reasons = {
  session_churn: "Muchas sesiones nuevas",
  invalid_requests: "Solicitudes inválidas",
  repeated_orders: "Compras repetidas",
  rate_limit: "Exceso de intentos",
  captcha_failed: "Verificación rechazada",
  manual: "Revisión administrativa",
};
const when = (value) => (value ? new Date(value).toLocaleString("es-AR") : "—");
const blocked = (item) =>
  item.is_blocked ??
  (item.blocked_until && Date.parse(item.blocked_until) > Date.now());
async function load() {
  const current = ++revision;
  loading.value = true;
  error.value = "";
  try {
    const value = await getJSON(
      `/platform/security/incidents?status=${filter.value}&limit=25&offset=${offset.value}`,
    );
    if (current === revision) data.value = value;
  } catch (e) {
    if (current === revision) error.value = e.message;
  } finally {
    if (current === revision) loading.value = false;
  }
}
function select(item, operation) {
  selected.value = item;
  action.value = operation;
  note.value = "";
  minutes.value = 15;
  notice.value = "";
}
async function apply() {
  if (busy.value || !selected.value) return;
  const path = `/platform/security/incidents/${selected.value.id}/actions`;
  const body = JSON.stringify({
    action: action.value,
    minutes: action.value === "block" ? Number(minutes.value) : 0,
    note: note.value,
  });
  const operation = path + body;
  if (!keys.has(operation)) keys.set(operation, crypto.randomUUID());
  busy.value = true;
  error.value = "";
  try {
    const response = await apiFetch(path, {
      method: "POST",
      headers: { "Idempotency-Key": keys.get(operation) },
      body,
    });
    const result = await response.json();
    if (!response.ok)
      throw new Error(result.message || "No pudimos guardar la acción.");
    keys.delete(operation);
    selected.value = null;
    notice.value =
      "Acción registrada. El motivo quedó guardado en la actividad de seguridad.";
    await load();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
watch(filter, () => {
  offset.value = 0;
  load();
});
function page(value) {
  offset.value = value;
  load();
}
onMounted(load);
</script>
<template>
  <DashboardShell role="owner"
    ><section class="operations-page incidents-page">
      <PageHeading
        title="Seguridad de compras"
        description="Revisá actividad inusual, verificaciones y bloqueos temporales."
      >
        <button class="btn-secondary" :disabled="loading || busy" @click="load">
          Actualizar
        </button>
      </PageHeading>
      <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
      <p v-if="notice" class="status-notice" role="status">{{ notice }}</p>
      <StableContent
        :loading="loading"
        :busy="busy"
        label="Consultando incidentes de compra…"
      >
        <div class="metric-grid">
          <article class="card">
            <h3>Incidentes pendientes</h3>
            <strong>{{ data.pending }}</strong>
          </article>
          <article class="card">
            <h3>Orígenes bloqueados</h3>
            <strong>{{ data.blocked }}</strong>
          </article>
          <article class="card">
            <h3>Verificación adicional</h3>
            <strong class="verification-state">{{
              data.verification_configured ? "Configurada" : "Sin configurar"
            }}</strong><small v-if="!data.verification_configured"
              >Los intentos sospechosos quedan en pausa hasta que se habilite la
              verificación o venza su ventana.</small
            >
          </article>
        </div>
        <div class="workspace-filters">
          <label
            >Mostrar<select v-model="filter" :disabled="busy">
              <option value="open">Pendientes</option>
              <option value="blocked">Con bloqueo activo</option>
              <option value="all">Todos</option>
            </select></label
          >
        </div>
        <form
          v-if="selected"
          v-form-care
          class="card incident-action"
          @submit.prevent="apply"
        >
          <h2>
            {{
              action === "block"
                ? "Bloquear compras"
                : action === "unblock"
                  ? "Desbloquear compras"
                  : "Resolver incidente"
            }}
            · {{ selected.reference }}
          </h2>
          <p v-if="action === 'unblock'">
            Se retirará el bloqueo de este origen y se reiniciarán sus
            contadores. Las nuevas acciones abusivas podrán generar otro
            bloqueo.
          </p>
          <p v-else-if="action === 'resolve'">
            Resolver cierra la revisión; un bloqueo activo conserva su
            vencimiento. Usá Desbloquear para retirarlo.
          </p>
          <p v-else>
            El bloqueo afecta los intentos de compra de esta conexión, que
            podría ser compartida por varias personas. La carta y el seguimiento
            siguen disponibles.
          </p>
          <label v-if="action === 'block'"
            >Duración<select v-model="minutes">
              <option :value="15">15 minutos</option>
              <option :value="60">1 hora</option>
              <option :value="1440">24 horas</option>
            </select></label
          >
          <label
            >Motivo<textarea
              v-model="note"
              required
              minlength="10"
              maxlength="500"
              rows="2"
            />
          </label>
          <div class="form-actions">
            <button
              class="btn-primary"
              :disabled="busy || note.trim().length < 10"
            >
              Guardar acción</button
            ><button
              type="button"
              class="btn-secondary"
              :disabled="busy"
              @click="selected = null"
            >
              Cancelar
            </button>
          </div>
        </form>
        <div class="card table-scroll incident-table">
          <table class="data-table">
            <thead>
              <tr>
                <th>Origen</th>
                <th>Actividad</th>
                <th>Local</th>
                <th>Última señal</th>
                <th>Estado</th>
                <th>Acciones</th>
              </tr>
            </thead>
            <tbody>
              <LoadingRows v-if="loading" :columns="6" />
              <tr v-for="item in data.items" :key="item.id" v-show="!loading">
                <td>
                  <strong>{{ item.label }}</strong><small
                    >Referencia {{ item.reference }}</small
                  >
                </td>
                <td>
                  {{ reasons[item.reason] || item.reason
                  }}<small
                    >{{ item.occurrences }} señales ·
                    {{ item.evidence?.new_sessions || 0 }} sesiones ·
                    {{ item.evidence?.invalid_requests || 0 }} inválidas ·
                    {{ item.evidence?.orders || 0 }} compras</small
                  >
                </td>
                <td>
                  {{ item.business }}<small>{{ item.location }}</small>
                </td>
                <td>{{ when(item.last_seen_at) }}</td>
                <td>
                  {{
                    blocked(item)
                      ? "Bloqueado hasta " + when(item.blocked_until)
                      : item.resolved_at
                        ? "Resuelto"
                        : "Pendiente de revisión"
                  }}<small v-if="item.resolution_note">{{
                    item.resolution_note
                  }}</small>
                </td>
                <td>
                  <div class="incident-actions">
                    <button
                      class="btn-secondary"
                      :disabled="busy"
                      @click="select(item, 'unblock')"
                    >
                      Desbloquear</button
                    ><button
                      class="btn-secondary"
                      :disabled="busy"
                      @click="select(item, 'block')"
                    >
                      Bloquear</button
                    ><button
                      v-if="!item.resolved_at"
                      class="text-button"
                      :disabled="busy"
                      @click="select(item, 'resolve')"
                    >
                      Resolver
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="!data.items.length" v-show="!loading">
                <td colspan="6">No hay incidentes para este filtro.</td>
              </tr>
            </tbody>
          </table>
        </div>
        <ListPager
          :offset="offset"
          :count="data.items.length"
          :size="25"
          :total="data.total"
          :disabled="loading || busy"
          @change="page"
        />
      </StableContent></section
  ></DashboardShell>
</template>
<style scoped>
.incident-action {
  padding: 16px;
  margin: 12px 0;
  display: grid;
  gap: 12px;
}
.incident-action h2 {
  font-size: 18px;
  margin: 0;
}
.incident-action label {
  display: grid;
  gap: 6px;
}
.incident-action p {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}
.incident-table {
  min-height: 300px;
}
.incident-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.incident-actions button {
  min-height: 40px;
}
small {
  display: block;
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 12px;
}
.verification-state {
  font-size: 20px;
}
@media (max-width: 800px) {
  .incident-actions button {
    min-height: 44px;
  }
}
</style>
