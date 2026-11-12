<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace } from "../../composables/useWorkspace.js";
const items = ref([]),
  locals = ref([]),
  offset = ref(0),
  selected = ref(null),
  first = ref(""),
  last = ref(""),
  role = ref("employee"),
  active = ref(true),
  assigned = ref([]),
  home = ref("");
const roles = { employee: "Colaborador", cashier: "Caja", kitchen: "Cocina" };
const w = useWorkspace(async ({ request, current }) => {
  const [people, locations] = await Promise.all([
    request(`/business/team?limit=25&offset=${offset.value}`),
    request("/business/locations"),
  ]);
  if (current()) {
    items.value = people.items || [];
    locals.value = locations.items || [];
    selected.value = null;
  }
});
function edit(item) {
  selected.value = item;
  first.value = item.first_name;
  last.value = item.last_name;
  role.value = item.role;
  active.value = item.is_active;
  assigned.value = [...item.location_ids];
  home.value = item.tenant_id;
}
function save() {
  return w.act(
    () =>
      w.request(
        `/business/team/${selected.value.id}`,
        {
          first_name: first.value,
          last_name: last.value,
          role: role.value,
          is_active: active.value,
          location_ids: assigned.value,
          tenant_id: home.value,
        },
        "PATCH",
      ),
    "Colaborador actualizado. Los accesos retirados dejan de funcionar inmediatamente.",
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
    title="Equipo del restaurante"
    description="Cada persona con su cuenta, su función y sus locales asignados."
    ><router-link to="/team/invitations" class="btn-primary"
      >Invitar colaborador</router-link
    >
    <div class="card table-scroll">
      <table class="data-table">
        <thead>
          <tr>
            <th>Persona</th>
            <th>Función</th>
            <th>Locales</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody><LoadingRows v-if="w.loading.value" :columns="5" />
          <tr v-for="item in items" :key="item.id">
            <td>
              {{ item.first_name }} {{ item.last_name
              }}<small>{{ item.email }}</small>
            </td>
            <td>{{ roles[item.role] || "Rol heredado" }}</td>
            <td>
              {{
                item.location_ids
                  .map(
                    (id) =>
                      locals.find((t) => t.id === id)?.display_name ||
                      locals.find((t) => t.id === id)?.subdomain ||
                      id,
                  )
                  .join(", ")
              }}
            </td>
            <td>{{ item.is_active ? "Activo" : "Inactivo" }}</td>
            <td>
              <button class="btn-secondary" @click="edit(item)">
                Administrar acceso
              </button>
            </td>
          </tr>
          <tr v-if="!items.length">
            <td colspan="5">
              Invitá a las personas que trabajan en tu restaurante.
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <ListPager
      :offset="offset"
      :count="items.length"
      :disabled="w.busy.value"
      @change="page"
    />
    <form v-form-care v-if="selected" class="card operation-form" @submit.prevent="save">
      <h2>{{ selected.email }}</h2>
      <div class="form-row">
        <label>Nombre<input v-model="first" maxlength="50" required /></label
        ><label>Apellido<input v-model="last" maxlength="50" required /></label
        ><label
          >Función<select v-model="role">
            <option v-for="(text, key) in roles" :key="key" :value="key">
              {{ text }}
            </option>
          </select></label
        ><label
          >Acceso<select v-model="active">
            <option :value="true">Activo</option>
            <option :value="false">Desactivado</option>
          </select></label
        >
      </div>
      <fieldset>
        <legend>Locales asignados</legend>
        <label
          v-for="item in locals.filter((t) => t.brand_id === selected.brand_id)"
          :key="item.id"
          ><span
            ><input v-model="assigned" type="checkbox" :value="item.id" />
            {{ item.display_name || item.subdomain }}
            {{ item.active ? "" : "(inactivo)" }}</span
          ></label
        >
      </fieldset>
      <label
        >Local principal<select v-model="home" required>
          <option v-for="id in assigned" :key="id" :value="id">
            {{
              locals.find((t) => t.id === id)?.display_name ||
              locals.find((t) => t.id === id)?.subdomain
            }}
          </option>
        </select></label
      >
      <div class="operation-actions">
        <button
          class="btn-primary"
          :disabled="
            w.busy.value || !assigned.length || !assigned.includes(home)
          "
        >
          Guardar accesos</button
        ><button class="btn-secondary" type="button" @click="selected = null">
          Cancelar
        </button>
      </div>
    </form></WorkspaceFrame
  >
</template>
