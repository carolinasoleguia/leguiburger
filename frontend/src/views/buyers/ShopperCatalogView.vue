<script setup>
import { computed, nextTick, ref, watch } from "vue";
import { useRoute } from "vue-router";
import RestaurantMenu from "../../components/RestaurantMenu.vue";
import ShopperCheckout from "../../components/ShopperCheckout.vue";
import { shopper, shopperRequest } from "../../composables/useShopper.js";
const props = defineProps({ tenantId: String });
const route = useRoute(),
  browser = ref(null),
  checkout = ref(null);
const tenant = computed(
  () => props.tenantId || route.params.tenantId || route.query.tenant_id || "",
);
const favorites = ref([]),
  favoritesReady = ref(false),
  favoriteBusy = ref(""),
  favoriteError = ref("");
let openedFavorite = "";
async function loadFavorites() {
  const id = tenant.value,
    account = shopper.profile?.id;
  favoritesReady.value = false;
  favorites.value = [];
  if (!account) return;
  try {
    const first = await shopperRequest(
      `/api/shopper/favorites?tenant_id=${id}&limit=100`,
    );
    let all = first.items || [];
    if (all.length === 100) {
      const next = await shopperRequest(
        `/api/shopper/favorites?tenant_id=${id}&limit=100&offset=100`,
      );
      all = all.concat(next.items || []);
    }
    if (id === tenant.value && account === shopper.profile?.id) {
      favorites.value = all;
      favoritesReady.value = true;
    }
  } catch (e) {
    favoriteError.value = e.message;
  }
}
function isFavorite(item) {
  return favorites.value.some((f) => f.product_id === item.id);
}
async function favorite(item) {
  if (favoriteBusy.value) return;
  const id = tenant.value;
  favoriteBusy.value = item.id;
  favoriteError.value = "";
  try {
    const exists = isFavorite(item);
    await shopperRequest(
      `/api/shopper/favorites/${id}/${item.id}`,
      {},
      exists ? "DELETE" : "PUT",
    );
    if (id !== tenant.value) return;
    if (exists)
      favorites.value = favorites.value.filter((f) => f.product_id !== item.id);
    else favorites.value.push({ ...item, product_id: item.id, tenant_id: id });
  } catch (e) {
    favoriteError.value = e.message;
  } finally {
    favoriteBusy.value = "";
  }
}
watch([tenant, () => shopper.profile?.id], loadFavorites, { immediate: true });
watch(
  [
    () => checkout.value?.ready,
    () => route.query.producto,
    favoritesReady,
    () => browser.value?.data,
    () => browser.value?.loading,
  ],
  async () => {
    // Wait for the resolved menu to reach the checkout before opening a deep link.
    await nextTick();
    const desired = String(route.query.producto || "");
    if (
      !desired ||
      !browser.value?.data ||
      browser.value?.loading ||
      !checkout.value?.ready ||
      !favoritesReady.value ||
      openedFavorite === tenant.value + desired
    )
      return;
    const product =
      browser.value?.data?.items?.find((x) => x.id === desired) ||
      favorites.value.find((x) => x.product_id === desired);
    if (product) {
      openedFavorite = tenant.value + desired;
      checkout.value.open({ ...product, id: product.product_id || product.id });
    }
  },
  { flush: "post" },
);
</script>
<template>
  <RestaurantMenu
    data-experience="shopper"
    :tenant-id="tenantId"
    ref="browser"
    :checkout-ready="!!checkout?.ready"
    ordering-notice="Elegí tus favoritas. Tu pedido se guarda en tu cuenta."
    @choose="checkout?.open($event)"
  >
    <template #account-link
      ><router-link
        class="text-link"
        :to="`/mi-cuenta?local=${encodeURIComponent(tenant)}`"
        >Mi cuenta</router-link
      ></template
    >
    <template #notices
      ><p v-if="favoriteError" class="error-msg" role="alert">
        {{ favoriteError }}
      </p></template
    >
    <template #product-actions="{ item }"
      ><button
        class="text-button menu-favorite"
        :aria-pressed="isFavorite(item)"
        :disabled="!!favoriteBusy || !favoritesReady"
        @click="favorite(item)"
      >
        {{ isFavorite(item) ? "Quitar de favoritos" : "Guardar favorito" }}
      </button></template
    >
    <template #checkout="{ tenant: local, menu }"
      ><ShopperCheckout
        v-if="local"
        :key="local + String(route.query.repetir || '')"
        ref="checkout"
        :tenant="local"
        :menu="menu"
        :repeat-order-id="String(route.query.repetir || '')"
    /></template>
  </RestaurantMenu>
</template>
