<script setup>
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRoute } from "vue-router";
import BuyerCheckout from "../../components/BuyerCheckout.vue";
const checkout = ref(null);
import BrandLogo from "../../components/BrandLogo.vue";
import ThemeToggle from "../../components/ThemeToggle.vue";
import AppIcon from "../../components/AppIcon.vue";
import ListPager from "../../components/ListPager.vue";
import { money } from "../../composables/useWorkspace.js";
const failedImages = ref(new Set());
const route = useRoute(),
  data = ref(null),
  loading = ref(true),
  error = ref(""),
  search = ref(""),
  category = ref(""),
  offset = ref(0);
const tenant = computed(
  () => route.params.tenantId || route.query.tenant_id || "",
);
let controller,
  revision = 0,
  timer;
async function load() {
  const current = ++revision;
  controller?.abort();
  controller = new AbortController();
  error.value = "";
  loading.value = true;
  if (!tenant.value) {
    data.value = null;
    loading.value = false;
    return;
  }
  try {
    const query = new URLSearchParams({
      limit: "24",
      offset: String(offset.value),
      search: search.value,
      category: category.value,
    });
    const response = await fetch(
      "/api/public/menu/" + encodeURIComponent(tenant.value) + "?" + query,
      { credentials: "omit", signal: controller.signal },
    );
    const result = await response.json();
    if (!response.ok)
      throw new Error(
        response.status === 404
          ? "Esta carta todavía no está publicada o el local no está disponible."
          : "No pudimos cargar la carta. Volvé a intentar.",
      );
    if (current === revision) data.value = result;
  } catch (e) {
    if (current === revision && e.name !== "AbortError") {
      error.value = e.message;
      data.value = null;
    }
  } finally {
    if (current === revision) loading.value = false;
  }
}
watch(
  tenant,
  () => {
    search.value = "";
    category.value = "";
    offset.value = 0;
    data.value = null;
    load();
  },
  { immediate: true },
);
watch([search, category], () => {
  offset.value = 0;
  clearTimeout(timer);
  timer = setTimeout(load, 300);
});
function page(n) {
  offset.value = n;
  load();
}
onBeforeUnmount(() => {
  controller?.abort();
  clearTimeout(timer);
});
function image(url) {
  if (!url) return "";
  try {
    const u = new URL(url, window.location.origin);
    return ["https:", "http:"].includes(u.protocol) ? u.href : "";
  } catch {
    return "";
  }
}
</script>
<template>
  <div class="public-menu">
    <a class="skip-link" href="#menu-content">Ir a la carta</a>
    <header class="public-menu-header">
      <router-link to="/" aria-label="LeguiBurger, inicio"
        ><BrandLogo /></router-link
      ><ThemeToggle />
    </header>
    <main id="menu-content" class="public-menu-main">
      <header class="menu-hero">
        <p class="eyebrow">La carta de la casa</p>
        <h1
          :class="{
            'menu-brand-long': (data?.location?.brand_name || '').length > 40,
          }"
        >
          {{ data?.location?.brand_name || "Tu próxima hamburguesa" }}
        </h1>
        <h2>{{ data?.location?.name || "Hecha en tu local favorito" }}</h2>
        <div class="menu-location-details">
          <p :aria-hidden="!data?.location?.address || undefined">
            <AppIcon v-if="data?.location?.address" name="store" :size="16" />{{
              data?.location?.address || " "
            }}
          </p>
          <p :aria-hidden="!data?.location?.hours || undefined">
            {{ data?.location?.hours || " " }}
          </p>
          <p :aria-hidden="!data?.location?.phone || undefined">
            {{
              data?.location?.phone ? `Contacto: ${data.location.phone}` : " "
            }}
          </p>
        </div>
      </header>
      <p class="menu-ordering-notice" :aria-hidden="!data || loading">
        {{
          !data || loading
            ? " "
            : data.ordering_enabled
              ? "Elegí tus favoritas. Pedí acá, sin crear una cuenta."
              : "Por ahora, este local comparte su carta sin recibir pedidos online."
        }}
      </p>
      <section v-if="!tenant" class="card menu-message">
        <AppIcon name="burger" :size="40" />
        <h2>Encontrá la carta de tu local</h2>
        <p>
          Abrí el enlace o escaneá el QR que te compartió la hamburguesería.
        </p>
      </section>
      <template v-else
        ><div class="menu-controls">
          <label
            >Buscar en la carta<input
              v-model="search"
              type="search"
              maxlength="100"
              placeholder="¿Qué te tienta hoy?" /></label
          ><label
            >Categoría<select v-model="category">
              <option value="">Toda la carta</option>
              <option v-for="c in data?.categories || []" :key="c" :value="c">
                {{ c }}
              </option>
            </select></label
          ><span class="menu-result-count" aria-live="polite">{{
            loading
              ? "Consultando…"
              : data
                ? `${data.total} ${data.total === 1 ? "producto" : "productos"}`
                : ""
          }}</span>
        </div>
        <div v-if="error" role="alert" class="card menu-message">
          <h2>No pudimos mostrar esta carta</h2>
          <p>{{ error }}</p>
          <button class="btn-secondary" @click="load">Reintentar</button>
        </div>
        <section
          v-else
          class="public-menu-grid"
          :aria-busy="loading"
          aria-label="Productos de la carta"
        >
          <template v-if="loading"
            ><article
              v-for="i in 6"
              :key="i"
              class="menu-product menu-product-skeleton"
              aria-hidden="true"
            >
              <div class="menu-product-image" />
              <div class="menu-product-copy"><span /><span /><span /></div>
            </article>
            <span class="sr-only" role="status"
              >Cargando la carta</span
            ></template
          >
          <article
            v-for="item in loading ? [] : data?.items || []"
            :key="item.id"
            class="menu-product"
            :class="{ 'menu-unavailable': !item.available }"
          >
            <div class="menu-product-image">
              <img
                v-if="image(item.image_url) && !failedImages.has(item.id)"
                :src="image(item.image_url)"
                :alt="item.name"
                loading="lazy"
                width="480"
                height="320"
                @error="failedImages.add(item.id)"
              /><AppIcon v-else name="burger" :size="72" />
            </div>
            <div class="menu-product-copy">
              <p class="menu-category">{{ item.category }}</p>
              <h3>{{ item.name }}</h3>
              <p class="menu-description">
                {{ item.description || "Preparada en la casa" }}
              </p>
              <div class="menu-product-footer">
                <strong>{{ money(item.price, data.currency) }}</strong><span>{{
                  item.available ? "Disponible" : "No disponible por ahora"
                }}</span>
              </div>
              <button
                v-if="data.ordering_enabled"
                class="btn-primary menu-add"
                :disabled="!item.available"
                @click="checkout?.open(item)"
              >
                Elegir producto
              </button>
            </div>
          </article>
          <div
            v-if="!loading && data && !data.items.length"
            class="card menu-message"
          >
            <h2>No encontramos productos</h2>
            <p>
              Probá otra búsqueda o categoría. Si la carta está vacía, el local
              todavía está preparando su oferta.
            </p>
          </div>
        </section>
        <ListPager
          v-if="data"
          :offset="offset"
          :count="data.items.length"
          :size="24"
          :total="data.total"
          :disabled="loading"
          @change="page"
      /></template>
    </main>
    <BuyerCheckout
      v-if="tenant"
      :key="tenant"
      ref="checkout"
      :tenant="tenant"
      :menu="data"
    />
    <footer class="public-menu-footer">
      <span>Tu marca. Tu sabor.</span
      ><router-link to="/">Con LeguiBurger</router-link>
    </footer>
  </div>
