<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import { computed, ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import { useWorkspace } from "../../composables/useWorkspace.js";
const items = ref([]),
  businesses = ref([]),
  editing = ref(null),
  brand = ref(""),
  subdomain = ref(""),
  name = ref(""),
  address = ref(""),
  phone = ref(""),
  hours = ref(""),
  active = ref(true);
const nextBusinessOffset = ref(100),
  moreBusinesses = ref(false);
const w = useWorkspace(async ({ request, platform, current }) => {
  const [data, biz] = await Promise.all([
    request("/business/locations"),
    request(platform ? "/platform/restaurants?limit=100" : "/business"),
  ]);
  if (current()) {
    items.value = data.items || [];
    businesses.value = biz.items || [];
    nextBusinessOffset.value = 100;
    moreBusinesses.value = businesses.value.length === 100;
    if (!brand.value) brand.value = businesses.value[0]?.id || "";
    editing.value = null;
  }
});
async function loadMoreBusinesses() {
  w.busy.value = true;
  w.error.value = "";
  try {
    const data = await w.request(
      `/platform/restaurants?limit=100&offset=${nextBusinessOffset.value}`,
    );
    businesses.value.push(...(data.items || []));
    nextBusinessOffset.value += 100;
    moreBusinesses.value =
      data.items?.length === 100 && nextBusinessOffset.value <= 10000;
  } catch (e) {
    w.error.value = e.message;
  } finally {
    w.busy.value = false;
  }
}
const shown = computed(() =>
  items.value.filter((x) => !w.platform.value || x.brand_id === brand.value),
);
function edit(item) {
  editing.value = item;
  brand.value = item.brand_id;
  subdomain.value = item.subdomain;
  name.value = item.display_name;
  address.value = item.address;
  phone.value = item.contact_phone;
  hours.value = item.service_hours;
  active.value = item.active;
}
function clear() {
  editing.value = null;
  subdomain.value = "";
  name.value = "";
  address.value = "";
  phone.value = "";
  hours.value = "";
  active.value = true;
}
function save() {
  return w.act(async () => {
    await w.request(
      "/business/locations" + (editing.value ? `/${editing.value.id}` : ""),
      {
        brand_id: brand.value,
        subdomain: subdomain.value,
        display_name: name.value,
        address: address.value,
        contact_phone: phone.value,
        service_hours: hours.value,
        active: active.value,
      },
      editing.value ? "PATCH" : "POST",
    );
    clear();
  }, "Local guardado.");
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    :local="false"
    title="Locales del restaurante"
    description="Sucursales, contacto y horarios de atención de tu negocio."
    ><div v-if="w.platform.value" class="workspace-filters">
      <label
        >Restaurante<select v-model="brand">
          <option v-for="b in businesses" :key="b.id" :value="b.id">
            {{ b.name }}
          </option>
        </select></label
      ><button
        v-if="moreBusinesses"
        class="btn-secondary"
        :disabled="w.busy.value"
        @click="loadMoreBusinesses"
      >
        Cargar más restaurantes
      </button>
    </div>
    <div class="card table-scroll">
      <table class="data-table">
        <thead>
          <tr>
            <th>Local</th>
            <th>Dirección</th>
            <th>Contacto</th>
            <th>Horario</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody><LoadingRows v-if="w.loading.value" :columns="6" />
          <tr v-show="!(w.loading.value)" v-for="item in shown" :key="item.id">
            <td>
              {{ item.display_name || item.subdomain
              }}<small>{{ item.business_name }}</small>
            </td>
            <td>{{ item.address || "Sin dirección" }}</td>
            <td>{{ item.contact_phone || "Sin teléfono" }}</td>
            <td>{{ item.service_hours || "Sin horario" }}</td>
            <td>{{ item.active ? "Activo" : "Inactivo" }}</td>
            <td>
              <button class="btn-secondary" @click="edit(item)">Editar</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <form v-form-care class="card operation-form" @submit.prevent="save">
      <h2>{{ editing ? "Editar local" : "Nuevo local" }}</h2>
      <div class="form-row">
        <label
          >Identificador<input
            v-model="subdomain"
            pattern="[a-z0-9][a-z0-9-]{0,62}"
            maxlength="63"
            required /></label
        ><label>Nombre visible<input v-model="name" maxlength="100" /></label
        ><label>Dirección<input v-model="address" maxlength="500" /></label
        ><label>Teléfono<input v-model="phone" maxlength="30" /></label
        ><label>Horarios<input v-model="hours" maxlength="1000" /></label
        ><label
          >Estado<select v-model="active">
            <option :value="true">Activo</option>
            <option :value="false">Inactivo</option>
          </select></label
        >
      </div>
      <p>
        Desactivar un local conserva su historial y cierra las sesiones que
        operan en esa sucursal.
      </p>
      <div class="operation-actions">
        <button class="btn-primary" :disabled="w.busy.value || !brand">
          Guardar local</button
        ><button
          v-if="editing"
          type="button"
          class="btn-secondary"
          @click="clear"
        >
          Cancelar
        </button>
      </div>
    </form></WorkspaceFrame
  >
</template>
