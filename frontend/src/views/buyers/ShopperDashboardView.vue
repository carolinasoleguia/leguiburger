<script setup>
import { computed, onMounted, ref, nextTick } from "vue";
import { useRoute } from "vue-router";
import BrandLogo from "../../components/BrandLogo.vue";
import ThemeToggle from "../../components/ThemeToggle.vue";
import ShopperCheckout from "../../components/ShopperCheckout.vue";
import {
  shopper,
  loadShopper,
  shopperRequest,
} from "../../composables/useShopper.js";
import { money, when, statusLabels } from "../../composables/useWorkspace.js";
const props = defineProps({ initialNotice: String, initialError: String });
const route = useRoute();
const tab = ref("pedidos"),
  busy = ref(false),
  loading = ref(true),
  error = ref(""),
  notice = ref(props.initialNotice || "");
const password = ref(""),
  currentPassword = ref(""),
  orders = ref([]),
  coupons = ref([]),
  favorites = ref([]),
  favoriteOffset = ref(0),
  preferences = ref([]),
  selected = ref(null),
  offset = ref(0),
  profileName = ref(""),
  profilePhone = ref(""),
  profileAddress = ref("");
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
    const [history, wallet, prefs, saved] = await Promise.all([
      shopperRequest(`/api/shopper/orders?limit=20&offset=${offset.value}`),
      shopperRequest("/api/shopper/coupons"),
      shopperRequest("/api/shopper/preferences"),
      shopperRequest(
        `/api/shopper/favorites?limit=20&offset=${favoriteOffset.value}`,
      ),
    ]);
    orders.value = history.items || [];
    coupons.value = wallet.items || [];
    preferences.value = prefs.items || [];
    favorites.value = saved.items || [];
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
async function openOrder(order) {
  selected.value = null;
  await nextTick();
  selected.value = order;
}
function removeFavorite(item) {
  return act(async () => {
    await shopperRequest(
      `/api/shopper/favorites/${item.tenant_id}/${item.product_id}`,
      {},
      "DELETE",
    );
    await load();
  }, "Favorito eliminado.");
}
function favoritePage(n) {
  favoriteOffset.value = n;
  load();
}
function logout() {
  return act(async () => {
    await shopperRequest("/api/public/shopper/logout", {});
    shopper.profile = null;
    favorites.value = [];
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
  await load();
  if (props.initialError && !error.value) error.value = props.initialError;
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
              ['favoritos', 'Mis favoritos'],
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
            Todavía no hay pedidos en esta página. Elegí una carta y encontrá tu
            próxima favorita.
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
                Ver y seguir pedido</button
              ><router-link
                v-if="
                  ['canceled', 'rejected'].includes(order.status) ||
                  (order.status === 'delivered' &&
                    Number(order.total_paid) >= Number(order.total_due))
                "
                class="btn-secondary"
                :to="`/catalogo/${order.tenant_id}?repetir=${order.id}`"
                >Pedir lo mismo</router-link
              >
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
        <template v-else-if="tab === 'favoritos'"
          ><div class="shopper-heading">
            <div>
              <p class="eyebrow">Los que siempre vuelven</p>
              <h2>Mis favoritos</h2>
            </div>
            <button class="btn-secondary" :disabled="loading" @click="load">
              Actualizar favoritos
            </button>
          </div>
          <p v-if="loading" role="status" class="shopper-skeleton">
            Buscando tus favoritos…
          </p>
          <p v-else-if="!favorites.length" class="card shopper-empty">
            Guardá tus hamburguesas favoritas desde la carta de cada local.
          </p>
          <article
            v-for="f in loading ? [] : favorites"
            :key="f.tenant_id + f.product_id"
            class="card shopper-order"
          >
            <div>
              <h3>{{ f.name }}</h3>
              <p>{{ f.business }} · {{ f.location }}</p>
              <small>{{
                f.available ? "Disponible" : "No disponible por ahora"
              }}</small>
            </div>
            <div>
              <strong>{{ money(f.price) }}</strong><router-link
                class="btn-secondary"
                :to="`/catalogo/${f.tenant_id}?producto=${f.product_id}`"
                >Elegir en la carta</router-link
              ><button
                class="text-button"
                :disabled="busy"
                @click="removeFavorite(f)"
              >
                Quitar favorito
              </button>
            </div>
          </article>
          <div class="shopper-pagination">
            <button
              class="btn-secondary"
              :disabled="favoriteOffset === 0 || loading"
              @click="favoritePage(Math.max(0, favoriteOffset - 20))"
            >
              Favoritos anteriores</button
            ><button
              class="btn-secondary"
              :disabled="favorites.length < 20 || loading"
              @click="favoritePage(favoriteOffset + 20)"
            >
              Más favoritos
            </button>
          </div></template
        >
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
            <strong>{{ brand.name }}</strong><label
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
                  ofrecerte beneficios. No incluye datos de otras marcas.</small
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
            ><button class="btn-primary" :disabled="busy">Guardar datos</button>
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

      <p v-if="error" role="alert" class="error-msg shopper-feedback">
        {{ error }}
      </p>
      <p v-if="notice" role="status" class="shopper-feedback">{{ notice }}</p>
    </main>
    <ShopperCheckout
      v-if="selected"
      :key="selected.id"
      :tenant="selected.tenant_id"
      :order-id="selected.id"
      @closed="selected = null"
    />
  </div>
</template>
<style scoped src="../../styles/shopper-account.css" />
