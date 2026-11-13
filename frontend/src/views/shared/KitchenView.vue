<script setup>
import { ref, watch } from "vue";
import { useOrderStream } from "../../composables/useOrderStream.js";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import {
  useWorkspace,
  statusLabels,
  when,
} from "../../composables/useWorkspace.js";
const load = ref({
  active_count: 0,
  max_active_orders: 0,
  full: false,
  prep_min: 20,
  prep_max: 35,
});
const items = ref([]),
  updated = ref("");
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  if (!tenant) return;
  const [data, queue] = await Promise.all([
    request("/kitchen"),
    request("/kitchen/load"),
  ]);
  if (current()) {
    items.value = data.items || [];
    load.value = queue;
    updated.value = when(new Date());
  }
});
let dirty = false;
async function refreshKitchen() {
  if (w.busy.value || w.loading.value) {
    dirty = true;
    return;
  }
  const tenant = w.tenant.value;
  try {
    const [data, queue] = await Promise.all([
      w.request("/kitchen"),
      w.request("/kitchen/load"),
    ]);
    if (tenant === w.tenant.value) {
      items.value = data.items || [];
      load.value = queue;
      updated.value = when(new Date());
    }
  } catch (e) {
    w.error.value = e.message;
  }
}
const { live } = useOrderStream({
  path: "/api/orders/events",
  token: () => w.auth.state.token,
  tenant: () => w.tenant.value,
  headers: () => ({ "X-Tenant-ID": w.tenant.value }),
  enabled: () => !!w.tenant.value,
  refresh: refreshKitchen,
});
watch([w.busy, w.loading], () => {
  if (dirty && !w.busy.value && !w.loading.value) {
    dirty = false;
    refreshKitchen();
  }
});
function advance(item, status) {
  return w.act(
    () =>
      w.request(
        `/kitchen/${item.id}/status`,
        { status, expected_version: item.version },
        "PATCH",
      ),
    "Preparación actualizada.",
  );
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="La cocina"
    description="Cada pedido, en su punto. Los pedidos se actualizan en vivo."
    ><p class="kitchen-update">
      <span>{{ live ? "En vivo · " : "Actualización automática · " }}</span>
      {{
        updated
          ? `Última actualización: ${updated} · Hasta 100 pedidos activos.`
          : "Consultando los pedidos activos…"
      }}
    </p>
    <section class="card operation-form kitchen-load">
      <h2>{{ load.full ? "Cocina al máximo" : "Carga de cocina" }}</h2>
      <p>
        {{ load.active_count || 0 }} pedidos pendientes o en
        preparación<template v-if="load.max_active_orders">
          de {{ load.max_active_orders }} admitidos</template
        ><template v-else> · sin límite configurado</template>.
      </p>
      <p>
        Demora estimada para nuevos pedidos: {{ load.prep_min }}–{{
          load.prep_max
        }}
        min.
      </p>
      <p class="kitchen-load-status" role="status">
        {{
          load.full
            ? "Las nuevas compras están pausadas hasta que un pedido quede listo, se entregue o se cancele."
            : "Los pedidos listos o cancelados liberan lugar para nuevas compras."
        }}
      </p>
    </section>
    <div class="kitchen-board">
      <section
        v-for="status in ['accepted', 'preparing', 'ready']"
        :key="status"
        class="kitchen-column"
      >
        <h2>
          {{ statusLabels[status] }}
          <small>({{ items.filter((x) => x.status === status).length }})</small>
        </h2>
        <article
          v-for="item in items.filter((x) => x.status === status)"
          :key="item.id"
          class="card kitchen-ticket"
        >
          <h3>#{{ item.id.slice(0, 8) }}</h3>
          <small>{{ when(item.created_at) }}</small><router-link
            class="text-button"
            target="_blank"
            rel="noopener"
            :to="`/orders/${item.id}/print?tenant=${w.tenant.value}&kind=kitchen`"
            >Imprimir comanda</router-link
          >
          <ul>
            <li v-for="(line, i) in item.items" :key="i">
              <strong>{{ line.quantity }} × {{ line.product_name }}</strong>
              <p v-if="line.note">Preferencias: {{ line.note }}</p>
              <div v-for="(extra, j) in line.extras" :key="j">
                + {{ extra.quantity }}
                {{ extra.group_name ? extra.group_name + ": " : ""
                }}{{ extra.name }} por unidad
              </div>
            </li>
          </ul>
          <button
            v-if="status !== 'ready'"
            class="btn-primary"
            :disabled="w.busy.value"
            @click="
              advance(item, status === 'accepted' ? 'preparing' : 'ready')
            "
          >
            {{ status === "accepted" ? "Empezar preparación" : "Marcar listo" }}
          </button>
          <p v-else>Esperando despacho desde caja.</p>
        </article>
        <p v-if="!items.some((x) => x.status === status)" class="no-data">
          Todo al día por acá.
        </p>
      </section>
    </div></WorkspaceFrame
  >
</template>

<style scoped>
.kitchen-load p {
  min-height: 3.2em;
}
.kitchen-load .kitchen-load-status {
  min-height: 4.8em;
}
@media (min-width: 800px) {
  .kitchen-load p,
  .kitchen-load .kitchen-load-status {
    min-height: 1.6em;
  }
}
</style>
