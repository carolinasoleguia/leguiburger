<script setup>
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
import { confirmAction } from "../../services/confirmation.js";
const items = ref([]),
  configured = ref(false),
  preview = ref(null),
  subject = ref(""),
  body = ref(""),
  segment = ref("all");
const labels = {
  draft: "Borrador",
  queued: "En cola",
  completed: "Finalizada",
  canceled: "Cancelada",
};
const segments = {
  all: "Todos los suscriptores",
  frequent: "Frecuentes · 3 o más compras",
  recover: "Volver a comprar · más de 30 días",
  new: "Nuevos · hasta 1 compra",
};
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  preview.value = null;
  if (!tenant) return;
  const data = await request("/marketing/campaigns");
  if (current()) {
    items.value = data.items || [];
    configured.value = data.mail_configured;
  }
});
async function create() {
  await w.act(async () => {
    await w.request("/marketing/campaigns", {
      subject: subject.value,
      body: body.value,
      segment: segment.value,
    });
    subject.value = "";
    body.value = "";
  }, "Borrador guardado. Revisá el mensaje y sus destinatarios antes de enviarlo.");
}
async function review(id) {
  if (w.busy.value) return;
  w.busy.value = true;
  const tenant = w.tenant.value;
  try {
    const data = await w.request("/marketing/campaigns/" + id);
    if (tenant === w.tenant.value) preview.value = data;
  } catch (e) {
    w.error.value = e.message;
  } finally {
    w.busy.value = false;
  }
}
async function send() {
  const c = preview.value;
  if (
    !c ||
    !(await confirmAction(
      `Se enviará “${c.subject}” a los suscriptores confirmados de esta marca (actualmente ${c.eligible_count}). No se puede retirar un correo ya enviado.`,
      "Enviar campaña",
    ))
  )
    return;
  return w.act(
    () => w.request(`/marketing/campaigns/${c.id}/send`, {}),
    "Campaña en cola. Los contadores mostrarán las entregas confirmadas por el servidor de correo.",
  );
}
async function cancel(c) {
  if (
    !(await confirmAction(
      "Se detendrán los correos pendientes. Los que ya están en tránsito o entregados pueden llegar igualmente.",
      "Detener campaña",
    ))
  )
    return;
  return w.act(
    () => w.request(`/marketing/campaigns/${c.id}/cancel`, {}),
    "Campaña detenida.",
  );
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Novedades de tu marca"
    description="Mensajes para los compradores que eligieron recibir tus novedades."
  >
    <p v-if="!configured" class="inline-notice">
      El envío todavía no está habilitado. Configurá SMTP y la URL pública del
      sitio; podés preparar y revisar borradores mientras tanto.
    </p>
    <form v-form-care class="card operation-form" @submit.prevent="create">
      <h2>Preparar una campaña</h2>
      <label
        >Asunto<input
          v-model="subject"
          minlength="3"
          maxlength="120"
          required
          placeholder="Este finde, hay una nueva burger"
      /></label>
      <label
        >Audiencia<select v-model="segment">
          <option v-for="(label, key) in segments" :key="key" :value="key">
            {{ label }}
          </option>
        </select></label
      >
      <p>
        Solo se incluyen cuentas con email confirmado y suscripción activa a
        esta marca. Las audiencias por compras además requieren permiso de
        personalización.
      </p>
      <label
        >Mensaje<textarea
          v-model="body"
          rows="7"
          minlength="3"
          maxlength="4000"
          required
          placeholder="Contales qué estás preparando…"
        ></textarea>
      </label>
      <small
        >Se agrega automáticamente un enlace para dejar de recibir
        novedades.</small
      >
      <button class="btn-primary" :disabled="w.busy.value || !w.tenant.value">
        Guardar borrador
      </button>
    </form>
    <section
      v-if="preview"
      class="card operation-form"
      aria-labelledby="campaign-preview"
    >
      <h2 id="campaign-preview">{{ preview.subject }}</h2>
      <p>
        {{ segments[preview.segment] }} ·
        {{ preview.eligible_count }} suscriptores elegibles actualmente
      </p>
      <div class="campaign-body">{{ preview.body }}</div>
      <p>
        Estado: {{ labels[preview.state] }} · {{ preview.delivered }} entregados
        · {{ preview.pending }} pendientes · {{ preview.failed }} fallidos
      </p>
      <div class="operation-actions">
        <button
          v-if="preview.state === 'draft'"
          class="btn-primary"
          :disabled="
            w.busy.value ||
            !preview.mail_configured ||
            !preview.eligible_count ||
            preview.eligible_count > 5000
          "
          @click="send"
        >
          Enviar campaña</button
        ><button class="btn-secondary" @click="preview = null">
          Cerrar vista previa
        </button>
      </div>
    </section>
    <section class="card operation-form">
      <h2>Campañas de tu marca</h2>
      <p v-if="!items.length" class="no-data">
        Tu primera novedad empieza con un borrador.
      </p>
      <article v-for="c in items" :key="c.id" class="report-row campaign-row">
        <div>
          <strong>{{ c.subject }}</strong>
          <p>
            {{ labels[c.state] }} · {{ c.delivered }} de
            {{ c.recipients }} entregados · {{ when(c.created_at) }}
          </p>
        </div>
        <div class="operation-actions">
          <button
            class="btn-secondary"
            :disabled="w.busy.value"
            @click="review(c.id)"
          >
            Revisar</button
          ><button
            v-if="['draft', 'queued'].includes(c.state)"
            class="text-button"
            :disabled="w.busy.value"
            @click="cancel(c)"
          >
            Detener
          </button>
        </div>
      </article>
    </section>
  </WorkspaceFrame>
</template>
<style scoped>
.campaign-body {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  border: 1px solid var(--border-color, var(--border));
  border-radius: 12px;
  padding: 1.25rem;
  line-height: 1.7;
}
.campaign-row {
  gap: 1rem;
  flex-wrap: wrap;
}
.campaign-row p {
  margin: 0.4rem 0;
}
</style>
