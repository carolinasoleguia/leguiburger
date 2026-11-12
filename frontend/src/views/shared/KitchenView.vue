<script setup>
import { ref, onBeforeUnmount } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import {
  useWorkspace,
  statusLabels,
  when,
} from "../../composables/useWorkspace.js";
const items = ref([]),
  updated = ref("");
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  if (!tenant) return;
  const data = await request("/kitchen");
  if (current()) {
    items.value = data.items || [];
    updated.value = when(new Date());
  }
});
const timer = setInterval(() => {
  if (!document.hidden && !w.busy.value && !w.loading.value) w.refresh();
}, 30000);
onBeforeUnmount(() => clearInterval(timer));
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
    description="Cada pedido, en su punto. La vista se actualiza cada 30 segundos."
    ><p class="kitchen-update">
      {{
        updated
          ? `Última actualización: ${updated} · Hasta 100 pedidos activos.`
          : "Consultando los pedidos activos…"
      }}
    </p>
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
          <small>{{ when(item.created_at) }}</small>
          <ul>
            <li v-for="(line, i) in item.items" :key="i">
              <strong>{{ line.quantity }} × {{ line.product_name }}</strong>
              <p v-if="line.note">Preferencias: {{ line.note }}</p>
              <div v-for="(extra, j) in line.extras" :key="j">
                + {{ extra.quantity }} {{ extra.name }}
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
