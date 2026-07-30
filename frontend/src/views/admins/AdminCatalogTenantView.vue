<template>
  <section class="admin-catalog-tenant">
    <div class="section-header">
      <div class="section-title">
        <h2>Asignación por tienda</h2>
        <p>Elegí una tienda, revisá el catálogo base de la marca y ajustá la configuración local de cada producto.</p>
      </div>
      <div class="catalog-actions">
        <button class="btn-secondary" @click="goBack">Volver al catálogo</button>
      </div>
    </div>

    <div class="card tenant-picker-card">
      <div class="panel-heading">
        <div>
          <h3>1. Seleccioná una tienda</h3>
          <p>Primero elegí la sucursal sobre la que vas a trabajar.</p>
        </div>
      </div>

      <div class="controls-grid controls-grid--tenant">
        <div class="input-group">
          <label>Tienda</label>
          <select v-model="selectedTenant">
            <option value="">Seleccioná una tienda</option>
            <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
              {{ tenant.subdomain || tenant.id }}
            </option>
          </select>
        </div>

        <div class="tenant-summary" v-if="selectedTenant">
          <span class="tenant-summary__label">Tienda activa</span>
          <strong>{{ selectedTenantLabel }}</strong>
          <span class="tenant-summary__meta">{{ tenantProducts.length }} productos configurados</span>
        </div>
      </div>
    </div>

    <div v-if="!selectedTenant" class="card empty-state-card">
      <h3>Seleccioná una tienda para continuar</h3>
      <p>Cuando elijas una sucursal vas a ver su catálogo actual y podrás abrir el listado base para sumar hamburguesas con un clic.</p>
    </div>

    <div v-else class="card">
      <div class="panel-heading panel-heading--stacked">
        <div>
          <h3>2. Catálogo de la tienda</h3>
          <p>{{ tenantProducts.length }} productos forman parte de esta sucursal</p>
        </div>
        <div class="panel-heading__actions">
          <button class="btn-primary btn-fab" type="button" @click="openCatalogModal" aria-label="Agregar hamburguesas al catálogo de la tienda">+</button>
        </div>
      </div>

      <div v-if="loadingTenantProducts" class="centered-loading compact">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="tenantProductError" class="status-text error">{{ tenantProductError }}</div>
      <div v-else-if="tenantProducts.length === 0" class="status-text">Esta tienda todavía no tiene productos asignados. Usá el botón `+` para agregar hamburguesas desde el catálogo base.</div>
      <div v-else class="tenant-product-grid">
        <article v-for="item in tenantProducts" :key="item.product_id" class="tenant-product-card">
          <div class="tenant-product-card__media">
            <img v-if="productImage(item.product || productById(item.product_id))" :src="productImage(item.product || productById(item.product_id))" :alt="productName(item.product || productById(item.product_id))" class="tenant-product-card__image" />
            <div v-else class="tenant-product-card__image tenant-product-card__image--placeholder"></div>
          </div>

          <div class="tenant-product-card__body">
            <div class="tenant-product-card__heading">
              <div>
                <h3>{{ productName(item.product || productById(item.product_id)) || item.product_id }}</h3>
                <p>{{ item.track_stock ? 'Con control de stock' : 'Sin control de stock' }}</p>
              </div>
              <span class="status-pill" :class="item.is_available && item.is_active ? 'status-pill--success' : 'status-pill--muted'">
                {{ item.is_available && item.is_active ? 'Disponible' : 'No disponible' }}
              </span>
            </div>

            <div class="tenant-product-card__meta">
              <div>
                <span class="meta-label">Precio</span>
                <strong>{{ money(effectivePrice(item)) }}</strong>
              </div>
              <div>
                <span class="meta-label">Stock</span>
                <strong>{{ item.track_stock ? item.current_stock : '-' }}</strong>
              </div>
            </div>

            <div class="tenant-product-card__actions">
              <button class="btn-secondary" @click="openAssignModal(item.product || productById(item.product_id), item)">Editar</button>
              <button class="btn-danger" @click="removeTenantProduct(item)">Quitar</button>
            </div>
          </div>
        </article>
      </div>
    </div>

    <div v-if="isCatalogModalOpen" class="modal-overlay" @click.self="closeCatalogModal">
      <div class="modal-card modal-card--catalog">
        <button type="button" class="modal-close" @click="closeCatalogModal">×</button>
        <h3>Catálogo base de {{ brandLabel }}</h3>
        <p>Elegí una hamburguesa para agregarla a {{ selectedTenantLabel }}.</p>

        <div v-if="loadingProducts" class="centered-loading compact">
          <span class="spinner-large"></span>
        </div>
        <div v-else-if="productError" class="status-text error">{{ productError }}</div>
        <div v-else-if="products.length === 0" class="status-text">Todavía no hay productos en el catálogo de la marca.</div>
        <div v-else class="base-product-list">
          <article v-for="product in orderedProducts" :key="productId(product)" class="base-product-row">
            <div class="base-product-row__main">
              <div class="base-product-row__thumb">
                <img v-if="productImage(product)" :src="productImage(product)" :alt="productName(product)" class="base-product-row__image" />
                <div v-else class="base-product-row__image base-product-row__image--placeholder"></div>
              </div>
              <div>
                <h4>{{ productName(product) }}</h4>
                <p>{{ productDescription(product) || 'Sin descripción cargada.' }}</p>
                <span class="meta-label">{{ money(productBasePrice(product)) }}</span>
              </div>
            </div>

            <div class="base-product-row__actions">
              <button
                v-if="!tenantProductById(productId(product))"
                class="btn-primary btn-action-label"
                type="button"
                @click="openAssignModal(product, tenantProductById(productId(product)))"
              >
                Agregar
              </button>
              <span v-else class="status-pill status-pill--success">
                En tienda
              </span>
            </div>
          </article>
        </div>
      </div>
    </div>

    <div v-if="isAssignModalOpen" class="modal-overlay" @click.self="closeAssignModal">
      <div class="modal-card modal-card--wide">
        <button type="button" class="modal-close" @click="closeAssignModal">×</button>
        <h3>Configuración por tienda</h3>
        <p>{{ productName(assignProduct) }} en {{ selectedTenantLabel }}</p>

        <form @submit.prevent="submitTenantProduct">
          <div class="input-grid">
            <div class="input-group">
              <label>Precio local</label>
              <input v-model.number="tenantProductForm.price_override" type="number" min="0" step="0.01" />
            </div>
            <div class="input-group">
              <label>Stock</label>
              <input v-model.number="tenantProductForm.current_stock" type="number" min="0" />
            </div>
          </div>

          <div class="toggle-grid">
            <label class="checkbox-row">
              <input v-model="tenantProductForm.track_stock" type="checkbox" />
              <span>Controlar stock</span>
            </label>
            <label class="checkbox-row">
              <input v-model="tenantProductForm.is_available" type="checkbox" />
              <span>Disponible</span>
            </label>
            <label v-if="tenantProductForm.exists" class="checkbox-row">
              <input v-model="tenantProductForm.is_active" type="checkbox" />
              <span>Activo</span>
            </label>
          </div>

          <div class="modal-actions modal-actions--split" v-if="tenantProductForm.exists">
            <div class="modal-actions__group">
              <button type="button" class="btn-secondary" @click="closeAssignModal" :disabled="actionLoading">Cancelar</button>
              <button type="submit" class="btn-primary" :disabled="actionLoading">
                <span v-if="actionLoading" class="spinner"></span>
                {{ actionLoading ? 'Guardando...' : 'Guardar configuración' }}
              </button>
            </div>
          </div>

          <div class="modal-actions" v-else>
            <button type="button" class="btn-secondary" @click="closeAssignModal" :disabled="actionLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="actionLoading">
              <span v-if="actionLoading" class="spinner"></span>
              {{ actionLoading ? 'Guardando...' : 'Agregar a tienda' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="isAlertOpen" class="alert-overlay" @click.self="closeAlert">
      <div class="alert-card">
        <h3 class="alert-title">Aviso</h3>
        <p class="alert-message">{{ alertText }}</p>
        <div class="modal-actions">
          <button type="button" class="btn-primary" @click="closeAlert">OK</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { apiFetch, getJSON } from '../../services/api.js';
import { useAuth } from '../../composables/useAuth.js';

const router = useRouter();
const auth = useAuth();
const tenants = ref([]);
const products = ref([]);
const tenantProducts = ref([]);
const selectedTenant = ref('');
const loadingProducts = ref(false);
const loadingTenantProducts = ref(false);
const productError = ref('');
const tenantProductError = ref('');
const actionLoading = ref(false);
const isCatalogModalOpen = ref(false);
const isAssignModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const assignProduct = ref(null);
const assignTenantProduct = ref(null);
const tenantProductForm = ref(emptyTenantProductForm());

const brandId = computed(() => auth.user.value?.brand_id || auth.user.value?.brandID || '');
const brandLabel = computed(() => {
  const tenantWithBrand = tenants.value.find((item) => item.id === selectedTenant.value || item.brand?.name || item.brand_name || item.brandName);
  return tenantWithBrand?.brand?.name || tenantWithBrand?.brand?.Name || tenantWithBrand?.brand_name || tenantWithBrand?.brandName || '';
});
const selectedTenantLabel = computed(() => {
  const tenant = tenants.value.find((item) => item.id === selectedTenant.value);
  return tenant?.subdomain || tenant?.id || 'la tienda seleccionada';
});

const tenantProductMap = computed(() => new Map(tenantProducts.value.map((item) => [item.product_id, item])));
const orderedProducts = computed(() => {
  return [...products.value].sort((left, right) => {
    const leftAssigned = !!tenantProductById(productId(left));
    const rightAssigned = !!tenantProductById(productId(right));
    if (leftAssigned === rightAssigned) {
      return productName(left).localeCompare(productName(right));
    }
    return leftAssigned ? 1 : -1;
  });
});

watch(selectedTenant, async () => {
  await loadTenantProducts();
});

function goBack() {
  router.push({ name: 'AdminCatalog' });
}

function openCatalogModal() {
  isCatalogModalOpen.value = true;
}

function closeCatalogModal() {
  isCatalogModalOpen.value = false;
}

function emptyTenantProductForm() {
  return {
    exists: false,
    price_override: null,
    current_stock: 0,
    track_stock: true,
    is_available: true,
    is_active: true
  };
}

function money(value) {
  const numeric = Number(value || 0);
  return numeric.toLocaleString('es-AR', { style: 'currency', currency: 'ARS' });
}

function productId(product) {
  return product?.id || product?.ID || '';
}

function productName(product) {
  return product?.name || product?.Name || '';
}

function productDescription(product) {
  return product?.description || product?.Description || '';
}

function productImage(product) {
  return product?.image_url || product?.ImageURL || '';
}

function productBasePrice(product) {
  return product?.base_price ?? product?.BasePrice ?? 0;
}

function effectivePrice(item) {
  if (item.price_override !== null && item.price_override !== undefined) {
    return item.price_override;
  }
  return productBasePrice(item.product || productById(item.product_id)) || 0;
}

function productById(id) {
  return products.value.find((product) => productId(product) === id) || null;
}

function productActive(product) {
  return product?.is_active ?? product?.IsActive ?? false;
}

function tenantProductById(productID) {
  return tenantProductMap.value.get(productID) || null;
}

function productHeaders() {
  return brandId.value ? { 'X-Brand-ID': brandId.value } : {};
}

function tenantHeaders() {
  return selectedTenant.value ? { 'X-Tenant-ID': selectedTenant.value } : {};
}

async function loadTenants() {
  const url = brandId.value ? `/tenants?brand_id=${brandId.value}` : '/tenants';
  tenants.value = await getJSON(url);
}

async function loadProducts() {
  loadingProducts.value = true;
  productError.value = '';
  try {
    const response = await apiFetch('/products', {
      method: 'GET',
      headers: productHeaders()
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}));
      throw new Error(payload.message || 'No se pudo cargar el catálogo.');
    }
    products.value = await response.json();
  } catch (err) {
    productError.value = err.message || 'No se pudo cargar el catálogo.';
  } finally {
    loadingProducts.value = false;
  }
}

