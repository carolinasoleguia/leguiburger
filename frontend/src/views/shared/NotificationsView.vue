<script setup>
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
const items = ref([]),
  unread = ref(0),
  offset = ref(0);
const w = useWorkspace(async ({ request, current }) => {
  const data = await request(`/notifications?limit=25&offset=${offset.value}`);
  if (current()) {
    items.value = data.items || [];
    unread.value = data.unread || 0;
  }
});
function mark(id = "") {
  return w.act(
    () => w.request("/notifications/read", { id }),
    "Lectura actualizada.",
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
    title="Novedades de tu negocio"
    description="Stock, pedidos, cobros y equipo. Una bandeja personal para seguir lo importante."
    ><div class="section-header">
      <p>{{ unread }} sin leer</p>
      <button
        class="btn-secondary"
        :disabled="w.busy.value || !unread"
        @click="mark()"
      >
        Marcar todas como leídas
      </button>
    </div>
    <article
      v-for="item in items"
      :key="item.id"
      class="card notification-row"
      :class="{ unread: !item.read_at }"
    >
      <div>
        <small>{{ when(item.created_at) }}</small>
        <h2>{{ item.title }}</h2>
        <p>{{ item.message }}</p>
        <router-link :to="item.destination" class="text-link"
          >Abrir sección</router-link
        >
      </div>
      <button
        v-if="!item.read_at"
        class="btn-secondary"
        :disabled="w.busy.value"
        @click="mark(item.id)"
      >
        Marcar como leída
      </button>
    </article>
    <p v-if="!items.length" class="no-data">
      Todo al día. Las novedades del negocio van a aparecer acá.
    </p>
    <ListPager
      :offset="offset"
      :count="items.length"
      :disabled="w.busy.value"
      @change="page"
  /></WorkspaceFrame>
</template>
