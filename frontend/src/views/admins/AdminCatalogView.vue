<template>
  <section class="admin-catalog">
    <div class="section-header">
      <div class="section-title">
        <h2>Gestión de catálogo</h2>
        <p>Definí la oferta base de la marca y ajustá disponibilidad o precios por sucursal cuando haga falta.</p>
      </div>
      <div v-if="!initializingCatalog" class="catalog-actions">
        <button class="btn-primary" @click="openProductModal">Nuevo producto</button>
      </div>
    </div>

    <div v-if="initializingCatalog" class="card centered-loading catalog-initial-loading">
      <span class="spinner-large"></span>
    </div>

    <nav v-else class="catalog-tabs" aria-label="Secciones de gestión de catálogo">
      <router-link :to="{ name: 'AdminCatalog' }" class="catalog-tab" exact-active-class="catalog-tab--active">
        {{ brandCatalogLabel }}
      </router-link>
      <router-link :to="{ name: 'AdminCatalogTenants' }" class="catalog-tab" active-class="catalog-tab--active">
        Catálogo por sucursal
      </router-link>
    </nav>

    <div v-if="!initializingCatalog" class="card">
      <div class="panel-heading">
        <div>
          <h3>{{ brandCatalogLabel }}</h3>
          <p>{{ products.length }} productos registrados como oferta general de la marca</p>
        </div>
      </div>

      <div v-if="loadingProducts" class="centered-loading compact">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="productError" class="status-text error">{{ productError }}</div>
      <div v-else-if="products.length === 0" class="status-text">Todavía no hay productos en el catálogo.</div>

      <div v-else class="table-shell">
        <table class="data-table compact-table catalog-table">
          <thead>
            <tr>
              <th>Producto</th>
              <th>Precio base</th>
              <th>Estado</th>
              <th class="actions-column">Acciones</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="product in products" :key="productId(product)">
              <td class="product-cell">
                <div class="product-cell__content">
                  <img v-if="productImage(product)" :src="productImage(product)" :alt="productName(product)" class="product-thumb" />
                  <div>
                    <strong>{{ productName(product) }}</strong>
                    <span>{{ productDescription(product) || '-' }}</span>
                  </div>
                </div>
              </td>
              <td>{{ money(productBasePrice(product)) }}</td>
              <td class="status-cell status-column">
                <span class="status-pill" :class="productActive(product) ? 'status-pill--success' : 'status-pill--muted'">
                  {{ productActive(product) ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
              <td class="table-actions actions-column">
                <div class="table-actions__inner">
                  <button class="btn-secondary catalog-action-button" @click="openProductModal(product)">Editar</button>
                  <button class="btn-danger catalog-action-button" @click="deleteProduct(product)">Eliminar</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="isProductModalOpen" class="modal-overlay" @click.self="closeProductModal">
      <div class="modal-card modal-card--product">
        <button type="button" class="modal-close" @click="closeProductModal">×</button>
        <h3>{{ productForm.id ? 'Editar producto' : 'Nuevo producto' }}</h3>
        <p>Estos datos pertenecen al catálogo general de la marca.</p>

        <form class="product-modal-form" @submit.prevent="submitProduct">
          <div class="input-group">
            <label>Nombre</label>
            <input v-model="productForm.name" type="text" required />
          </div>
          <div class="input-group">
            <label>Descripción</label>
            <textarea v-model="productForm.description"></textarea>
          </div>
          <div class="input-grid">
            <div class="input-group">
              <label>Precio base</label>
              <input v-model.number="productForm.base_price" type="number" min="0" step="0.01" required />
            </div>
            <div class="input-group">
              <label>Imagen</label>
              <input type="file" accept="image/*" @change="handleImageChange" />
              <span class="file-hint">{{ productForm.image_name || 'No se eligió ningún archivo' }}</span>
            </div>
          </div>

          <div class="image-preview" v-if="previewImageSrc">
            <span class="image-preview__label">Vista previa</span>
            <img :src="previewImageSrc" :alt="productForm.name || 'Vista previa del producto'" class="image-preview__img" />
          </div>

          <div v-if="productForm.id" class="product-status-toggle">
            <div>
              <span class="product-status-toggle__label">Estado del producto</span>
              <strong>{{ productForm.is_active ? 'Activo' : 'Inactivo' }}</strong>
            </div>
            <label class="switch-control" for="product-active">
              <input id="product-active" v-model="productForm.is_active" type="checkbox" />
              <span class="switch-control__track">
                <span class="switch-control__thumb"></span>
              </span>
            </label>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeProductModal" :disabled="actionLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="actionLoading">
              <span v-if="actionLoading" class="spinner"></span>
              {{ actionLoading ? 'Guardando...' : 'Guardar' }}
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { apiFetch, getJSON } from '../../services/api.js';
import { useAuth } from '../../composables/useAuth.js';

const auth = useAuth();
const tenants = ref([]);
const products = ref([]);
const initializingCatalog = ref(true);
const loadingProducts = ref(false);
const productError = ref('');
const actionLoading = ref(false);
const isProductModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const productForm = ref(emptyProductForm());
const previewObjectUrl = ref('');

const brandId = computed(() => auth.user.value?.brand_id || auth.user.value?.brandID || '');
const brandName = computed(() => {
  const user = auth.user.value || {};
  const tenantWithBrand = tenants.value.find((tenant) => tenant.brand?.name || tenant.brand?.Name || tenant.brand_name || tenant.brandName);
  return (
    user.brand?.name ||
    user.brand?.Name ||
    user.brand_name ||
    user.brandName ||
    tenantWithBrand?.brand?.name ||
    tenantWithBrand?.brand?.Name ||
    tenantWithBrand?.brand_name ||
    tenantWithBrand?.brandName ||
    ''
  );
});
const brandCatalogLabel = computed(() => (brandName.value ? `Catálogo de ${brandName.value}` : 'Catálogo de marca'));
const previewImageSrc = computed(() => previewObjectUrl.value || productForm.value.image_url || '');

function emptyProductForm() {
  return {
    id: '',
    name: '',
    description: '',
    base_price: 0,
    image_url: '',
    image_file: null,
    image_name: '',
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

function productActive(product) {
  return product?.is_active ?? product?.IsActive ?? false;
}

function productHeaders() {
  return brandId.value ? { 'X-Brand-ID': brandId.value } : {};
}

async function loadTenants() {
  if (!brandId.value) {
    return;
  }
  tenants.value = await getJSON(`/tenants?brand_id=${brandId.value}`);
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

function openProductModal(product = null) {
  revokePreviewObjectUrl();
  productForm.value = product
    ? {
        id: productId(product),
        name: productName(product),
        description: productDescription(product),
        base_price: Number(productBasePrice(product)),
        image_url: productImage(product),
        image_file: null,
        image_name: productImage(product) ? 'Imagen actual' : '',
        is_active: productActive(product)
      }
    : emptyProductForm();
  isProductModalOpen.value = true;
}

function closeProductModal() {
  revokePreviewObjectUrl();
  isProductModalOpen.value = false;
}

function handleImageChange(event) {
  const [file] = event.target.files || [];
  revokePreviewObjectUrl();
  productForm.value.image_file = file || null;
  productForm.value.image_name = file ? file.name : '';
  previewObjectUrl.value = file ? URL.createObjectURL(file) : '';
  if (!file) {
    productForm.value.image_url = '';
  }
}

function revokePreviewObjectUrl() {
  if (previewObjectUrl.value) {
    URL.revokeObjectURL(previewObjectUrl.value);
    previewObjectUrl.value = '';
  }
}

function showAlert(message) {
  alertText.value = message;
  isAlertOpen.value = true;
}

function closeAlert() {
  isAlertOpen.value = false;
  alertText.value = '';
}

async function deleteProduct(product) {
  const confirmed = confirm(`¿Eliminar ${productName(product)} del catálogo base?`);
  if (!confirmed) {
    return;
  }

  actionLoading.value = true;
  try {
    const response = await apiFetch(`/products/${productId(product)}`, {
      method: 'DELETE',
      headers: productHeaders()
    });
    if (!response.ok) {
      const errorPayload = await response.json().catch(() => ({}));
      throw new Error(errorPayload.message || 'No se pudo eliminar el producto.');
    }
    await loadProducts();
  } catch (err) {
    showAlert(err.message || 'No se pudo eliminar el producto.');
  } finally {
    actionLoading.value = false;
  }
}

async function submitProduct() {
  actionLoading.value = true;
  try {
    const endpoint = `/products${productForm.value.id ? `/${productForm.value.id}` : ''}`;
    const payload = new FormData();
    payload.append('name', productForm.value.name);
    payload.append('description', productForm.value.description);
    payload.append('base_price', String(Number(productForm.value.base_price || 0)));
    if (brandId.value) {
      payload.append('brand_id', brandId.value);
    }
    if (productForm.value.id) {
      payload.append('is_active', String(productForm.value.is_active));
    }
    if (productForm.value.image_file) {
      payload.append('image_file', productForm.value.image_file);
    }
    if (productForm.value.image_url) {
      payload.append('image_url', productForm.value.image_url);
    }

    const response = await apiFetch(endpoint, {
      method: productForm.value.id ? 'PUT' : 'POST',
      headers: productHeaders(),
      body: payload
    });
    if (!response.ok) {
      const errorPayload = await response.json().catch(() => ({}));
      throw new Error(errorPayload.message || 'No se pudo guardar el producto.');
    }
    await loadProducts();
    closeProductModal();
  } catch (err) {
    showAlert(err.message || 'No se pudo guardar el producto.');
  } finally {
    actionLoading.value = false;
  }
}

onMounted(async () => {
  try {
    await loadTenants();
    await loadProducts();
  } catch (err) {
    showAlert(err.message || 'No se pudo inicializar catálogo.');
  } finally {
    initializingCatalog.value = false;
  }
});

onBeforeUnmount(() => {
  revokePreviewObjectUrl();
});
</script>

<style scoped>
.admin-catalog {
  padding: 1.5rem;
}

.catalog-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.catalog-initial-loading {
  min-height: 220px;
}

.catalog-tabs {
  display: inline-flex;
  gap: 6px;
  padding: 6px;
  margin-bottom: 20px;
  border-radius: 14px;
  background: rgba(15, 23, 42, 0.76);
  border: 1px solid rgba(148, 163, 184, 0.14);
}

.catalog-tab {
  display: inline-flex;
  align-items: center;
  min-height: 42px;
  padding: 0 16px;
  border-radius: 10px;
  color: var(--text-muted);
  font-weight: 700;
  text-decoration: none;
  transition: background 0.2s ease, color 0.2s ease;
}

.catalog-tab:hover,
.catalog-tab--active {
  background: rgba(59, 130, 246, 0.14);
  color: var(--text);
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

.panel-heading h3 {
  margin: 0;
}

.panel-heading p {
  margin: 6px 0 0;
  color: var(--text-muted);
}

.table-shell {
  width: 100%;
  overflow-x: auto;
  border-radius: 18px;
}

.compact {
  min-height: 180px;
}

.compact-table th,
.compact-table td {
  padding: 14px 16px;
}

.catalog-table {
  table-layout: fixed;
}

.catalog-table th:nth-child(1),
.catalog-table td:nth-child(1) {
  width: 48%;
}

.catalog-table th:nth-child(2),
.catalog-table td:nth-child(2) {
  width: 14%;
}

.actions-column {
  width: 240px;
  white-space: nowrap;
}

.product-cell {
  text-align: left !important;
}

.product-cell__content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-cell {
  text-align: center;
  width: 120px;
  white-space: nowrap;
}

.product-cell strong,
.product-cell span {
  display: block;
}

.product-cell span {
  color: var(--text-muted);
  font-size: 0.86rem;
  margin-top: 4px;
}

.file-hint {
  display: block;
  margin-top: 8px;
  color: var(--text-muted);
  font-size: 0.84rem;
}

.image-preview {
  display: grid;
  gap: 10px;
  margin-bottom: 18px;
}

.image-preview__label {
  color: var(--text-muted);
  font-size: 0.88rem;
}

.image-preview__img {
  width: 100%;
  max-height: 220px;
  object-fit: cover;
  border-radius: 16px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: rgba(255, 255, 255, 0.04);
}

.modal-card--product {
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.product-modal-form {
  overflow-y: auto;
  padding-right: 4px;
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

.product-thumb {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  object-fit: cover;
  background: rgba(255, 255, 255, 0.08);
}

.table-actions {
  text-align: right;
  vertical-align: middle;
}

.table-actions__inner {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: nowrap;
}

.catalog-action-button {
  min-height: 42px;
  min-width: 96px;
}

.input-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.toggle-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.product-status-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 0;
  padding: 14px 16px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(148, 163, 184, 0.16);
}

.product-status-toggle__label {
  display: block;
  margin-bottom: 4px;
  color: var(--text-muted);
  font-size: 0.82rem;
}

.product-status-toggle strong {
  color: var(--text);
}

.switch-control {
  display: inline-flex;
  align-items: center;
  width: 54px;
  height: 30px;
  position: relative;
  flex: 0 0 auto;
  cursor: pointer;
}

.switch-control input {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  margin: 0;
  opacity: 0;
  cursor: pointer;
}

.switch-control__track {
  position: relative;
  display: block;
  width: 54px;
  height: 30px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.28);
  border: 1px solid rgba(148, 163, 184, 0.24);
  transition: background 0.2s ease, border-color 0.2s ease;
}

.switch-control__thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #ffffff;
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.28);
  transition: transform 0.2s ease;
}

.switch-control input:checked + .switch-control__track {
  background: rgba(16, 185, 129, 0.42);
  border-color: rgba(110, 231, 183, 0.42);
}

.switch-control input:checked + .switch-control__track .switch-control__thumb {
  transform: translateX(24px);
}

.switch-control input:focus-visible + .switch-control__track {
  outline: 2px solid rgba(96, 165, 250, 0.8);
  outline-offset: 3px;
}

@media (max-width: 1100px) {
}

@media (max-width: 760px) {
  .input-grid,
  .toggle-grid {
    grid-template-columns: 1fr;
  }

  .panel-heading {
    flex-direction: column;
  }

  .table-actions {
    justify-content: flex-start;
  }
}
</style>
