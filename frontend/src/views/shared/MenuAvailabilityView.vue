<script setup>
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace } from "../../composables/useWorkspace.js";
const items = ref([]),
  search = ref(""),
  offset = ref(0);
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  if (!tenant) return;
  const data = await request(
    `/menu/availability?limit=25&offset=${offset.value}&search=${encodeURIComponent(search.value)}`,
  );
  if (current()) items.value = data.items || [];
});
function toggle(item) {
  return w.act(
    () =>
      w.request(
        `/menu/products/${item.id}/availability`,
        { enabled: !item.enabled },
        "PATCH",
      ),
    "Disponibilidad actualizada en la carta.",
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
    title="Disponibilidad de la carta"
    description="Pausá productos del local y consultá si quedan ingredientes para prepararlos."
  >
    <form class="card operation-form" @submit.prevent="page(0)">
      <label
        >Buscar producto<input
          v-model="search"
          type="search"
          maxlength="100" /></label
      ><button class="btn-secondary" :disabled="w.busy.value">Buscar</button>
    </form>
    <section class="card operation-form">
      <h2>Productos de este local</h2>
      <p>
        La disponibilidad automática usa stock de productos terminados o las
        recetas configuradas para este local. Los pedidos pendientes no reservan
        ingredientes.
      </p>
      <article
        v-for="item in items"
        :key="item.id"
        class="report-row availability-row"
      >
        <div>
          <strong>{{ item.name }}</strong>
          <p>
            {{
              !item.enabled
                ? "Pausado por el local"
                : item.in_stock
                  ? "Disponible"
                  : "Agotado según inventario"
            }}
          </p>
        </div>
        <button
          class="btn-secondary"
          :disabled="w.busy.value"
          @click="toggle(item)"
        >
          {{ item.enabled ? "Pausar producto" : "Reactivar producto" }}
        </button>
      </article>
      <p v-if="!items.length" class="no-data">No hay productos para mostrar.</p>
      <ListPager
        :offset="offset"
        :count="items.length"
        :disabled="w.busy.value"
        @change="page"
      /></section
  ></WorkspaceFrame>
</template>
<style scoped>
.availability-row {
  gap: 1rem;
  flex-wrap: wrap;
}
.availability-row p {
  margin: 0.4rem 0;
}
</style>