async function loadTenantProducts() {
  if (!selectedTenant.value) {
    tenantProducts.value = [];
    return;
  }
  loadingTenantProducts.value = true;
  tenantProductError.value = '';
  try {
    const response = await apiFetch('/tenant-products', {
      method: 'GET',
      headers: tenantHeaders()
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}));
      throw new Error(payload.message || 'No se pudo cargar el catálogo de la tienda.');
    }
    tenantProducts.value = await response.json();
  } catch (err) {
    tenantProductError.value = err.message || 'No se pudo cargar el catálogo de la tienda.';
    tenantProducts.value = [];
  } finally {
    loadingTenantProducts.value = false;
  }
}

function openAssignModal(product, existing = null) {
  if (!selectedTenant.value) {
    showAlert('Seleccioná una tienda antes de agregar productos.');
    return;
  }
  assignProduct.value = product;
  assignTenantProduct.value = existing;
  tenantProductForm.value = existing
    ? {
        exists: true,
        price_override: existing.price_override ?? null,
        current_stock: Number(existing.current_stock || 0),
        track_stock: !!existing.track_stock,
        is_available: !!existing.is_available,
        is_active: !!existing.is_active
      }
    : emptyTenantProductForm();
  isCatalogModalOpen.value = false;
  isAssignModalOpen.value = true;
}

