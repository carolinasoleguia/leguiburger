<script setup>
import { computed, ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import { useWorkspace, when } from "../../composables/useWorkspace.js";
const data = ref(null);
const w = useWorkspace(async ({ request, tenant, current }) => {
  data.value = null;
  if (!tenant) return;
  const value = await request("/onboarding");
  if (current()) data.value = value;
});
const required = computed(
    () => data.value?.steps?.filter((x) => !x.optional) || [],
  ),
  done = computed(() => required.value.filter((x) => x.done).length);
function finish() {
  return w.act(
    () => w.request("/onboarding/complete", {}),
    "Tu negocio está listo para trabajar.",
  );
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Dejá todo a punto"
    description="Un recorrido por lo necesario para empezar a tomar pedidos."
    ><template v-if="data"
      ><p>{{ done }} de {{ required.length }} pasos esenciales completos.</p>
      <div
        class="progress-track"
        role="progressbar"
        :aria-valuenow="done"
        :aria-valuemax="required.length"
        aria-valuemin="0"
        aria-label="Preparación del negocio"
      >
        <span
          :style="{ width: `${(done / Math.max(1, required.length)) * 100}%` }"
        />
      </div>
      <ul class="steps-list">
        <li v-for="step in data.steps" :key="step.key" class="card">
          <div>
            <h2>{{ step.done ? "✓ " : "" }}{{ step.title }}</h2>
            <p>
              {{
                step.done
                  ? "Listo para el servicio"
                  : step.optional
                    ? "Podés seguir cobrando manualmente mientras conectás tu cuenta."
                    : "Todavía queda este paso."
              }}
            </p>
          </div>
          <router-link :to="step.to" class="btn-secondary">{{
            step.done ? "Revisar" : "Configurar"
          }}</router-link>
        </li>
      </ul>
      <p v-if="data.completed_at">
        Preparación finalizada el {{ when(data.completed_at) }}. Los pasos
        reflejan la configuración actual.
      </p>
      <button
        v-else
        class="btn-primary"
        :disabled="w.busy.value || done !== required.length"
        @click="finish"
      >
        Finalizar preparación
      </button></template
    >
    <p v-else class="no-data">
      La administración de la app debe asignar un local activo a tu marca para
      continuar.
    </p></WorkspaceFrame
  >
</template>