</template>
<style scoped>
.menu-add {
  margin-top: 20px;
  width: 100%;
}
.menu-ordering-notice {
  min-height: 48px;
  color: var(--text-muted);
  margin-bottom: 24px;
}
.public-menu {
  min-height: 100dvh;
  background: var(--bg);
  color: var(--text);
}
.public-menu-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px max(24px, calc((100% - 1200px) / 2));
  border-bottom: 1px solid var(--border);
}
.public-menu-main {
  max-width: 1248px;
  margin: auto;
  padding: 48px 24px 64px;
}
.menu-hero {
  min-height: 200px;
  margin-bottom: 32px;
}
.menu-location-details {
  min-height: 104px;
}
.menu-hero h1 {
  min-height: 120px;
  display: flex;
  align-items: center;
  overflow-wrap: anywhere;
  font-size: clamp(36px, 5vw, 60px);
  margin-bottom: 8px;
}
.menu-hero .menu-brand-long {
  font-size: clamp(24px, 3vw, 36px);
}
.menu-hero h2 {
  min-height: 48px;
  color: var(--accent);
  font-size: 20px;
}
.menu-hero p {
  display: flex;
  gap: 8px;
  align-items: center;
  color: var(--text-muted);
  margin: 8px 0;
}
.menu-controls {
  display: flex;
  align-items: end;
  flex-wrap: wrap;
  gap: 16px;
  min-height: 76px;
  margin-bottom: 24px;
}
.menu-controls label {
  display: grid;
  gap: 8px;
  font-size: 13px;
  flex: 1 1 240px;
}
.menu-controls :is(input, select) {
  padding: 12px;
  min-height: 44px;
  color: var(--text);
  background: var(--field);
  border: 1px solid var(--control-border);
  border-radius: 8px;
  width: 100%;
}
.menu-result-count {
  min-width: 120px;
  min-height: 44px;
  display: grid;
  align-items: center;
  color: var(--text-muted);
  font-size: 13px;
}
.public-menu-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 24px;
  min-height: 400px;
}
.menu-product {
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 16px;
  overflow: hidden;
  background: var(--surface);
  animation: menu-appear 220ms ease-out both;
}
.menu-product-image {
  height: 224px;
  position: relative;
  display: grid;
  place-items: center;
  background: var(--surface-raised);
  color: var(--accent);
  overflow: hidden;
}
.menu-product-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.menu-product-copy {
  padding: 24px;
  display: flex;
  flex-direction: column;
  min-height: 224px;
}
.menu-product-copy h3 {
  font-size: 22px;
  margin: 0 0 12px;
}
.menu-category {
  font-size: 12px;
  color: var(--text-muted);
  margin: 0 0 8px;
}
.menu-description {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0 0 24px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.menu-product-footer {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
}
.menu-product-footer strong {
  font-size: 20px;
  font-variant-numeric: tabular-nums;
}
.menu-product-footer span {
  font-size: 12px;
  color: var(--text-muted);
  text-align: right;
}
.menu-unavailable .menu-product-image {
  filter: grayscale(0.8);
}
.menu-message {
  grid-column: 1/-1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  text-align: center;
  min-height: 300px;
  padding: 32px;
}
.menu-message p {
  max-width: 50ch;
  color: var(--text-muted);
}
.public-menu-footer {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  max-width: 1200px;
  margin: auto;
  padding: 24px 24px 110px;
  border-top: 1px solid var(--border);
  font-size: 13px;
  color: var(--text-muted);
}
.menu-product-skeleton .menu-product-image,
.menu-product-skeleton .menu-product-copy span {
  background: var(--surface-raised);
  animation: menu-shimmer 1.3s ease-in-out infinite alternate;
}
.menu-product-skeleton .menu-product-copy span {
  height: 16px;
  border-radius: 4px;
  margin-bottom: 16px;
  width: 80%;
}
.menu-product-skeleton .menu-product-copy span:last-child {
  width: 45%;
  margin-top: 24px;
}
.public-menu :deep(.list-pager) {
  margin-top: 32px;
}
@keyframes menu-shimmer {
  to {
    opacity: 0.5;
  }
}
@keyframes menu-appear {
  from {
    opacity: 0.65;
  }
  to {
    opacity: 1;
  }
}
@media (max-width: 1000px) {
  .public-menu-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 600px) {
  .public-menu-main {
    padding-top: 32px;
  }
  .public-menu-grid {
    grid-template-columns: 1fr;
  }
  .menu-product-image {
    height: 224px;
  }
  .menu-controls label {
    flex-basis: 100%;
  }
  .menu-hero {
    min-height: 200px;
  }
}
</style>
