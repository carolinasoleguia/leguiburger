<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import StableContent from "../../components/StableContent.vue";
import { onMounted, ref } from "vue";
import DashboardShell from "../../components/DashboardShell.vue";
import { useAuth } from "../../composables/useAuth.js";
import { apiFetch, getJSON, postJSON } from "../../services/api.js";
const auth = useAuth(),
  tenants = ref([]),
  invitations = ref([]),
  error = ref(""),
  link = ref(""),
  busy = ref(false),
  loading = ref(true);
const email = ref(""),
  role = ref("employee"),
  tenant = ref("");
const shellRole = auth.userRole.value === "admin" ? "admin" : "owner";
const roles = { employee: "Colaborador", cashier: "Caja", kitchen: "Cocina" };
async function load() {
  loading.value = true;
  try {
    [tenants.value, invitations.value] = await Promise.all([
      getJSON("/tenants"),
      getJSON("/team/invitations"),
    ]);
    tenants.value = tenants.value.filter((item) => item.active);
    if (!tenant.value) tenant.value = tenants.value[0]?.id || "";
  } catch {
    error.value = "No pudimos cargar las invitaciones";
  } finally {
    loading.value = false;
  }
}
async function invite() {
  error.value = "";
  link.value = "";
  busy.value = true;
  try {
    const result = await postJSON("/team/invitations", {
      email: email.value,
      tenant_id: tenant.value,
      role: role.value,
    });
    link.value = result.invitation_url;
    email.value = "";
    await load();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
async function revoke(item) {
  error.value = "";
  busy.value = true;
  try {
    const response = await apiFetch(`/team/invitations/${item.id}`, {
      method: "DELETE",
    });
    if (!response.ok) throw new Error("No se pudo revocar la invitación");
    await load();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
onMounted(load);
</script>

<template>
  <DashboardShell :role="shellRole"
    ><section>
      <div class="section-header">
        <div class="section-title">
          <h2>Invitaciones al equipo</h2>
          <p>
            Cada integrante elige su propia contraseña. Las invitaciones vencen
            en 48 horas.
          </p>
        </div>
      </div>
      <p v-if="error" role="alert" class="error-msg">{{ error }}</p>
      <div class="card invite-card">
        <form v-form-care @submit.prevent="invite">
          <div class="input-group">
            <label for="invite-email">Correo del integrante</label
            ><input
              id="invite-email"
              v-model="email"
              type="email"
              required
              autocomplete="off"
            />
          </div>
          <div class="input-group">
            <label for="invite-tenant">Local</label
            ><select id="invite-tenant" v-model="tenant" required>
              <option v-for="item in tenants" :key="item.id" :value="item.id">
                {{ item.brand?.name }} · {{ item.subdomain }}
              </option>
            </select>
          </div>
          <div class="input-group">
            <label for="invite-role">Rol</label
            ><select id="invite-role" v-model="role">
              <option
                v-for="(label, value) in roles"
                :key="value"
                :value="value"
              >
                {{ label }}
              </option>
            </select>
          </div>
          <button class="btn-primary" :disabled="busy || loading || !tenant">
            Crear invitación
          </button>
        </form>
        <div v-if="link" class="invite-link" role="status">
          <label for="invitation-link"
            >Invitación lista. Copiá este enlace y compartilo con la
            persona:</label
          ><input
            id="invitation-link"
            :value="link"
            readonly
            @focus="$event.target.select()"
          /><small
            >Este enlace permite crear la cuenta. Compartilo únicamente con su
            destinatario.</small
          >
        </div>
      </div>
      <StableContent :loading="loading" label="Cargando invitaciones">
        <div class="card invite-table">
          <table class="data-table">
            <thead>
              <tr>
                <th>Correo</th>
                <th>Local</th>
                <th>Rol</th>
                <th>Estado</th>
                <th></th>
              </tr>
            </thead>
            <tbody><LoadingRows v-if="loading" :columns="5" />
              <tr v-show="!(loading)" v-for="item in invitations" :key="item.id">
                <td>{{ item.email }}</td>
                <td>{{ item.subdomain }}</td>
                <td>{{ roles[item.role] }}</td>
                <td>
                  {{
                    item.used_at
                      ? "Finalizada"
                      : Date.parse(item.expires_at) < Date.now()
                        ? "Vencida"
                        : "Pendiente"
                  }}
                </td>
                <td>
                  <button
                    v-if="
                      !item.used_at && Date.parse(item.expires_at) > Date.now()
                    "
                    class="btn-secondary"
                    :disabled="busy"
                    @click="revoke(item)"
                  >
                    Revocar
                  </button>
                </td>
              </tr>
              <tr v-show="!(loading)" v-if="!invitations.length">
                <td colspan="5">Todavía no hay invitaciones.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </StableContent>
    </section></DashboardShell
  >
</template>

<style scoped>
.invite-card {
  padding: 24px;
  margin-bottom: 24px;
}
form {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr auto;
  gap: 16px;
  align-items: end;
}
.invite-link {
  margin-top: 24px;
}
.invite-link input {
  display: block;
  width: 100%;
  margin: 12px 0;
}
.invite-table {
  overflow-x: auto;
}
small {
  color: var(--text-muted);
}
@media (max-width: 1000px) {
  form {
    grid-template-columns: 1fr;
  }
}
</style>
