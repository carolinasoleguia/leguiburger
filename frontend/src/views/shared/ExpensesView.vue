<script setup>
import LoadingRows from "../../components/LoadingRows.vue";
import { ref, watch } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace, money, when } from "../../composables/useWorkspace.js";
const items = ref([]),
  offset = ref(0),
  category = ref("other"),
  amount = ref(""),
  description = ref(""),
  date = ref(new Date().toISOString().slice(0, 10)),
  selected = ref(null),
  voidReason = ref("");
const labels = {
  payroll: "Sueldos",
  rent: "Alquiler",
  utilities: "Servicios",
  processing_fees: "Comisiones",
  delivery: "Envíos",
  marketing: "Marketing",
  maintenance: "Mantenimiento",
  other: "Otros",
};
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  selected.value = null;
  if (!tenant) return;
  const data = await request(`/expenses?limit=25&offset=${offset.value}`);
  if (current()) items.value = data.items || [];
});
function create() {
  return w.act(async () => {
    await w.request("/expenses", {
      category: category.value,
      amount: String(amount.value),
      currency: "ARS",
      incurred_at: `${date.value}T12:00:00-03:00`,
      description: description.value,
    });
    amount.value = "";
    description.value = "";
  }, "Gasto registrado. Ya se incluye en la analítica de ese período.");
}
function cancel() {
  return w.act(async () => {
    await w.request(`/expenses/${selected.value.id}/void`, {
      reason: voidReason.value,
    });
    selected.value = null;
    voidReason.value = "";
  }, "Gasto anulado. El historial se conserva.");
}
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
    title="Gastos del local"
    description="Conocé cuánto cuesta sostener el servicio y conservá el historial."
    ><form v-form-care class="card operation-form" @submit.prevent="create">
      <h2>Registrar un gasto</h2>
      <div class="form-row">
        <label
          >Categoría<select v-model="category">
            <option v-for="(name, key) in labels" :key="key" :value="key">
              {{ name }}
            </option>
          </select></label
        ><label
          >Importe en pesos<input
            v-model="amount"
            type="text"
            inputmode="decimal"
            pattern="[0-9]+([.][0-9]{1,2})?"
            required /></label
        ><label>Fecha<input v-model="date" type="date" required /></label>
      </div>
      <label
        >Descripción<textarea v-model="description" maxlength="2000" /></label
      ><button class="btn-primary" :disabled="w.busy.value || !w.tenant.value">
        Guardar gasto
      </button>
    </form>
    <div class="card table-scroll">
      <table class="data-table">
        <thead>
          <tr>
            <th>Fecha</th>
            <th>Categoría</th>
            <th>Detalle</th>
            <th>Importe</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody><LoadingRows v-if="w.loading.value" :columns="6" />
          <tr v-show="!(w.loading.value)" v-for="item in items" :key="item.id">
            <td>{{ when(item.incurred_at) }}</td>
            <td>{{ labels[item.category] }}</td>
            <td>{{ item.description }}</td>
            <td>{{ money(item.amount, item.currency) }}</td>
            <td>
              {{ item.voided_at ? "Anulado" : "Vigente"
              }}<small v-if="item.voided_at">{{ item.void_reason }}</small>
            </td>
            <td>
              <button
                v-if="!item.voided_at"
                class="btn-secondary"
                @click="selected = item"
              >
                Anular
              </button>
            </td>
          </tr>
          <tr v-show="!(w.loading.value)" v-if="!items.length">
            <td colspan="6">Todavía no hay gastos registrados.</td>
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
    <form v-form-care v-if="selected" class="card operation-form" @submit.prevent="cancel">
      <h3>Anular {{ money(selected.amount, selected.currency) }}</h3>
      <label
        >Motivo<input
          v-model="voidReason"
          minlength="3"
          maxlength="500"
          required
      /></label>
      <div class="operation-actions">
        <button class="btn-danger" :disabled="w.busy.value">
          Confirmar anulación</button
        ><button type="button" class="btn-secondary" @click="selected = null">
          Cancelar
        </button>
      </div>
    </form></WorkspaceFrame
  >
</template>
