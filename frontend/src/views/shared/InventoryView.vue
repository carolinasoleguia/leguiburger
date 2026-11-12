<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import { ref, watch, onBeforeUnmount } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace, when, money } from "../../composables/useWorkspace.js";
const items = ref([]),
  movements = ref([]),
  offset = ref(0),
  tab = ref("stock"),
  search = ref(""),
  low = ref(false),
  kind = ref(""),
  selected = ref(null),
  delta = ref(""),
  reason = ref("purchase"),
  minimum = ref(""),
  cost = ref("");
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  movements.value = [];
  selected.value = null;
  if (!tenant) return;
  const data = await request(
    `${tab.value === "stock" ? "/inventory" : "/inventory/movements"}?limit=25&offset=${offset.value}&search=${encodeURIComponent(search.value)}&low=${low.value}&kind=${kind.value}`,
  );
  if (current())
    (tab.value === "stock" ? items : movements).value = data.items || [];
});
let searchTimer;
watch([search, low, kind], () => {
  offset.value = 0;
  clearTimeout(searchTimer);
  searchTimer = setTimeout(w.refresh, 300);
});
onBeforeUnmount(() => clearTimeout(searchTimer));

function select(item) {
  selected.value = item;
  minimum.value = String(item.minimum_stock);
  cost.value = "";
  delta.value = "";
}
function movement() {
  return w.act(async () => {
    await w.request("/inventory/movements", {
      item_kind: selected.value.kind,
      item_id: selected.value.id,
      quantity_delta: String(delta.value),
      expected_version: selected.value.stock_version,
      reason: reason.value,
    });
    selected.value = null;
  }, "Movimiento registrado con su historial.");
}
function settings() {
  return w.act(async () => {
    await w.request(
      `/inventory/${selected.value.kind}/${selected.value.id}/settings`,
      {
        minimum_stock: String(minimum.value),
        ...(cost.value !== "" ? { unit_cost: String(cost.value) } : {}),
      },
      "PATCH",
    );
    selected.value = null;
  }, "Mínimo y costo actualizados.");
}
watch(tab, () => {
  offset.value = 0;
  w.refresh();
});
watch(w.tenant, () => {
  offset.value = 0;
});
function page(n) {
  offset.value = n;
  w.refresh();
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Stock e insumos"
    description="Compras, producción, desperdicios y ajustes. Cada movimiento deja su registro."
  >
    <template #filters>
      <div class="operation-actions">
        <button
          class="btn-secondary"
          :aria-pressed="tab === 'stock'"
          @click="tab = 'stock'"
        >
          Existencias</button
        ><button
          class="btn-secondary"
          :aria-pressed="tab === 'movements'"
          @click="tab = 'movements'"
        >
          Historial
        </button>
      </div>
      <div v-if="tab === 'stock'" class="workspace-filters">
        <label
          >Buscar en todo el inventario<input
            v-model="search"
            type="search" /></label
        ><label
          ><span
            ><input v-model="low" type="checkbox" /> Sólo bajo mínimo</span
          ></label
        >
        <label
          >Tipo<select v-model="kind">
            <option value="">Todos</option>
            <option value="product">Productos</option>
            <option value="extra">Extras</option>
            <option value="supply">Insumos</option>
          </select></label
        >
      </div>
    </template>
    <template v-if="tab === 'stock'">
      <div class="card table-scroll">
        <table class="data-table">
          <thead>
            <tr>
              <th>Producto / insumo</th>
              <th>Stock</th>
              <th>Mínimo</th>
              <th>Seguimiento</th>
              <th v-if="w.manager.value">Costo unitario</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <LoadingRows
              v-if="w.loading.value"
              :columns="w.manager.value ? 6 : 5"
            />
            <tr
              v-show="!w.loading.value"
              v-for="item in items"
              :key="item.kind + item.id"
            >
              <td>
                {{ item.name
                }}<small>{{
                  { product: "Producto", extra: "Extra", supply: "Insumo" }[
                    item.kind
                  ]
                }}</small>
              </td>
              <td
                :class="{
                  'stock-low':
                    Number(item.current_stock) <= Number(item.minimum_stock),
                }"
              >
                {{ item.current_stock }} {{ item.unit }}
              </td>
              <td>{{ item.minimum_stock }}</td>
              <td>{{ item.track_stock ? "Activo" : "Sin descontar stock" }}</td>
              <td v-if="w.manager.value">{{ money(item.unit_cost) }}</td>
              <td>
                <button
                  v-if="w.manager.value"
                  class="btn-secondary"
                  :disabled="w.busy.value"
                  @click="select(item)"
                >
                  Gestionar
                </button>
              </td>
            </tr>
            <tr v-show="!w.loading.value" v-if="!items.length">
              <td :colspan="w.manager.value ? 6 : 5">
                No hay resultados para estos filtros.
              </td>
            </tr>
          </tbody>
        </table>
      </div></template
    >
    <div v-else class="card table-scroll">
      <table class="data-table">
        <thead>
          <tr>
            <th>Fecha</th>
            <th>Tipo</th>
            <th>Motivo</th>
            <th>Variación</th>
            <th>Antes → Después</th>
          </tr>
        </thead>
        <tbody>
          <LoadingRows v-if="w.loading.value" :columns="5" />
          <tr
            v-show="!w.loading.value"
            v-for="item in movements"
            :key="item.id"
          >
            <td>{{ when(item.occurred_at) }}</td>
            <td>{{ item.item_kind }}</td>
            <td>{{ item.reason }}</td>
            <td>{{ item.quantity_delta }}</td>
            <td>{{ item.balance_before }} → {{ item.balance_after }}</td>
          </tr>
          <tr v-show="!w.loading.value" v-if="!movements.length">
            <td colspan="5">Todavía no hay movimientos.</td>
          </tr>
        </tbody>
      </table>
    </div>
    <ListPager
      :offset="offset"
      :count="tab === 'stock' ? items.length : movements.length"
      :disabled="w.busy.value"
      @change="page"
    />
    <template v-if="selected"
      ><form v-form-care class="card operation-form" @submit.prevent="movement">
        <h2>{{ selected.name }}</h2>
        <p>
          Stock actual: {{ selected.current_stock }}. Para una salida, ingresá
          una cantidad negativa.
        </p>
        <div class="form-row">
          <label
            >Cantidad a sumar o restar<input
              v-model="delta"
              type="text"
              inputmode="decimal"
              pattern="-?[0-9]+([.][0-9]{1,4})?"
              required /></label
          ><label
            >Motivo<select v-model="reason">
              <option value="purchase">Compra</option>
              <option value="production">Producción</option>
              <option value="waste">Desperdicio</option>
              <option value="return">Devolución</option>
              <option value="adjustment">Ajuste</option>
            </select></label
          >
        </div>
        <div class="operation-actions">
          <button class="btn-primary" :disabled="w.busy.value">
            Registrar movimiento</button
          ><button type="button" class="btn-secondary" @click="selected = null">
            Cerrar
          </button>
        </div>
      </form>
      <form v-form-care class="card operation-form" @submit.prevent="settings">
        <h3>Mínimo y costo</h3>
        <div class="form-row">
          <label
            >Stock mínimo<input
              v-model="minimum"
              type="text"
              inputmode="decimal"
              pattern="[0-9]+([.][0-9]{1,4})?"
              required /></label
          ><label v-if="selected.kind !== 'product'"
            >Nuevo costo unitario (opcional)<input
              v-model="cost"
              type="text"
              inputmode="decimal"
              pattern="[0-9]+([.][0-9]{1,2})?"
          /></label>
        </div>
        <p v-if="selected.kind === 'product'">
          El costo de los productos se calcula desde sus recetas.
        </p>
        <button class="btn-secondary" :disabled="w.busy.value">
          Guardar configuración
        </button>
      </form></template
    >
  </WorkspaceFrame>
</template>
