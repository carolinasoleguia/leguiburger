<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import { ref, watch } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import StatePanel from "../../components/StatePanel.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
import { apiFetch } from "../../services/api.js";
const tickets = ref([]),
  messages = ref([]),
  backups = ref([]),
  businesses = ref([]),
  issues = ref([]),
  closure = ref(null);
const current = ref(null),
  brand = ref(""),
  subject = ref(""),
  body = ref(""),
  reply = ref(""),
  password = ref(""),
  confirmation = ref(""),
  reason = ref("");
const accountPassword = ref(""),
  accountConfirmation = ref("");
const scope = () =>
  brand.value ? `?brand_id=${encodeURIComponent(brand.value)}` : "";
const w = useWorkspace(
  async ({ request, platform, manager, current: isCurrent }) => {
    const data = await request("/care/support");
    if (!isCurrent()) return;
    tickets.value = data;
    if (manager) {
      businesses.value =
        (
          await request(
            platform ? "/platform/restaurants?limit=100" : "/business",
          )
        ).items || [];
      if (!brand.value) brand.value = businesses.value[0]?.id || "";
      if (brand.value) {
        const [b, c] = await Promise.all([
          request(`/care/backups${scope()}`),
          request(`/care/closure${scope()}`),
        ]);
        if (isCurrent()) {
          backups.value = b;
          closure.value = c;
        }
      }
    }
    if (platform) issues.value = await request("/care/errors");
    if (current.value)
      messages.value = await request(`/care/support/${current.value.id}`);
  },
);
watch(brand, (next, old) => {
  if (old && next !== old) {
    closure.value = null;
    backups.value = [];
    w.refresh();
  }
});
function createTicket() {
  return w.act(async () => {
    const item = await w.request("/care/support", {
      subject: subject.value,
      body: body.value,
    });
    current.value = { id: item.id, subject: subject.value };
    subject.value = "";
    body.value = "";
  }, "Consulta enviada. La respuesta quedará en este historial.");
}
function openTicket(item) {
  return w.act(async () => {
    const data = await w.request(`/care/support/${item.id}`);
    current.value = item;
    messages.value = data;
    reply.value = "";
  }, "Consulta abierta.");
}
function sendReply() {
  return w.act(async () => {
    await w.request(`/care/support/${current.value.id}/messages`, {
      body: reply.value,
    });
    reply.value = "";
  }, "Respuesta guardada.");
}
function closeTicket() {
  return w.act(
    () => w.request(`/care/support/${current.value.id}/close`, {}),
    "Consulta resuelta. Podés volver a abrirla enviando otro mensaje.",
  );
}
function createBackup() {
  return w.act(
    () => w.request(`/care/backups${scope()}`, {}),
    "Respaldo validado. Se está guardando en el almacenamiento privado.",
  );
}
function closeBusiness() {
  return w.act(async () => {
    await w.request(`/care/closure${scope()}`, {
      password: password.value,
      confirmation: confirmation.value,
      reason: reason.value,
    });
    password.value = "";
    confirmation.value = "";
    reason.value = "";
  }, "Restaurante archivado. Revisá abajo la confirmación de las renovaciones pendientes.");
}
function download(item) {
  return w.act(async () => {
    const response = await apiFetch(`/care/backups/${item.id}/download`);
    if (!response.ok) {
      const e = await response.json().catch(() => ({}));
      throw new Error(e.message || "No pudimos descargar el respaldo");
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob),
      a = document.createElement("a");
    a.href = url;
    a.download = `leguiburger-${item.id}.json`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }, "Respaldo descargado. Guardalo en un lugar privado: contiene información del negocio.");
}
const backupState = {
  ready: "Disponible",
  pending: "Guardando en privado",
  review: "Requiere revisión",
};
async function closeAccount() {
  if (w.busy.value) return;
  w.busy.value = true;
  w.error.value = "";
  try {
    await w.request("/care/account-close", {
      password: accountPassword.value,
      confirmation: accountConfirmation.value,
    });
    accountPassword.value = "";
    await w.auth.logout("account-closed");
  } catch (e) {
    w.error.value = e.message;
  } finally {
    w.busy.value = false;
  }
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Soporte y continuidad"
    description="Consultas con historial, respaldos privados y gestión del cierre de tu negocio."
    :local="false"
  >
    <div class="care-grid">
      <div>
        <form
          v-form-care
          class="card operation-form"
          @submit.prevent="createTicket"
        >
          <h2>Nueva consulta</h2>
          <p>No incluyas contraseñas, tarjetas ni claves de acceso.</p>
          <label
            >Asunto<input
              v-model="subject"
              name="subject"
              required
              minlength="4"
              maxlength="160"
          /></label>
          <label
            >¿En qué podemos ayudarte?<textarea
              v-model="body"
              name="message"
              required
              minlength="4"
              maxlength="6000"
              rows="5"
            ></textarea>
          </label>
          <button class="btn-primary" :disabled="w.busy.value">
            Enviar consulta
          </button>
        </form>
        <section class="card care-card" aria-label="Consultas">
          <h2>
            {{ w.platform.value ? "Consultas recibidas" : "Mis consultas" }}
          </h2>
          <StatePanel
            v-if="!tickets.length"
            title="Sin consultas pendientes"
            description="Cuando necesites ayuda, tu conversación quedará guardada acá."
          />
          <button
            v-for="item in tickets"
            :key="item.id"
            class="care-ticket"
            :aria-pressed="current?.id === item.id"
            :disabled="w.busy.value"
            @click="openTicket(item)"
          >
            <span
              >{{ item.subject
              }}<small style="display: block">{{
                when(item.updated_at)
              }}</small></span
            ><small>{{
              item.state === "closed" ? "Resuelta" : "Abierta"
            }}</small>
          </button>
        </section>
      </div>
      <section class="card care-card" aria-label="Conversación">
        <h2>{{ current?.subject || "Tu conversación" }}</h2>
        <StatePanel
          v-if="!current"
          title="Todo en un mismo lugar"
          description="Elegí una consulta para ver las respuestas y continuar la conversación."
        />
        <template v-else>
          <article
            v-for="message in messages"
            :key="message.id"
            class="care-message"
            :class="{ 'care-message--platform': message.platform }"
          >
            <strong>{{
              message.platform ? "Equipo LeguiBurger" : "Consulta del negocio"
            }}</strong>
            <p>{{ message.body }}</p>
            <small>{{ when(message.created_at) }}</small>
          </article>
          <form v-form-care class="operation-form" @submit.prevent="sendReply">
            <label
              >Tu respuesta<textarea
                v-model="reply"
                required
                minlength="4"
                maxlength="6000"
                rows="4"
              />
            </label>
            <div class="operation-actions">
              <button class="btn-primary" :disabled="w.busy.value">
                Enviar respuesta</button
              ><button
                type="button"
                class="btn-secondary"
                :disabled="w.busy.value"
                @click="closeTicket"
              >
                Marcar resuelta
              </button>
            </div>
          </form>
        </template>
      </section>
    </div>
    <template v-if="w.manager.value">
      <div v-if="w.platform.value" class="workspace-filters">
        <label
          >Restaurante<select v-model="brand" :disabled="w.busy.value">
            <option v-for="b in businesses" :key="b.id" :value="b.id">
              {{ b.name }}
            </option>
          </select></label
        >
      </div>
      <section class="card care-card">
        <h2>Respaldos del restaurante</h2>
        <p>
          Exportación de datos operativos, sin contraseñas ni credenciales de
          cobro. Se valida una reconstrucción en tablas temporales y se guarda
          cifrada en un bucket privado. No reemplaza el backup completo de la
          base de datos.
        </p>
        <button
          class="btn-primary"
          :disabled="w.busy.value"
          @click="createBackup"
        >
          Crear respaldo verificado
        </button>
        <StatePanel
          v-if="!backups.length"
          title="Todavía no hay respaldos"
          description="Creá el primero para conservar una copia de los datos del restaurante."
        />
        <div v-else class="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Creado</th>
                <th>Estado</th>
                <th>Verificación</th>
                <th>Archivo</th>
              </tr>
            </thead>
            <tbody><LoadingRows v-if="w.loading.value" :columns="4" />
              <tr v-show="!(w.loading.value)" v-for="item in backups" :key="item.id">
                <td>{{ when(item.created_at) }}</td>
                <td>{{ backupState[item.state] }}</td>
                <td>
                  {{
                    item.report.verified
                      ? "Reconstrucción validada"
                      : "Pendiente"
                  }}<small>{{ when(item.verified_at) }}</small>
                </td>
                <td>
                  <button
                    class="btn-secondary"
                    :disabled="item.state !== 'ready' || w.busy.value"
                    @click="download(item)"
                  >
                    Descargar JSON
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
      <section v-if="closure" class="card care-card care-danger">
        <h2>
          {{
            closure.closed_at ? "Restaurante archivado" : "Cerrar restaurante"
          }}
        </h2>
        <p>
          El cierre detiene la operación del restaurante y revoca el acceso del
          equipo. Conserva el historial de ventas, pagos y stock. Tu cuenta
          personal permanece disponible para soporte, membresía y respaldos.
        </p>
        <p v-if="closure.closed_at" role="status">
          Cierre registrado el {{ when(closure.closed_at) }}.
        </p>
        <form
          v-else-if="closure.primary || w.platform.value"
          v-form-care
          class="operation-form"
          @submit.prevent="closeBusiness"
        >
          <label
            >Motivo<textarea
              v-model="reason"
              required
              minlength="4"
              maxlength="500"
            /></label
          ><label
            >Contraseña actual<input
              v-model="password"
              type="password"
              autocomplete="current-password"
              required
              maxlength="72" /></label
          ><label
            >Escribí CERRAR RESTAURANTE<input
              v-model="confirmation"
              required
              pattern="CERRAR RESTAURANTE"
              autocomplete="off" /></label
          ><button class="btn-danger" :disabled="w.busy.value">
            Archivar restaurante y solicitar baja de renovaciones
          </button>
        </form>
        <p v-else>El responsable principal puede solicitar el cierre.</p>
        <div
          v-if="closure.closed_at && closure.renewals.length"
          class="table-scroll"
        >
          <table>
            <thead>
              <tr>
                <th>Membresía</th>
                <th>Renovación</th>
              </tr>
            </thead>
            <tbody><LoadingRows v-if="w.loading.value" :columns="2" />
              <tr v-show="!(w.loading.value)" v-for="renewal in closure.renewals" :key="renewal.id">
                <td>{{ renewal.provider || "Gestión manual" }}</td>
                <td>
                  {{
                    renewal.provider_status === "cancelled"
                      ? "Baja confirmada por el proveedor"
                      : renewal.cancel_pending
                        ? "Pendiente de confirmación del proveedor"
                        : renewal.cancel_at_period_end
                          ? "Sin renovación al finalizar el período"
                          : "Revisar membresía"
                  }}<small v-if="renewal.operation_state === 'review'"
                    >El resultado requiere conciliación. Contactá a
                    soporte.</small
                  >
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
    <section v-if="w.platform.value" class="card care-card">
      <h2>Errores de la aplicación</h2>
      <p>
        Eventos agrupados por categoría. No se registran mensajes de
        formularios, contraseñas, datos personales ni URLs privadas.
      </p>
      <StatePanel v-if="!issues.length" title="Sin errores registrados" />
      <div v-else class="table-scroll">
        <table>
          <thead>
            <tr>
              <th>Origen</th>
              <th>Categoría</th>
              <th>Ocurrencias</th>
              <th>Última vez</th>
            </tr>
          </thead>
          <tbody><LoadingRows v-if="w.loading.value" :columns="4" />
            <tr v-show="!(w.loading.value)" v-for="item in issues" :key="`${item.component}:${item.code}`">
              <td>{{ item.component }}</td>
              <td>{{ item.code }}</td>
              <td>{{ item.occurrences }}</td>
              <td>{{ when(item.last_seen_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <section class="card care-card">
      <h2>Dar de baja mi cuenta</h2>
      <p>
        Desactiva tu acceso personal y revoca tus sesiones. El historial del
        negocio se conserva. Si sos responsable de un restaurante, primero debés
        transferirlo o cerrarlo.
      </p>
      <details>
        <summary class="text-link">Quiero desactivar mi cuenta</summary>
        <form v-form-care class="operation-form" @submit.prevent="closeAccount">
          <label
            >Contraseña de tu cuenta<input
              v-model="accountPassword"
              type="password"
              autocomplete="current-password"
              required
              maxlength="72" /></label
          ><label
            >Escribí CERRAR CUENTA<input
              v-model="accountConfirmation"
              required
              pattern="CERRAR CUENTA"
              autocomplete="off" /></label
          ><button class="btn-danger" :disabled="w.busy.value">
            Desactivar mi cuenta
          </button>
        </form>
      </details>
    </section>
  </WorkspaceFrame>
</template>
