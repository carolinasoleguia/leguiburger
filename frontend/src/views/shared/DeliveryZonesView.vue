<script setup>
import { ref, watch } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import { useWorkspace, money } from "../../composables/useWorkspace.js";
const items = ref([]),
  enabled = ref(false),
  id = ref(""),
  name = ref(""),
  description = ref(""),
  cost = ref("0"),
  minimum = ref("0"),
  codes = ref(""),
  active = ref(true);
const w = useWorkspace(async ({ request, tenant, current }) => {
  items.value = [];
  if (!tenant) return;
  const [settings, zones] = await Promise.all([
    request("/delivery/settings"),
    request("/delivery/zones"),
  ]);
  if (current()) {
    enabled.value = settings.zones_enabled;
    items.value = zones.items || [];
  }
});
function edit(z) {
  id.value = z?.id || "";
  name.value = z?.name || "";
  description.value = z?.description || "";
  cost.value = z?.cost || "0";
  minimum.value = z?.minimum || "0";
  codes.value = z?.postal_codes?.join(", ") || "";
  active.value = z?.active ?? true;
}
watch(w.tenant, () => edit(null));
function save() {
  return w.act(async () => {
    await w.request(
      "/delivery/zones" + (id.value ? "/" + id.value : ""),
      {
        name: name.value,
        description: description.value,
        cost: cost.value,
        minimum: minimum.value,
        postal_codes: codes.value.split(/[,;\s]+/).filter(Boolean),
        active: active.value,
      },
      id.value ? "PATCH" : "POST",
    );
    edit(null);
  }, "Zona de envío guardada.");
}
function toggle() {
  return w.act(
    () =>
      w.request(
        "/delivery/settings",
        { zones_enabled: !enabled.value },
        "PATCH",
      ),
    enabled.value
      ? "Cobertura por zonas desactivada."
      : "Cobertura por códigos postales habilitada.",
  );
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Zonas de delivery"
    description="Definí cobertura, tarifas y compra mínima para los envíos de este local."
  >
    <section class="card operation-form">
      <h2>
        {{
          enabled
            ? "Cobertura por código postal activa"
            : "Envío con métodos habituales"
        }}
      </h2>
      <p>
        Cuando habilitás las zonas, el checkout exige un código postal cubierto
        por la zona elegida. La dirección debe estar dentro de ese código
        postal; no se verifica la ubicación exacta del domicilio.
      </p>
      <button
        class="btn-secondary"
        :disabled="w.busy.value || !w.tenant.value"
        @click="toggle"
      >
        {{
          enabled
            ? "Desactivar cobertura por zonas"
            : "Habilitar cobertura por zonas"
        }}
      </button>
    </section>
    <form v-form-care class="card operation-form" @submit.prevent="save">
      <h2>{{ id ? "Editar zona" : "Nueva zona de envío" }}</h2>
      <fieldset :disabled="w.busy.value || !w.tenant.value">
        <div class="form-row">
          <label
            >Nombre<input
              v-model="name"
              required
              minlength="2"
              maxlength="80"
              placeholder="City Bell centro" /></label
          ><label
            >Tarifa de envío<input
              v-model="cost"
              inputmode="decimal"
              pattern="[0-9]+([.][0-9]{1,2})?"
              required /></label
          ><label
            >Compra mínima<input
              v-model="minimum"
              inputmode="decimal"
              pattern="[0-9]+([.][0-9]{1,2})?"
              required
          /></label>
        </div>
        <label
          >Barrios o referencias<textarea
            v-model="description"
            maxlength="300"
            rows="2"
            placeholder="Descripción de la zona para tus compradores"
          ></textarea>
        </label>
        <label
          >Códigos postales<input
            v-model="codes"
            required
            maxlength="1500"
            placeholder="1896, 1897" /></label
        ><small
          >Separalos con comas. También podés ingresar CPA completos; se utiliza
          su código postal base de cuatro dígitos.</small
        >
        <label class="zone-check"
          ><input v-model="active" type="checkbox" />Zona activa</label
        >
        <p>
          La compra mínima se calcula después de descuentos, sin incluir el
          envío. Los pedidos anteriores conservan la tarifa y la zona
          confirmadas al comprar.
        </p>
      </fieldset>
      <div class="operation-actions">
        <button class="btn-primary" :disabled="w.busy.value || !w.tenant.value">
          Guardar zona</button
        ><button
          v-if="id"
          type="button"
          class="btn-secondary"
          @click="edit(null)"
        >
          Cancelar edición
        </button>
      </div>
    </form>
    <section class="card operation-form">
      <h2>Zonas del local</h2>
      <article v-for="z in items" :key="z.id" class="report-row zone-row">
        <div>
          <strong>{{ z.name }} · {{ z.active ? "Activa" : "Pausada" }}</strong>
          <p>{{ money(z.cost) }} de envío · mínimo {{ money(z.minimum) }}</p>
          <p>CP {{ z.postal_codes.join(", ") }}</p>
          <p>{{ z.description }}</p>
        </div>
        <button class="btn-secondary" :disabled="w.busy.value" @click="edit(z)">
          Editar zona
        </button>
      </article>
      <p v-if="!items.length" class="no-data">
        Agregá la primera zona antes de habilitar la cobertura.
      </p>
    </section>
  </WorkspaceFrame>
</template>
<style scoped>
.zone-row {
  gap: 1rem;
  flex-wrap: wrap;
}
.zone-row p {
  margin: 0.4rem 0;
}
.zone-check {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}
.zone-check input {
  width: auto;
}
fieldset {
  border: 0;
  padding: 0;
  display: grid;
  gap: 1rem;
  min-width: 0;
}
</style>
