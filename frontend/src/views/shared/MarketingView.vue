<script setup>
import { ref } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import { useWorkspace, money, when } from "../../composables/useWorkspace.js";
const coupons = ref([]),
  customers = ref([]),
  code = ref(""),
  percent = ref(10),
  minimum = ref("0"),
  days = ref(7),
  maxUses = ref(100),
  recipient = ref(""),
  allLocations = ref(false);
const w = useWorkspace(async ({ request, tenant, current }) => {
  coupons.value = [];
  customers.value = [];
  if (!tenant) return;
  const [c, p] = await Promise.all([
    request("/marketing/coupons"),
    request("/marketing/customers"),
  ]);
  if (current()) {
    coupons.value = c.items || [];
    customers.value = p.items || [];
  }
});
function create() {
  return w.act(async () => {
    await w.request("/marketing/coupons", {
      code: code.value,
      percent: percent.value,
      minimum: minimum.value,
      days: days.value,
      max_uses: maxUses.value,
      shopper_id: recipient.value || null,
      all_locations: allLocations.value,
    });
    code.value = "";
  }, "Cupón creado. Ya está disponible para las cuentas elegidas.");
}
function toggle(c) {
  return w.act(
    () =>
      w.request(`/marketing/coupons/${c.id}`, { active: !c.active }, "PATCH"),
    "Disponibilidad actualizada.",
  );
}
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Beneficios para tus compradores"
    description="Creá cupones para tu marca y consultá quiénes eligieron recibir sus novedades."
    ><form v-form-care class="card operation-form" @submit.prevent="create">
      <h2>Nuevo cupón</h2>
      <p>
        Porcentaje sobre el precio base de los productos. Sin descuento sobre
        extras o envío. Un uso por cuenta; una compra cancelada conserva el uso.
      </p>
      <div class="form-row">
        <label
          >Código<input
            v-model="code"
            required
            pattern="[A-Za-z0-9_-]{3,32}"
            minlength="3"
            maxlength="32"
            placeholder="VOLVE10" /></label
        ><label
          >Descuento (%)<input
            v-model.number="percent"
            type="number"
            required
            min="1"
            max="50" /></label
        ><label
          >Compra mínima<input
            v-model="minimum"
            inputmode="decimal"
            pattern="[0-9]+([.][0-9]{1,2})?"
            required
        /></label>
      </div>
      <div class="form-row">
        <label
          >Vigencia en días<input
            v-model.number="days"
            type="number"
            min="1"
            max="90"
            required /></label
        ><label
          >Usos totales<input
            v-model.number="maxUses"
            type="number"
            min="1"
            max="100000"
            required /></label
        ><label
          >Destinatario<select v-model="recipient">
            <option value="">Todas las cuentas de compradores</option>
            <option v-for="p in customers" :key="p.id" :value="p.id">
              {{ p.name }} · {{ p.orders }} pedidos en tu marca
            </option>
          </select></label
        >
      </div>
      <label class="marketing-check"
        ><input v-model="allLocations" type="checkbox" /> Disponible en todas
        las sucursales de esta marca</label
      >
      <p v-if="!allLocations">
        Se aplicará únicamente en el local seleccionado.
      </p>
      <button class="btn-primary" :disabled="w.busy.value || !w.tenant.value">
        Crear cupón
      </button>
    </form>
    <section class="card operation-form">
      <h2>Cupones de tu marca</h2>
      <p v-if="!coupons.length">Todavía no hay cupones.</p>
      <article v-for="c in coupons" :key="c.id" class="marketing-coupon">
        <div>
          <strong>{{ c.code }} · {{ c.percent }}%</strong>
          <p>
            {{ c.recipient || "Todas las cuentas" }} ·
            {{ c.tenant_id ? "Una sucursal" : "Toda la marca" }}
          </p>
          <small
            >Mínimo {{ money(c.minimum) }} · {{ c.uses }}/{{ c.max_uses }} usos
            · Hasta {{ when(c.expires_at) }}</small
          >
        </div>
        <button
          class="btn-secondary"
          :disabled="w.busy.value"
          @click="toggle(c)"
        >
          {{ c.active ? "Pausar" : "Activar" }}
        </button>
      </article>
    </section>
    <section class="card operation-form">
      <h2>Compradores con cuenta</h2>
      <p>
        Solo se muestran personas que compraron en tu marca. La actividad de
        otras marcas queda privada.
      </p>
      <p v-if="!customers.length">
        Aparecerán acá cuando hagan o vinculen su primera compra.
      </p>
      <div v-for="p in customers" :key="p.id" class="report-row">
        <strong>{{ p.name }}</strong><span
          >{{ p.orders }} pedidos ·
          {{
            p.newsletter_confirmed
              ? "Novedades habilitadas con correo confirmado"
              : "Sin suscripción confirmada"
          }}</span
        >
        <div v-if="p.personalization" class="marketing-insight">
          <strong>{{ p.segment }}</strong>
          <p>
            {{ p.paid_orders }} compras pagadas · {{ money(p.spent) }} recibido
          </p>
          <p v-if="p.last_purchase">
            Última compra: {{ when(p.last_purchase) }}
          </p>
          <p v-if="p.favorite_product">
            Producto más comprado: {{ p.favorite_product }}
          </p>
          <p>
            {{
              p.segment === "Para recuperar"
                ? "Podés ofrecerle un beneficio para que vuelva."
                : p.segment === "Frecuente"
                  ? "Podés premiar su frecuencia con un cupón personal."
                  : "Podés ofrecerle un beneficio para una próxima compra."
            }}
          </p>
        </div>
        <p v-else class="marketing-no-profile">
          Este comprador no habilitó promociones según sus compras.
        </p>
      </div>
    </section></WorkspaceFrame
  >
</template>
<style scoped>
.report-row {
  flex-wrap: wrap;
}
.marketing-insight,
.marketing-no-profile {
  flex-basis: 100%;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}
.marketing-insight {
  padding: 16px;
  background: var(--surface-raised);
  border-radius: 8px;
  margin-top: 12px;
}
.marketing-coupon {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  padding: 20px 0;
  border-bottom: 1px solid var(--border);
}
.marketing-coupon p,
.marketing-coupon small {
  color: var(--text-muted);
  line-height: 1.6;
}
.marketing-check {
  display: flex !important;
  align-items: center;
  gap: 12px;
}
.marketing-check input {
  width: 20px;
  height: 20px;
  accent-color: var(--accent);
}
@media (max-width: 600px) {
  .marketing-coupon {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