function closeAssignModal() {
  isAssignModalOpen.value = false;
  assignProduct.value = null;
  assignTenantProduct.value = null;
}

function showAlert(message) {
  alertText.value = message;
  isAlertOpen.value = true;
}

function closeAlert() {
  isAlertOpen.value = false;
  alertText.value = '';
}

async function submitTenantProduct() {
  const selectedProductID = productId(assignProduct.value);
  if (!selectedProductID) {
    showAlert('Seleccioná un producto válido.');
    return;
  }
  actionLoading.value = true;
  try {
    const payload = {
      product_id: selectedProductID,
      price_override: tenantProductForm.value.price_override === '' ? null : tenantProductForm.value.price_override,
      current_stock: Number(tenantProductForm.value.current_stock || 0),
      track_stock: tenantProductForm.value.track_stock,
      is_available: tenantProductForm.value.is_available
    };
    if (tenantProductForm.value.exists) {
      payload.is_active = tenantProductForm.value.is_active;
    }

    const response = await apiFetch(tenantProductForm.value.exists ? `/tenant-products/${selectedProductID}` : '/tenant-products', {
      method: tenantProductForm.value.exists ? 'PUT' : 'POST',
      headers: tenantHeaders(),
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const errorPayload = await response.json().catch(() => ({}));
      throw new Error(errorPayload.message || 'No se pudo guardar la configuración.');
    }
    await loadTenantProducts();
    closeAssignModal();
    closeCatalogModal();
  } catch (err) {
    showAlert(err.message || 'No se pudo guardar la configuración.');
  } finally {
    actionLoading.value = false;
  }
}

async function removeTenantProduct(item, fromModal = false) {
  const confirmed = confirm(`¿Quitar ${item.product?.name || item.product_id} de esta tienda?`);
  if (!confirmed) {
    return;
  }
  if (fromModal) {
    actionLoading.value = true;
  }
  try {
    const response = await apiFetch(`/tenant-products/${item.product_id}`, {
      method: 'DELETE',
      headers: tenantHeaders()
    });
    if (!response.ok) {
      const errorPayload = await response.json().catch(() => ({}));
      throw new Error(errorPayload.message || 'No se pudo quitar el producto.');
    }
    await loadTenantProducts();
    if (fromModal) {
      closeAssignModal();
    }
  } catch (err) {
    showAlert(err.message || 'No se pudo quitar el producto.');
  } finally {
    if (fromModal) {
      actionLoading.value = false;
    }
  }
}

onMounted(async () => {
  try {
    await loadTenants();
    await loadProducts();
  } catch (err) {
    showAlert(err.message || 'No se pudo inicializar catálogo.');
  }
});
</script>

<style scoped>
.admin-catalog-tenant {
  padding: 1.5rem;
}

.catalog-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.panel-heading__actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.tenant-picker-card,
.empty-state-card {
  margin-bottom: 20px;
}

.controls-grid--tenant {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(240px, 0.8fr);
  gap: 18px;
  align-items: stretch;
}

.tenant-summary {
  display: grid;
  gap: 4px;
  padding: 14px 16px;
  border-radius: 16px;
  background: rgba(59, 130, 246, 0.08);
  border: 1px solid rgba(96, 165, 250, 0.16);
}

.tenant-summary__label,
.meta-label {
  font-size: 0.78rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.tenant-summary__meta {
  color: var(--text-muted);
}

.empty-state-card {
  margin-bottom: 20px;
}

.panel-heading {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  margin-bottom: 18px;
}

.panel-heading--stacked {
  align-items: flex-start;
}

.panel-heading h3,
.empty-state-card h3,
.product-card__heading h3 {
  margin: 0;
}

.panel-heading p,
.empty-state-card p,
.product-card__heading p {
  margin: 6px 0 0;
  color: var(--text-muted);
}

.tenant-product-grid {
  display: grid;
  gap: 14px;
}

.tenant-product-card {
  display: grid;
  grid-template-columns: minmax(0, 280px) minmax(0, 1fr);
  gap: 18px;
  padding: 16px;
  border-radius: 18px;
  background: rgba(15, 23, 42, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: var(--shadow);
}

.tenant-product-card__media {
  border-radius: 14px;
  overflow: hidden;
  min-height: 140px;
}

.tenant-product-card__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.tenant-product-card__image--placeholder {
  background: linear-gradient(135deg, rgba(96, 165, 250, 0.18), rgba(15, 23, 42, 0.18));
}

.tenant-product-card__body {
  display: grid;
  grid-template-rows: auto auto 1fr auto;
  gap: 16px;
  align-content: start;
}

.tenant-product-card__heading {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
}

.tenant-product-card__heading h3,
.base-product-row h4 {
  margin: 0;
}

.tenant-product-card__heading p,
.base-product-row p {
  margin: 6px 0 0;
  color: var(--text-muted);
}

.tenant-product-card__heading p,
.base-product-row p {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 2.5em;
}

.tenant-product-card__meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.tenant-product-card__meta strong {
  display: block;
  margin-top: 4px;
}

.tenant-product-card__actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  align-items: stretch;
}

.tenant-product-card__actions .btn-secondary,
.tenant-product-card__actions .btn-danger,
.modal-actions .btn-secondary,
.modal-actions .btn-primary,
.modal-actions .btn-danger {
  min-height: 44px;
}

.base-product-row__actions .btn-action-label {
  min-height: 44px;
  width: 100%;
}

.modal-card--catalog {
  width: min(100%, 900px);
}

.base-product-list {
  display: grid;
  gap: 12px;
  max-height: 60vh;
  overflow: auto;
  padding-right: 4px;
}

.base-product-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  padding: 14px 16px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(148, 163, 184, 0.12);
}

.base-product-row__main {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.base-product-row__thumb {
  width: 72px;
  height: 72px;
  border-radius: 14px;
  overflow: hidden;
  flex-shrink: 0;
}

.base-product-row__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.base-product-row__image--placeholder {
  background: linear-gradient(135deg, rgba(96, 165, 250, 0.18), rgba(15, 23, 42, 0.18));
}

.base-product-row__actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.btn-action-label {
  min-width: 110px;
  min-height: 42px;
}

.base-product-row__actions .status-pill {
  min-height: 42px;
}

.btn-fab {
  width: 42px;
  height: 42px;
  padding: 0;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 1.35rem;
  line-height: 1;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 18px;
}

.product-card {
  border-radius: 20px;
  overflow: hidden;
  background: rgba(15, 23, 42, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: var(--shadow);
  display: grid;
}

.product-card__media {
  aspect-ratio: 16 / 10;
  background: rgba(255, 255, 255, 0.03);
}

.product-card__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.product-card__image--placeholder {
  background: linear-gradient(135deg, rgba(96, 165, 250, 0.2), rgba(15, 23, 42, 0.2));
}

.product-card__body {
  padding: 18px;
  display: grid;
  gap: 16px;
}

.product-card__heading {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
}

.product-card__heading h3 {
  font-size: 1.05rem;
}

.product-card__meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.product-card__meta strong {
  display: block;
  margin-top: 4px;
}

.product-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.modal-card--wide {
  width: min(100%, 620px);
}

.modal-actions--split {
  justify-content: space-between;
}

.modal-actions__group {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 84px;
  padding: 8px 12px;
  border-radius: 999px;
  font-size: 0.82rem;
  font-weight: 700;
}

.status-pill--success {
  background: rgba(16, 185, 129, 0.14);
  color: #6ee7b7;
}

.status-pill--muted {
  background: rgba(148, 163, 184, 0.14);
  color: var(--text-muted);
}

@media (max-width: 900px) {
  .controls-grid--tenant {
    grid-template-columns: 1fr;
  }

  .panel-heading,
  .panel-heading--stacked,
  .product-card__heading,
  .tenant-product-card__heading,
  .base-product-row {
    flex-direction: column;
  }

  .tenant-product-card {
    grid-template-columns: 1fr;
  }

  .tenant-product-card__meta {
    grid-template-columns: 1fr;
  }

  .tenant-product-card__actions {
    grid-template-columns: 1fr;
  }

  .product-card__meta {
    grid-template-columns: 1fr;
  }

  .modal-actions--split {
    flex-direction: column;
    align-items: stretch;
  }

  .modal-actions__group {
    justify-content: stretch;
  }
}
</style>