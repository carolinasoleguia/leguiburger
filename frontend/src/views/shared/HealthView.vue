<script setup>
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
const data = ref(null);
const businesses = ref([]),
  offset = ref(0);
const w = useWorkspace(async ({ request, current }) => {
  const [value, list] = await Promise.all([
    request("/platform/health"),
    request(`/platform/businesses?limit=25&offset=${offset.value}`),
  ]);
  if (current()) {
    data.value = value;
    businesses.value = list.items || [];
  }
});
function classify(item) {
  return w.act(
    () =>
      w.request(
        `/platform/businesses/${item.id}`,
        { data_classification: item.data_classification },
        "PATCH",
      ),
    "Clasificación guardada. Las métricas respetan esta clasificación.",
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
    title="Estado del sistema"
    description="Seguimiento de los procesos y las colas de trabajo de LeguiBurger."
    ><template v-if="data"
      ><div class="metric-grid">
        <article class="card">
          <h3>Correos pendientes</h3>
          <strong>{{ data.mail_pending }}</strong>
        </article>
        <article class="card">
          <h3>Correos que agotaron reintentos</h3>
          <strong>{{ data.mail_failed }}</strong>
        </article>
        <article class="card">
          <h3>Webhooks pendientes</h3>
          <strong>{{ data.webhooks_pending }}</strong>
        </article>
      </div>
      <p v-if="!data.workers?.length" class="error-msg">
        No hay señales recientes del worker. Revisá el proceso de segundo plano.
      </p>
      <article
        v-for="worker in data.workers"
        :key="worker.instance_id"
        class="card operation-form"
      >
        <h2>
          {{
            worker.stale
              ? "Sin señal reciente"
              : worker.failed
                ? "Reintentando tareas"
                : "Worker activo"
          }}
        </h2>
        <p>
          Última señal: {{ when(worker.last_seen_at) }}<br />Último ciclo de
          notificaciones correcto: {{ when(worker.last_success_at) }}
        </p>
        <small>{{ worker.instance_id }}</small>
      </article>
      <section class="card operation-form">
        <h2>Negocios y clasificación de datos</h2>
        <p>
          Los cobros de prueba, internos o sin clasificar quedan fuera de la
          facturación de producción.
        </p>
        <div v-for="item in businesses" :key="item.id" class="form-row">
          <strong>{{ item.name }}</strong
          ><label
            >Clasificación de {{ item.name
            }}<select v-model="item.data_classification">
              <option value="unclassified">Sin clasificar</option>
              <option value="production">Negocio real</option>
              <option value="test">Prueba</option>
              <option value="internal">Uso interno</option>
            </select></label
          ><button
            class="btn-secondary"
            :disabled="w.busy.value"
            @click="classify(item)"
          >
            Guardar clasificación
          </button>
        </div>
        <ListPager
          :offset="offset"
          :count="businesses.length"
          :disabled="w.busy.value"
          @change="page"
        />
      </section> </template
  ></WorkspaceFrame>
</template>
