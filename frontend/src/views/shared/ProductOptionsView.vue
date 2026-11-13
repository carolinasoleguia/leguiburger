<script setup>
import { ref, watch } from "vue";
import ListPager from "../../components/ListPager.vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import { useWorkspace, money } from "../../composables/useWorkspace.js";
const search = ref(""),
  offset = ref(0);
const products = ref([]),
  extras = ref([]),
  product = ref(""),
  loaded = ref(false),
  configLoading = ref(false),
  config = ref({ use_product_options: false, groups: [] });
const extraName = ref(""),
  extraPrice = ref("0"),
  extraStock = ref(0),
  extraTracked = ref(false);
const w = useWorkspace(async ({ request, tenant, current }) => {
  products.value = [];
  extras.value = [];
  if (!tenant) return;
  const [p, x] = await Promise.all([
    request(
      `/menu/availability?limit=25&offset=${offset.value}&search=${encodeURIComponent(search.value)}`,
    ),
    request("/extras"),
  ]);
  if (current()) {
    products.value = p.items || [];
    extras.value = (Array.isArray(x) ? x : []).map((e) => ({
      id: e.id || e.ID,
      name: e.name || e.Name,
      price: e.current_price ?? e.CurrentPrice,
      active: e.is_active ?? e.IsActive,
    }));
  }
});
let revision = 0;
watch(product, async (id) => {
  const current = ++revision;
  loaded.value = false;
  if (!id) return;
  configLoading.value = true;
  try {
    const result = await w.request(`/menu/products/${id}/options`);
    if (current === revision) {
      config.value = result;
      loaded.value = true;
    }
  } catch (e) {
    w.error.value = e.message;
  } finally {
    if (current === revision) configLoading.value = false;
  }
});
watch(w.tenant, () => {
  product.value = "";
  offset.value = 0;
  search.value = "";
  config.value = { use_product_options: false, groups: [] };
  extraName.value = "";
  extraPrice.value = "0";
  extraStock.value = 0;
});
function page(n) {
  product.value = "";
  offset.value = n;
  w.refresh();
}
function addGroup() {
  config.value.groups.push({ name: "", min: 0, max: 1, options: [] });
}
function used(id, own) {
  return (
    id !== own &&
    config.value.groups.some((g) => g.options.some((o) => o.extra_id === id))
  );
}
function save() {
  return w.act(
    () =>
      w.request(
        `/menu/products/${product.value}/options`,
        {
          use_product_options: config.value.use_product_options,
          groups: config.value.groups.map((g) => ({
            ...g,
            options: g.options.map((o) => ({
              extra_id: o.extra_id,
              price_override:
                o.price_override === "" || o.price_override == null
                  ? null
                  : o.price_override,
            })),
          })),
        },
        "PUT",
      ),
    "Opciones del producto guardadas. Los pedidos anteriores conservan sus elecciones.",
  );
}
function createExtra() {
  return w.act(async () => {
    await w.request("/extras", {
      name: extraName.value,
      current_price: Number(extraPrice.value),
      current_stock: extraTracked.value ? Number(extraStock.value) : 0,
      track_stock: extraTracked.value,
    });
    extraName.value = "";
    extraPrice.value = "0";
    extraStock.value = 0;
  }, "Extra creado en este local. Ya podés vincularlo a un producto.");
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Combos y opciones"
    description="Usá los extras del local para definir variantes, acompañamientos y bebidas por producto."
  >
    <div class="workspace-split options-layout">
      <aside>
        <section class="card operation-form">
          <h2>Elegí el producto</h2>
          <form class="form-row" @submit.prevent="page(0)">
            <label
              >Buscar en toda la carta<input
                v-model="search"
                type="search"
                maxlength="100" /></label
            ><button class="btn-secondary" :disabled="w.busy.value">
              Buscar producto
            </button>
          </form>
          <label
            >Producto de este local<select
              v-model="product"
              :disabled="w.busy.value"
            >
              <option value="">Elegí de la carta</option>
              <option v-for="p in products" :key="p.id" :value="p.id">
                {{ p.name }}
              </option>
            </select></label
          >
          <ListPager
            :offset="offset"
            :count="products.length"
            :disabled="w.busy.value"
            @change="page"
          />
          <p>
            Para un combo, creá el producto base en tu catálogo y agregá grupos
            de bebida o acompañamiento. Cada elección reutiliza el precio, costo
            y stock de un extra.
          </p>
        </section>
        <details class="card operation-form">
          <summary>Crear un extra en este local</summary>
          <form
            v-form-care
            class="operation-form"
            @submit.prevent="createExtra"
          >
            <p>
              Después podés administrar su stock y costo desde Stock e insumos.
            </p>
            <div class="form-row">
              <label
                >Nombre del extra<input
                  v-model="extraName"
                  required
                  maxlength="100"
                  placeholder="Bebida a elección" /></label
              ><label
                >Precio habitual<input
                  v-model="extraPrice"
                  required
                  inputmode="decimal"
                  pattern="[0-9]+([.][0-9]{1,2})?"
              /></label>
            </div>
            <label class="options-check"
              ><input v-model="extraTracked" type="checkbox" />Controlar stock
              de este extra</label
            ><label v-if="extraTracked"
              >Unidades disponibles<input
                v-model.number="extraStock"
                type="number"
                min="0"
                step="1"
                required /></label
            ><button
              class="btn-secondary"
              :disabled="w.busy.value || !w.tenant.value"
            >
              Crear extra
            </button>
          </form>
        </details>
      </aside>
      <div>
        <p v-if="configLoading" role="status">
          Cargando las opciones del producto…
        </p>
        <form
          v-if="product"
          v-form-care
          class="card operation-form"
          @submit.prevent="save"
        >
          <fieldset :disabled="!loaded || w.busy.value">
            <legend>
              Opciones para {{ products.find((p) => p.id === product)?.name }}
            </legend>
            <label class="options-check"
              ><input
                v-model="config.use_product_options"
                type="checkbox"
              />Usar opciones específicas de este producto</label
            >
            <p v-if="!config.use_product_options">
              Se mantienen los extras generales del local. Activá esta opción
              para restringir elecciones y definir grupos obligatorios.
            </p>
            <template v-if="config.use_product_options"
              ><p>
                Un precio vacío usa el precio habitual del extra. Cero significa
                incluido en el precio base. Los cupones siguen aplicándose al
                precio base del producto.
              </p>
              <section
                v-for="(g, index) in config.groups"
                :key="index"
                class="option-editor"
              >
                <div class="form-row">
                  <label
                    >Nombre del grupo<input
                      v-model="g.name"
                      required
                      minlength="2"
                      maxlength="80"
                      placeholder="Bebida" /></label
                  ><label
                    >Mínimo de elecciones<input
                      v-model.number="g.min"
                      type="number"
                      min="0"
                      :max="g.max"
                      required /></label
                  ><label
                    >Máximo de elecciones<input
                      v-model.number="g.max"
                      type="number"
                      :min="Math.max(1, g.min)"
                      max="5"
                      required
                  /></label>
                </div>
                <div
                  v-for="(o, i) in g.options"
                  :key="i"
                  class="form-row option-editor-row"
                >
                  <label
                    >Extra<select v-model="o.extra_id" required>
                      <option value="">Elegí un extra</option>
                      <option
                        v-for="x in extras"
                        :key="x.id"
                        :value="x.id"
                        :disabled="used(x.id, o.extra_id) || !x.active"
                      >
                        {{ x.name }} · {{ money(x.price)
                        }}{{ x.active ? "" : " · Inactivo" }}
                      </option>
                    </select></label
                  ><label
                    >Precio en este producto<input
                      v-model="o.price_override"
                      inputmode="decimal"
                      pattern="[0-9]+([.][0-9]{1,2})?"
                      placeholder="Usar precio del extra" /></label
                  ><button
                    type="button"
                    class="btn-secondary"
                    @click="g.options.splice(i, 1)"
                  >
                    Quitar opción
                  </button>
                </div>
                <div class="operation-actions">
                  <button
                    type="button"
                    class="btn-secondary"
                    :disabled="g.options.length >= 30"
                    @click="
                      g.options.push({ extra_id: '', price_override: '' })
                    "
                  >
                    Agregar opción</button
                  ><button
                    type="button"
                    class="text-button"
                    @click="config.groups.splice(index, 1)"
                  >
                    Quitar grupo
                  </button>
                </div>
              </section>
              <p v-if="!config.groups.length">
                Con esta configuración activa y sin grupos, el producto se vende
                sin extras.
              </p>
              <button
                type="button"
                class="btn-secondary"
                :disabled="config.groups.length >= 6"
                @click="addGroup"
              >
                Agregar grupo
              </button></template
            >
          </fieldset>
          <button class="btn-primary" :disabled="w.busy.value || !loaded">
            Guardar opciones del producto
          </button>
        </form>
      </div>
    </div>
  </WorkspaceFrame>
</template>
<style scoped>
fieldset {
  border: 0;
  padding: 0;
  min-width: 0;
  display: grid;
  gap: 1rem;
}
legend {
  font-weight: 700;
  margin-bottom: 1rem;
}
.options-check {
  display: flex;
  gap: 0.6rem;
  align-items: center;
}
.options-check input {
  width: auto;
  accent-color: var(--accent);
}
.option-editor {
  padding: 1rem;
  border: 1px solid var(--border);
  border-radius: 12px;
  display: grid;
  gap: 1rem;
}
.option-editor-row {
  align-items: end;
}
summary {
  cursor: pointer;
  font-weight: 700;
  min-height: 44px;
  display: flex;
  align-items: center;
}
</style>
