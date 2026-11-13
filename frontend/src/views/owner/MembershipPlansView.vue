<script setup>
import { ref, computed, onMounted } from "vue";
import DashboardShell from "../../components/DashboardShell.vue";
import StableContent from "../../components/StableContent.vue";
import { apiFetch, getJSON } from "../../services/api.js";
const busy = ref(false),
  loading = ref(true),
  error = ref(""),
  notice = ref(""),
  pendingKeys = new Map();
const money = (value, currency = "ARS") =>
  new Intl.NumberFormat("es-AR", { style: "currency", currency }).format(
    Number(value),
  );
const platformPlans = ref([]),
  newPlanName = ref(""),
  pricingPlan = ref(""),
  newPrice = ref("0");
const newPlanCode = ref(
  `plan_${crypto.randomUUID().replaceAll("-", "").slice(0, 16)}`,
);
const membershipTier = ref("standard"),
  extraLocationAmount = ref("0");
const priceRows = computed(() =>
  platformPlans.value.flatMap((item) =>
    (item.prices || []).map((price) => ({ ...price, plan_name: item.name })),
  ),
);
async function createPlan() {
  return execute(async () => {
    const result = await mutate("/platform/billing/plans", {
      code: newPlanCode.value,
      name: newPlanName.value,
    });
    pricingPlan.value = result.id;
    newPlanName.value = "";
    newPlanCode.value = `plan_${crypto.randomUUID().replaceAll("-", "").slice(0, 16)}`;
    notice.value = "Plan creado. Publicá su precio para ofrecerlo.";
  });
}
async function publishPrice() {
  return execute(async () => {
    await mutate(`/platform/billing/plans/${pricingPlan.value}/prices`, {
      currency: "ARS",
      amount: String(newPrice.value),
      interval_months: 1,
      billing_unit: "brand",
      trial_days: 0,
      membership_tier: membershipTier.value,
      extra_location_amount:
        membershipTier.value === "premium"
          ? String(extraLocationAmount.value)
          : "0",
      published: false,
    });
    newPrice.value = "";
    notice.value =
      "Tarifa guardada en borrador. Podés publicarla cuando estén definidos los importes.";
  });
}
function setPricePublished(item) {
  return execute(async () => {
    await mutate(
      `/platform/billing/prices/${item.id}/publication`,
      { published: !item.is_active },
      "PATCH",
    );
    notice.value = item.is_active
      ? "Tarifa retirada de nuevas contrataciones. Los contratos vigentes se conservan."
      : "Tarifa publicada para nuevas contrataciones.";
  });
}
async function mutate(path, body, method = "POST") {
  const operation = method + ":" + path + ":" + JSON.stringify(body);
  if (!pendingKeys.has(operation))
    pendingKeys.set(operation, crypto.randomUUID());
  const response = await apiFetch(path, {
    method,
    headers: { "Idempotency-Key": pendingKeys.get(operation) },
    body: JSON.stringify(body),
  });
  const result = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new Error(result.message || "No se pudo guardar la tarifa");
  pendingKeys.delete(operation);
  return result;
}
async function refresh() {
  platformPlans.value = (await getJSON("/platform/billing/plans")).items || [];
}
async function execute(fn) {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  notice.value = "";
  try {
    await fn();
    await refresh();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
onMounted(async () => {
  try {
    await refresh();
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
});
</script>
<template>
  <DashboardShell role="owner"
    ><section class="plans-page">
      <header class="section-header">
        <div>
          <h1>Planes y tarifas</h1>
          <p>
            Las membresías que los restaurantes pagan a LeguiBurger. Un
            administrador por marca; terminales operativas por local.
          </p>
        </div>
      </header>
      <p v-if="error" role="alert" class="error-msg">{{ error }}</p>
      <p v-if="notice" role="status" class="status-notice">{{ notice }}</p>
      <StableContent :loading="loading" :busy="busy" label="Cargando planes…"
        ><div class="plan-forms">
          <form
            v-form-care
            class="card action-form"
            @submit.prevent="createPlan"
          >
            <h3>Planes de LeguiBurger</h3>
            <div class="input-group">
              <label for="new-plan-name">Nombre del nuevo plan</label
              ><input
                id="new-plan-name"
                v-model="newPlanName"
                minlength="2"
                maxlength="120"
                required
              />
            </div>
            <button class="btn-primary" :disabled="busy">Crear plan</button>
          </form>
          <form
            v-form-care
            class="card action-form"
            @submit.prevent="publishPrice"
          >
            <h3>Preparar una tarifa</h3>
            <p>
              Cada precio nuevo conserva intactos los contratos ya vendidos.
            </p>
            <div class="input-group">
              <label for="pricing-plan">Plan</label
              ><select id="pricing-plan" v-model="pricingPlan" required>
                <option
                  v-for="item in platformPlans"
                  :key="item.id"
                  :value="item.id"
                >
                  {{ item.name }}
                </option>
              </select>
            </div>
            <div class="input-group">
              <label for="new-plan-price">Importe en pesos</label
              ><input
                id="new-plan-price"
                v-model="newPrice"
                type="text"
                inputmode="decimal"
                pattern="[0-9]+([.][0-9]{1,2})?"
                required
              />
            </div>
            <div class="input-group">
              <label for="membership-tier">Tipo de membresía</label>
              <select id="membership-tier" v-model="membershipTier">
                <option value="standard">Clásico</option>
                <option value="premium">Premium</option>
              </select>
            </div>
            <p>
              Mensual, por marca, con un local incluido.
              {{
                membershipTier === "premium"
                  ? "Cinco terminales por local; permite contratar tiendas adicionales."
                  : "Una tienda y dos terminales."
              }}
            </p>
            <div v-if="membershipTier === 'premium'" class="input-group">
              <label for="extra-location-price"
                >Adicional mensual por local, en pesos</label
              >
              <input
                id="extra-location-price"
                v-model="extraLocationAmount"
                type="text"
                inputmode="decimal"
                pattern="[0-9]+([.][0-9]{1,2})?"
                required
              />
            </div>
            <p>
              Podés guardar importes en cero como borrador; para publicar hace
              falta definir el precio mensual y, en Premium, el adicional.
            </p>
            <button class="btn-primary" :disabled="busy || !pricingPlan">
              Guardar borrador
            </button>
          </form>
          <div class="card payment-table">
            <table class="data-table">
              <caption>
                Tarifas de membresía · las condiciones contratadas se conservan
              </caption>
              <thead>
                <tr>
                  <th>Plan y versión</th>
                  <th>Mensual</th>
                  <th>Local adicional</th>
                  <th>Estado</th>
                  <th>Publicación</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in priceRows" :key="item.id">
                  <td>
                    {{ item.plan_name }} · v{{ item.version }}<br /><small>{{
                      item.membership_tier === "premium"
                        ? "Premium · 5 terminales por local"
                        : item.membership_tier === "standard"
                          ? "Clásico"
                          : "Condiciones anteriores"
                    }}</small>
                  </td>
                  <td>{{ money(item.amount, item.currency) }}</td>
                  <td>
                    {{
                      item.membership_tier === "premium"
                        ? money(item.extra_location_amount, item.currency)
                        : "—"
                    }}
                  </td>
                  <td>
                    {{ item.is_active ? "Publicada" : "Borrador / retirada" }}
                  </td>
                  <td>
                    <button
                      type="button"
                      class="btn-secondary"
                      :disabled="
                        busy ||
                        (!item.is_active &&
                          item.membership_tier !== 'legacy' &&
                          (Number(item.amount) <= 0 ||
                            (item.membership_tier === 'premium' &&
                              Number(item.extra_location_amount) <= 0)))
                      "
                      @click="setPricePublished(item)"
                    >
                      {{ item.is_active ? "Retirar" : "Publicar" }}
                    </button>
                  </td>
                </tr>
                <tr v-if="!priceRows.length">
                  <td colspan="5">
                    No hay tarifas. Prepará un borrador para empezar.
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div></StableContent
      >
    </section></DashboardShell
  >
</template>
<style scoped>
.plans-page {
  max-width: 76rem;
  margin: 0 auto;
}
.plan-forms {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
}
.action-form {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding: 1.25rem;
  min-width: 0;
  margin: 0;
}
.action-form:nth-child(2) {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.action-form:nth-child(2) :is(h3, p, button) {
  grid-column: 1/-1;
}
.action-form :is(h3, p, .input-group) {
  margin: 0;
}
.action-form p {
  font-size: 13px;
  line-height: 1.5;
}
.action-form .input-group {
  min-width: 0;
}
.action-form .input-group label {
  margin-bottom: 5px;
}
.payment-table {
  grid-column: 1/-1;
  overflow: auto;
  padding: 1rem;
}
.data-table {
  width: 100%;
  min-width: 620px;
}
.data-table caption {
  text-align: left;
  padding-bottom: 1rem;
}
.data-table td,
.data-table th {
  padding: 0.7rem;
  text-align: left;
}
.action-form p {
  color: var(--text-muted);
}
@media (max-width: 720px) {
  .action-form:nth-child(2) {
    grid-template-columns: 1fr;
  }
  .plan-forms {
    grid-template-columns: 1fr;
  }
  .payment-table {
    grid-column: auto;
  }
}
</style>
