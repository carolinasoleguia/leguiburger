<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import { ref, watch } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
const items = ref([]),
  search = ref(""),
  offset = ref(0),
  editing = ref(""),
  first = ref(""),
  last = ref(""),
  email = ref(""),
  phone = ref("");
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  if (!tenant) return;
  const data = await request(
    `/crm/customers?limit=25&offset=${offset.value}&search=${encodeURIComponent(search.value)}`,
  );
  if (current()) items.value = data.items || [];
});
function edit(item) {
  editing.value = item.id;
  first.value = item.first_name;
  last.value = item.last_name;
  email.value = item.email;
  phone.value = item.phone;
}
function clear() {
  editing.value = "";
  first.value = "";
  last.value = "";
  email.value = "";
  phone.value = "";
}
function save() {
  return w.act(async () => {
    await w.request(
      "/customers" + (editing.value ? `/${editing.value}` : ""),
      {
        first_name: first.value,
        last_name: last.value,
        email: email.value,
        phone: phone.value,
      },
      editing.value ? "PUT" : "POST",
    );
    clear();
  }, "Cliente guardado.");
}
watch(w.tenant, () => {
  offset.value = 0;
  clear();
});
function page(n) {
  offset.value = n;
  w.refresh();
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Clientes"
    description="La gente que vuelve por tus hamburguesas, en un mismo lugar."
    ><form v-form-care
      class="workspace-filters"
      @submit.prevent="
        offset = 0;
        w.refresh();
      "
    >
      <label
        >Nombre o correo<input
          v-model="search"
          type="search"
          maxlength="100" /></label
      ><button class="btn-secondary">Buscar</button>
    </form>
    <div class="card table-scroll">
      <table class="data-table">
        <thead>
          <tr>
            <th>Cliente</th>
            <th>Contacto</th>
            <th>Pedidos</th>
            <th>Último pedido</th>
            <th></th>
          </tr>
        </thead>
        <tbody><LoadingRows v-if="w.loading.value" :columns="5" />
          <tr v-show="!(w.loading.value)" v-for="item in items" :key="item.id">
            <td>{{ item.first_name }} {{ item.last_name }}</td>
            <td>
              {{ item.email }}<small>{{ item.phone }}</small>
            </td>
            <td>{{ item.orders }}</td>
            <td>{{ when(item.last_order_at) }}</td>
            <td>
              <button class="btn-secondary" @click="edit(item)">Editar</button>
            </td>
          </tr>
          <tr v-show="!(w.loading.value)" v-if="!items.length">
            <td colspan="5">No encontramos clientes con ese criterio.</td>
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
    <form v-form-care class="card operation-form" @submit.prevent="save">
      <h2>{{ editing ? "Editar cliente" : "Nuevo cliente" }}</h2>
      <div class="form-row">
        <label>Nombre<input v-model="first" maxlength="50" required /></label
        ><label>Apellido<input v-model="last" maxlength="50" required /></label
        ><label
          >Correo<input
            v-model="email"
            type="email"
            maxlength="150"
            required /></label
        ><label
          >Teléfono<input v-model="phone" type="tel" maxlength="20"
        /></label>
      </div>
      <div class="operation-actions">
        <button class="btn-primary" :disabled="w.busy.value || !w.tenant.value">
          Guardar cliente</button
        ><button
          v-if="editing"
          type="button"
          class="btn-secondary"
          @click="clear"
        >
          Cancelar edición
        </button>
      </div>
    </form></WorkspaceFrame
  >
</template>
