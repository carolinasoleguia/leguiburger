<script setup>
import { computed, onMounted, ref, nextTick } from "vue";
import { useRoute } from "vue-router";
import BrandLogo from "../../components/BrandLogo.vue";
import ThemeToggle from "../../components/ThemeToggle.vue";
import BuyerCheckout from "../../components/BuyerCheckout.vue";
import {
  shopper,
  loadShopper,
  shopperRequest,
} from "../../composables/useShopper.js";
import { money, when, statusLabels } from "../../composables/useWorkspace.js";
const route = useRoute();
const tab = ref("pedidos"),
  mode = ref("login"),
  busy = ref(false),
  loading = ref(true),
  error = ref(""),
  notice = ref("");
const name = ref(""),
  email = ref(""),
  password = ref(""),
  currentPassword = ref(""),
  orders = ref([]),
  coupons = ref([]),
  preferences = ref([]),
  selected = ref(null),
  offset = ref(0),
  profileName = ref(""),
  profilePhone = ref(""),
  profileAddress = ref("");
let actionToken =
  new URLSearchParams(location.hash.slice(1)).get("token") || "";
if (actionToken)
  history.replaceState(null, "", location.pathname + location.search);
const action = ref(
  ["verify", "reset"].includes(route.query.accion) ? route.query.accion : "",
);
const activeOrders = computed(
  () =>
    orders.value.filter(
      (o) => !["delivered", "canceled", "rejected"].includes(o.status),
    ).length,
);
async function load() {
  loading.value = true;
  error.value = "";
  try {
    if (!shopper.ready) await loadShopper();
    if (!shopper.profile) return;
    profileName.value = shopper.profile.name;
    profilePhone.value = shopper.profile.phone || "";
    profileAddress.value = shopper.profile.delivery_address || "";
    const [history, wallet, prefs] = await Promise.all([
      shopperRequest(`/api/shopper/orders?limit=20&offset=${offset.value}`),
      shopperRequest("/api/shopper/coupons"),
      shopperRequest("/api/shopper/preferences"),
    ]);
    orders.value = history.items || [];
    coupons.value = wallet.items || [];
    preferences.value = prefs.items || [];
  } catch (e) {
    error.value = e.message;
    if (e.status === 401) shopper.profile = null;
  } finally {
    loading.value = false;
  }
}
async function act(fn, message = "") {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  notice.value = "";
  try {
    await fn();
    if (message) notice.value = message;
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
function submitAuth() {
  return act(async () => {
    if (action.value) {
      await shopperRequest(`/api/shopper/${action.value}`, {
        token: actionToken,
        ...(action.value === "reset" ? { password: password.value } : {}),
      });
      notice.value =
        action.value === "verify"
          ? "Correo confirmado. Ya podés recibir novedades de las marcas que elegiste."
          : "Contraseña actualizada. Ingresá con tu nueva contraseña.";
      action.value = "";
      actionToken = "";
      password.value = "";
      await loadShopper();
      await load();
      return;
    }
    if (mode.value === "forgot") {
      const result = await shopperRequest("/api/shopper/forgot", {
        email: email.value,
      });
      notice.value = result.message;
      return;
    }
    shopper.profile = await shopperRequest(`/api/shopper/${mode.value}`, {
      email: email.value,
      password: password.value,
      ...(mode.value === "register" ? { name: name.value } : {}),
    });
    shopper.ready = true;
    password.value = "";
    await load();
  });
}
async function openOrder(order) {
  selected.value = null;
  await nextTick();
  selected.value = order;
}
function logout() {
  return act(async () => {
    await shopperRequest("/api/public/shopper/logout", {});
    shopper.profile = null;
    orders.value = [];
    coupons.value = [];
    preferences.value = [];
    selected.value = null;
  });
}
function preference(brand, event, field = "newsletter") {
  const value = event.target.checked;
  return act(async () => {
    try {
      await shopperRequest(
        "/api/shopper/preferences",
        { brand_id: brand.id, [field]: value },
        "PATCH",
      );
      brand[field] = value;
    } catch (e) {
      event.target.checked = brand[field];
      throw e;
    }
  }, "Preferencias guardadas.");
}
function claim() {
  return act(async () => {
    await shopperRequest("/api/shopper/claim", {
      tenant_id: route.query.local,
    });
    await load();
  }, "Pedido vinculado a tu cuenta.");
}
function saveProfile() {
  return act(async () => {
    await shopperRequest(
      "/api/shopper/profile",
      {
        name: profileName.value,
        phone: profilePhone.value,
        delivery_address: profileAddress.value,
      },
      "PATCH",
    );
    await loadShopper();
  }, "Datos guardados para tus próximas compras.");
}
function changePassword() {
  return act(async () => {
    await shopperRequest("/api/shopper/password", {
      current_password: currentPassword.value,
      password: password.value,
    });
    currentPassword.value = "";
    password.value = "";
  }, "Contraseña actualizada. Se cerraron las otras sesiones.");
}
function resend() {
  return act(
    () => shopperRequest("/api/shopper/resend", {}),
    "Correo de confirmación solicitado.",
  );
}
onMounted(async () => {
  try {
    if (action.value && !actionToken) {
      action.value = "";
      await load();
      error.value =
        "El enlace está incompleto. Volvé a abrir el correo original.";
    } else if (action.value === "verify") await submitAuth();
    else await load();
  } finally {
    loading.value = false;
  }
});
</script>
<template>
  <div class="shopper-account">
    <header class="shopper-top">
      <router-link to="/" aria-label="LeguiBurger, inicio"
        ><BrandLogo
      /></router-link>
      <div>
        <router-link to="/restaurantes" class="text-link"
          >Explorar cartas</router-link
        ><ThemeToggle />
      </div>
    </header>
    <main class="shopper-main">
      <template v-if="!shopper.profile || action">
        <section class="shopper-welcome">
          <p class="eyebrow">Tu próxima favorita te espera</p>
          <h1>Todas tus hamburgueserías.<br />Una sola cuenta.</h1>
          <p>
            Guardá tus pedidos, seguí cada entrega y encontrá beneficios de las
            marcas que te gustan.
          </p>
          <router-link to="/restaurantes" class="text-link"
            >También podés comprar sin registrarte →</router-link
          >
        </section>
        <section class="card shopper-auth" :aria-busy="loading || busy">
          <h2>
            {{
              action === "reset"
                ? "Elegí una nueva contraseña"
                : mode === "register"
                  ? "Creá tu cuenta"
                  : mode === "forgot"
                    ? "Recuperá tu cuenta"
                    : "Qué bueno verte de nuevo"
            }}
          </h2>
          <p v-if="loading" role="status">Consultando tu cuenta…</p>
          <div v-else-if="action === 'verify'">
            <p>
              Solicitá un nuevo enlace desde tu perfil si el anterior venció.
            </p>
            <button
              class="btn-secondary"
              @click="
                action = '';
                load();
              "
            >
              Volver a mi cuenta
            </button>
          </div>
          <form v-else @submit.prevent="submitAuth" class="shopper-form">
            <label v-if="mode === 'register' && !action"
              >Tu nombre<input
                v-model="name"
                autocomplete="name"
                required
                minlength="2"
                maxlength="100"
            /></label>
            <label v-if="!action"
              >Correo<input
                v-model="email"
                autocomplete="email"
                type="email"
                required
                maxlength="254"
            /></label>
            <label v-if="mode !== 'forgot' || action"
              >Contraseña<input
                v-model="password"
                :autocomplete="
                  mode === 'register' || action
                    ? 'new-password'
                    : 'current-password'
                "
                type="password"
                :minlength="mode === 'register' || action ? 12 : undefined"
                required
                maxlength="72"
            /></label>
            <p v-if="mode === 'register'" class="shopper-muted">
              Usá al menos 12 caracteres. Las novedades de cada marca se activan
              por separado, si vos querés.
            </p>
            <button class="btn-primary" :disabled="busy">
              {{
                busy
                  ? "Un momento…"
                  : action
                    ? "Guardar contraseña"
                    : mode === "register"
                      ? "Crear cuenta"
                      : mode === "forgot"
                        ? "Solicitar enlace"
                        : "Ingresar"
              }}
            </button>
            <button
              v-if="!action"
              type="button"
              class="text-button"
              @click="
                mode = mode === 'login' ? 'register' : 'login';
                error = '';
                notice = '';
              "
            >
              {{
                mode === "login"
                  ? "¿Primera vez? Creá tu cuenta"
                  : "Ya tengo una cuenta"
              }}
            </button>
            <button
              v-if="mode === 'login' && !action"
              type="button"
              class="text-button"
              @click="mode = 'forgot'"
            >
              Olvidé mi contraseña
            </button>
          </form>
        </section>
      </template>
      <template v-else>
        <aside class="shopper-sidebar">
          <p class="eyebrow">Mi cuenta</p>
          <h1>Hola, {{ shopper.profile.name }}</h1>
          <p class="shopper-muted">
            {{
              activeOrders
                ? `${activeOrders} ${activeOrders === 1 ? "pedido" : "pedidos"} en curso en esta página`
                : "Siempre hay lugar para una más."
            }}
          </p>
          <nav aria-label="Tu cuenta">
            <router-link to="/restaurantes" class="btn-primary shopper-start"
              >Explorar restaurantes
              <span aria-hidden="true">→</span></router-link
            >
            <button
              v-for="item in [
                ['pedidos', 'Mis pedidos'],
                ['cupones', 'Mis cupones'],
                ['preferencias', 'Novedades de marcas'],
                ['perfil', 'Mi perfil'],
              ]"
              :key="item[0]"
              :class="{ active: tab === item[0] }"
              :aria-current="tab === item[0] ? 'page' : undefined"
              @click="tab = item[0]"
            >
              {{ item[1] }}
            </button>
          </nav>
          <button
            type="button"
            class="text-button"
            :disabled="busy"
            @click="logout"
          >
            Cerrar sesión
          </button>
        </aside>
        <section class="shopper-content" :aria-busy="loading">
          <template v-if="tab === 'pedidos'"
            ><div class="shopper-heading">
              <div>
                <p class="eyebrow">Cada antojo tiene su historia</p>
                <h2>Mis pedidos</h2>
              </div>
              <button class="btn-secondary" :disabled="loading" @click="load">
                Actualizar
              </button>
            </div>
            <button
              v-if="route.query.local"
              class="btn-secondary"
              :disabled="busy"
              @click="claim"
            >
              Vincular el pedido de este navegador
            </button>
            <div v-if="loading" class="shopper-skeleton" role="status">
              Cargando tus pedidos…
            </div>
            <p v-else-if="!orders.length" class="card shopper-empty">
              Todavía no hay pedidos en esta página. Elegí una carta y encontrá
              tu próxima favorita.
            </p>
            <article
              v-for="order in loading ? [] : orders"
              :key="order.id"
              class="card shopper-order"
            >
              <div>
                <small>{{ when(order.created_at) }}</small>
                <h3>{{ order.business }}</h3>
                <p>{{ order.location }} · #{{ order.id.slice(0, 8) }}</p>
                <span class="shopper-status">{{
                  statusLabels[order.status] || order.status
                }}</span>
              </div>
              <div>
                <strong>{{ money(order.total_due) }}</strong><button
                  class="btn-secondary"
                  @click="openOrder(order)"
                >
                  Ver y seguir pedido
                </button>
              </div>
            </article>
            <div class="shopper-pagination">
              <button
                class="btn-secondary"
                :disabled="loading || offset === 0"
                @click="
                  offset = Math.max(0, offset - 20);
                  load();
                "
              >
                Anterior</button
              ><button
                class="btn-secondary"
                :disabled="loading || orders.length < 20"
                @click="
                  offset += 20;
                  load();
                "
              >
                Siguiente
              </button>
            </div>
          </template>
          <template v-else-if="tab === 'cupones'"
            ><p class="eyebrow">Un buen motivo para volver</p>
            <h2>Mis cupones</h2>
            <p>
              Descuentos de cada marca para tu cuenta. Un uso por comprador; se
              descuentan de los productos, sin extras ni envío.
            </p>
            <p v-if="!coupons.length" class="card shopper-empty">
              Tus próximos beneficios van a aparecer acá.
            </p>
            <div class="shopper-coupon-grid">
              <article
                v-for="coupon in coupons"
                :key="coupon.id"
                class="card shopper-coupon"
              >
                <small
                  >{{ coupon.business }} ·
                  {{
                    coupon.personal ? "Para vos" : "Para compradores con cuenta"
                  }}</small
                >
                <h3>{{ coupon.percent }}% de descuento</h3>
                <code>{{ coupon.code }}</code>
                <p>
                  Mínimo {{ money(coupon.minimum) }} · Hasta
                  {{ when(coupon.expires_at) }}
                </p>
                <span v-if="coupon.used">Ya utilizado</span
                ><router-link
                  v-else-if="coupon.location_id"
                  class="btn-primary"
                  :to="`/catalogo/${coupon.location_id}?cupon=${encodeURIComponent(coupon.code)}`"
                  >Elegir en la carta</router-link
                >
              </article>
            </div></template
          >
          <template v-else-if="tab === 'preferencias'"
            ><p class="eyebrow">Solo lo que te interesa</p>
            <h2>Novedades de tus marcas</h2>
            <p>
              Elegí de quién querés recibir newsletters y promociones. Podés
              cambiarlo cuando quieras.
            </p>
            <p v-if="!shopper.profile.email_verified" class="inline-notice">
              Tus preferencias se guardan como solicitud. Para recibir correos
              primero tenés que confirmar tu dirección.
            </p>
            <p v-if="!preferences.length" class="card shopper-empty">
              Las marcas con carta publicada aparecerán acá.
            </p>
            <article
              v-for="brand in preferences"
              :key="brand.id"
              class="card shopper-preference"
            >
              <strong>{{ brand.name }}</strong
              ><label
                ><span
                  >Novedades por correo<small>{{
                    brand.newsletter
                      ? shopper.profile.email_verified
                        ? "Suscripción habilitada"
                        : "Solicitud pendiente de confirmar correo"
                      : "Sin suscripción"
                  }}</small></span
                ><input
                  type="checkbox"
                  :checked="brand.newsletter"
                  :disabled="busy"
                  @change="preference(brand, $event)" /></label
              ><label
                ><span
                  >Promociones según mis compras<small
                    >Esta marca podrá usar solo las compras que le hiciste para
                    ofrecerte beneficios. No incluye datos de otras
                    marcas.</small
                  ></span
                ><input
                  data-personalization
                  type="checkbox"
                  :checked="brand.personalization"
                  :disabled="busy"
                  @change="preference(brand, $event, 'personalization')"
              /></label>
            </article>
          </template>
          <template v-else
            ><p class="eyebrow">Tu lugar en la mesa</p>
            <h2>Mi perfil</h2>
            <section class="card">
              <h3>{{ shopper.profile.name }}</h3>
              <p>{{ shopper.profile.email }}</p>
              <p>
                {{
                  shopper.profile.email_verified
                    ? "Correo confirmado"
                    : "Correo pendiente de confirmación"
                }}
              </p>
              <button
                v-if="!shopper.profile.email_verified"
                class="btn-secondary"
                :disabled="busy"
                @click="resend"
              >
                Enviar confirmación
              </button>
            </section>
            <form class="card shopper-form" @submit.prevent="saveProfile">
              <h3>Datos para tus compras</h3>
              <p class="shopper-muted">
                El teléfono y la dirección son opcionales. Se usarán para
                completar tu próxima compra; podés editarlos o borrarlos cuando
                quieras.
              </p>
              <label
                >Nombre<input
                  v-model="profileName"
                  autocomplete="name"
                  required
                  minlength="2"
                  maxlength="100" /></label
              ><label
                >Teléfono<input
                  v-model="profilePhone"
                  type="tel"
                  autocomplete="tel"
                  maxlength="30" /></label
              ><label
                >Dirección habitual<textarea
                  v-model="profileAddress"
                  autocomplete="street-address"
                  rows="3"
                  maxlength="500"
                ></textarea></label
              ><button class="btn-primary" :disabled="busy">
                Guardar datos
              </button>
            </form>
            <form class="card shopper-form" @submit.prevent="changePassword">
              <h3>Cambiar contraseña</h3>
              <label
                >Contraseña actual<input
                  v-model="currentPassword"
                  type="password"
                  autocomplete="current-password"
                  required
                  maxlength="72" /></label
              ><label
                >Nueva contraseña<input
                  v-model="password"
                  type="password"
                  autocomplete="new-password"
                  minlength="12"
                  maxlength="72"
                  required /></label
              ><button class="btn-primary" :disabled="busy">
                Guardar contraseña
              </button>
            </form></template
          >
        </section>
      </template>
      <p v-if="error" role="alert" class="error-msg shopper-feedback">
        {{ error }}
      </p>
      <p v-if="notice" role="status" class="shopper-feedback">{{ notice }}</p>
    </main>
    <BuyerCheckout
      v-if="selected"
      :key="selected.id"
      :tenant="selected.tenant_id"
      :order-id="selected.id"
      @closed="selected = null"
    />
  </div>
</template>
<style scoped>
.shopper-account {
  min-height: 100dvh;
  background: var(--bg);
  color: var(--text);
}
.shopper-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px max(20px, calc((100% - 1200px) / 2));
  border-bottom: 1px solid var(--border);
}
.shopper-top > div {
  display: flex;
  align-items: center;
  gap: 20px;
}
.shopper-main {
  max-width: 1240px;
  margin: auto;
  padding: 48px 20px;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 40px;
}
.shopper-welcome {
  align-self: center;
}
.shopper-welcome h1 {
  font-size: clamp(32px, 4vw, 52px);
  line-height: 1.12;
  letter-spacing: -0.04em;
}
.shopper-welcome p {
  line-height: 1.7;
  color: var(--text-muted);
}
.shopper-main:has(.shopper-auth) {
  grid-template-columns: minmax(0, 1fr) minmax(300px, 440px);
}
.shopper-auth {
  padding: 32px;
}
.shopper-form {
  display: grid;
  gap: 20px;
}
.shopper-form label {
  display: grid;
  gap: 8px;
  font-size: 14px;
}
.shopper-form input,
.shopper-form textarea {
  width: 100%;
  min-height: 46px;
  background: var(--field);
  border: 1px solid var(--control-border);
  border-radius: 8px;
  color: var(--text);
  padding: 12px;
}
.shopper-sidebar h1 {
  font-size: 28px;
  overflow-wrap: anywhere;
}
.shopper-sidebar nav {
  display: grid;
  gap: 8px;
  margin: 28px 0;
}
.shopper-start {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 54px;
  padding: 16px;
  font-weight: 700;
  text-decoration: none;
  grid-column: 1/-1;
  margin-bottom: 8px;
}
.shopper-sidebar nav button {
  padding: 14px 16px;
  text-align: left;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--text-muted);
  background: transparent;
  min-height: 48px;
}
.shopper-sidebar nav button.active {
  background: var(--surface-raised);
  border-color: var(--border);
  color: var(--text);
  font-weight: 650;
}
.shopper-content {
  min-width: 0;
  animation: shopper-in 220ms ease-out;
}
.shopper-content h2 {
  font-size: 32px;
  margin-top: 0;
}
.shopper-content > p {
  color: var(--text-muted);
  line-height: 1.6;
}
.shopper-heading,
.shopper-order {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}
.shopper-order {
  margin: 16px 0;
  padding: 24px;
  min-height: 174px;
}
.shopper-order h3 {
  margin: 8px 0;
  font-size: 24px;
}
.shopper-order p,
.shopper-order small {
  color: var(--text-muted);
}
.shopper-order > div:last-child {
  display: grid;
  gap: 18px;
  text-align: right;
  flex-shrink: 0;
}
.shopper-order strong {
  font-size: 22px;
}
.shopper-status {
  display: inline-block;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 20px;
  font-size: 13px;
}
.shopper-coupon-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}
.shopper-coupon {
  padding: 24px;
  display: grid;
  gap: 12px;
  border-top: 3px solid var(--accent);
}
.shopper-coupon h3 {
  font-size: 28px;
  margin: 0;
}
.shopper-coupon code {
  width: fit-content;
  padding: 10px 16px;
  border: 1px dashed var(--control-border);
  border-radius: 8px;
  letter-spacing: 0.08em;
}
.shopper-coupon p,
.shopper-muted {
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}
.shopper-preference {
  display: grid;
  gap: 20px;
  margin: 16px 0;
  padding: 24px;
}
.shopper-preference label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  cursor: pointer;
}
.shopper-preference small {
  display: block;
  margin-top: 8px;
  color: var(--text-muted);
}
.shopper-preference input {
  width: 24px;
  height: 24px;
  accent-color: var(--accent);
}
.shopper-empty,
.shopper-skeleton {
  min-height: 174px;
  padding: 32px;
  display: grid;
  align-items: center;
  color: var(--text-muted);
}
.shopper-skeleton {
  background: var(--surface-raised);
  border-radius: 16px;
  animation: shopper-pulse 1s ease-in-out infinite alternate;
}
.shopper-pagination {
  display: flex;
  justify-content: space-between;
  margin-top: 24px;
}
.shopper-feedback {
  grid-column: 1/-1;
}
.shopper-content > .card {
  margin-bottom: 24px;
}
@keyframes shopper-in {
  from {
    opacity: 0.6;
  }
  to {
    opacity: 1;
  }
}
@keyframes shopper-pulse {
  to {
    opacity: 0.55;
  }
}
@media (max-width: 760px) {
  .shopper-main,
  .shopper-main:has(.shopper-auth) {
    grid-template-columns: 1fr;
    gap: 28px;
    padding-top: 28px;
  }
  .shopper-sidebar nav {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    margin: 20px 0;
  }
  .shopper-sidebar h1 {
    margin-bottom: 8px;
  }
  .shopper-coupon-grid {
    grid-template-columns: 1fr;
  }
  .shopper-order {
    align-items: stretch;
    flex-direction: column;
  }
  .shopper-order > div:last-child {
    text-align: left;
  }
  .shopper-auth {
    padding: 24px;
  }
  .shopper-top > div {
    gap: 12px;
    font-size: 13px;
  }
}
</style>
