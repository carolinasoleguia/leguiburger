<script setup>
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
const items = ref([]),
  offset = ref(0),
  name = ref(""),
  tax = ref(""),
  local = ref("centro"),
  email = ref(""),
  locations = ref(1),
  collaborators = ref(5),
  selected = ref(null),
  inviteEmail = ref(""),
  password = ref(""),
  link = ref("");
const w = useWorkspace(async ({ request, platform, current }) => {
  const data = await request(
    platform
      ? `/platform/restaurants?limit=25&offset=${offset.value}`
      : "/business",
  );
  if (current()) {
    items.value = (data.items || []).map((x) => ({
      ...x,
      _suspended: !!x.suspended_at,
    }));
    selected.value = null;
  }
});
function create() {
  return w.act(async () => {
    const data = await w.request("/platform/restaurants", {
      name: name.value,
      tax_id: tax.value,
      local: local.value,
      email: email.value,
      base_location_limit: Number(locations.value),
      base_collaborator_limit: Number(collaborators.value),
    });
    link.value = data.invitation.invitation_url;
    name.value = "";
    email.value = "";
  }, "Restaurante y primer local creados. Compartí la invitación con su dueño.");
}
function invite() {
  return w.act(async () => {
    const data = await w.request(
      `/business/${selected.value.id}/owner-invitation`,
      { email: inviteEmail.value, current_password: password.value },
    );
    link.value = data.invitation_url;
    password.value = "";
  }, "Invitación creada. El responsable actual conserva su acceso hasta que la persona invitada acepte.");
}
function status(item) {
  return w.act(
    () =>
      w.request(
        `/platform/restaurants/${item.id}/status`,
        {
          suspended: item._suspended,
          reason: item.suspension_reason || "",
          base_location_limit: Number(item.base_location_limit),
          base_collaborator_limit: Number(item.base_collaborator_limit),
        },
        "PATCH",
      ),
    "Estado y límites guardados.",
  );
}
function revoke(item) {
  return w.act(() =>
    w.request(`/business/${item.id}/owner-invitation`, {}, "DELETE"),
  );
}
function page(n) {
  offset.value = n;
  w.refresh();
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    :local="false"
    :title="w.platform.value ? 'Restaurantes' : 'Mi restaurante'"
    description="Negocio, responsable principal, locales y capacidad del equipo."
  >
    <div v-if="link" class="card operation-form" role="status">
      <label
        >Invitación privada, válida por 48 horas<input
          :value="link"
          readonly
          @focus="$event.target.select()"
      /></label>
      <p>
        El destinatario elige su contraseña. También se envía por correo cuando
        SMTP está configurado.
      </p>
    </div>
    <form
      v-if="w.platform.value"
      class="card operation-form"
      @submit.prevent="create"
    >
      <h2>Dar de alta un restaurante</h2>
      <div class="form-row">
        <label
          >Nombre del negocio<input
            v-model="name"
            maxlength="100"
            minlength="2"
            required /></label
        ><label>CUIT (opcional)<input v-model="tax" maxlength="20" /></label
        ><label
          >Identificador del primer local<input
            v-model="local"
            pattern="[a-z0-9][a-z0-9-]{0,62}"
            maxlength="63"
            required /></label
        ><label
          >Correo del dueño<input
            v-model="email"
            type="email"
            required /></label
        ><label
          >Locales del acceso base<input
            v-model.number="locations"
            type="number"
            min="1"
            max="1000"
            required /></label
        ><label
          >Colaboradores del acceso base<input
            v-model.number="collaborators"
            type="number"
            min="1"
            max="10000"
            required
        /></label>
      </div>
      <button class="btn-primary" :disabled="w.busy.value">
        Crear restaurante e invitar dueño
      </button>
    </form>
    <article v-for="item in items" :key="item.id" class="card operation-form">
      <h2>{{ item.name }}</h2>
      <p>
        Responsable principal:
        <strong>{{ item.owner_email || "Pendiente de aceptación" }}</strong>
      </p>
      <p v-if="item.suspended_at" class="error-msg">
        Restaurante suspendido desde {{ when(item.suspended_at) }}.
        {{ item.suspension_reason }}
        <router-link to="/payments">Consultar membresía</router-link>
      </p>
      <div class="metric-grid">
        <article>
          <h3>Locales activos</h3>
          <strong
            >{{ item.locations }} /
            {{ item.maximum_locations ?? "Sin límite" }}</strong
          >
        </article>
        <article>
          <h3>Colaboradores activos</h3>
          <strong
            >{{ item.collaborators }} /
            {{ item.maximum_collaborators ?? "Sin límite" }}</strong
          >
        </article>
      </div>
      <p>
        Capacidad vigente:
        {{
          item.capacity_source === "subscription"
            ? "membresía contratada"
            : "acceso base"
        }}. Las invitaciones pendientes reservan cupo de colaborador.
      </p>
      <div v-if="w.platform.value" class="form-row">
        <label
          >Estado<select v-model="item._suspended">
            <option :value="false">Activo</option>
            <option :value="true">Suspendido</option>
          </select></label
        ><label
          >Motivo de suspensión<input
            v-model="item.suspension_reason"
            maxlength="500" /></label
        ><label
          >Límite base de locales<input
            v-model.number="item.base_location_limit"
            type="number"
            min="1"
            max="1000" /></label
        ><label
          >Límite base de colaboradores<input
            v-model.number="item.base_collaborator_limit"
            type="number"
            min="1"
            max="10000" /></label
        ><button
          class="btn-secondary"
          :disabled="w.busy.value"
          @click="status(item)"
        >
          Guardar estado y límites
        </button>
      </div>
      <button
        v-if="item.invitations.length"
        class="text-button"
        :disabled="w.busy.value"
        @click="revoke(item)"
      >
        Revocar invitaciones pendientes
      </button>
      <div v-for="inv in item.invitations" :key="inv.id" class="report-row">
        <span>Invitación pendiente: {{ inv.email }}</span
        ><small>Vence {{ when(inv.expires_at) }}</small>
      </div>
      <button
        v-if="
          !item.suspended_at &&
          (w.platform.value || item.primary_owner_id === w.auth.user.value?.id)
        "
        class="btn-secondary"
        :disabled="w.busy.value"
        @click="
          selected = item;
          inviteEmail = '';
          password = '';
        "
      >
        {{
          item.primary_owner_id
            ? "Transferir responsabilidad"
            : "Invitar responsable"
        }}
      </button>
    </article>
    <form v-if="selected" class="card operation-form" @submit.prevent="invite">
      <h2>
        {{
          selected.primary_owner_id ? "Transferir restaurante" : "Invitar dueño"
        }}
        · {{ selected.name }}
      </h2>
      <p>
        Al aceptar la transferencia, el responsable anterior pierde su acceso.
        El historial y los colaboradores se conservan.
      </p>
      <label
        >Correo del responsable<input
          v-model="inviteEmail"
          type="email"
          required /></label
      ><label v-if="!w.platform.value"
        >Tu contraseña actual<input
          v-model="password"
          type="password"
          autocomplete="current-password"
          required
      /></label>
      <div class="operation-actions">
        <button class="btn-primary" :disabled="w.busy.value">
          Generar invitación</button
        ><button class="btn-secondary" type="button" @click="selected = null">
          Cancelar
        </button>
      </div>
    </form>
    <ListPager
      v-if="w.platform.value"
      :offset="offset"
      :count="items.length"
      :disabled="w.busy.value"
      @change="page"
    />
  </WorkspaceFrame>
</template>
