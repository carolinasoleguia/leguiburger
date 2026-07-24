<template>
  <section>
    <div class="section-header">
      <div class="section-title">
        <h2>Tenants</h2>
        <p>Gestiona los negocios que contratan el sistema.</p>
      </div>
      <button class="btn-primary" @click="openCreateModal">Nuevo tenant</button>
    </div>

    <div class="card">
      <div v-if="loading" class="centered-loading">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="tenants.length === 0" class="status-text">No hay tenants registrados.</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Marca / Comercio</th>
            <th>Subdominio</th>
            <th>CUIT / Tax ID</th>
            <th>Estado</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(tenant, index) in tenants" :key="tenant.id">
            <td>{{ index + 1 }}</td>
            <td>{{ tenant.brand?.name || tenant.name || '—' }}</td>
            <td>{{ tenant.subdomain }}</td>
            <td>{{ tenant.brand?.tax_id || tenant.tax_id || '—' }}</td>
            <td>{{ tenant.active ? 'Activo' : 'Inactivo' }}</td>
            <td class="table-actions">
              <button class="btn-secondary" @click="openEditModal(tenant)">Editar</button>
              <button :class="[tenant.active ? 'btn-danger' : 'btn-success']" @click="openDeleteModal(tenant)">
                {{ tenant.active ? 'Eliminar' : 'Reactivar' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="isCreateModalOpen" class="modal-overlay" @click.self="closeCreateModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeCreateModal" aria-label="Cerrar modal">×</button>
        <h3>Nuevo tenant</h3>
        <p>Registra un nuevo tenant asociándolo a una marca existente.</p>

        <form @submit.prevent="submitCreate">
          <div class="input-group">
            <label>Marca</label>
            <select v-model="createForm.brand_id" required>
              <option value="" disabled>Seleccionar marca</option>
              <option v-for="brand in brands" :key="brand.id" :value="brand.id">
                {{ brand.name }} - {{ brand.tax_id }}
              </option>
            </select>
          </div>

          <div class="input-group">
            <label>Subdominio</label>
            <input type="text" v-model="createForm.subdomain" required />
          </div>

          <div v-if="createError" class="status-text error">{{ createError }}</div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeCreateModal" :disabled="createLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="createLoading || !createForm.brand_id">
              <span v-if="createLoading" class="spinner"></span>
              {{ createLoading ? 'Guardando...' : 'Crear tenant' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="isEditModalOpen" class="modal-overlay" @click.self="closeEditModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeEditModal" aria-label="Cerrar modal">×</button>
        <h3>Editar Tenant</h3>
        <p>Actualiza el subdominio para el comercio <strong>{{ editingTenant.brand?.name || editingTenant.subdomain }}</strong>.</p>

        <form @submit.prevent="submitEdit">
          <div class="input-group">
            <label>Marca</label>
            <select v-model="editForm.brand_id" required>
              <option value="" disabled>Seleccionar marca</option>
              <option v-for="brand in brands" :key="brand.id" :value="brand.id">
                {{ brand.name }} - {{ brand.tax_id }}
              </option>
            </select>
          </div>

          <div class="input-group">
            <label>Subdominio</label>
            <input type="text" v-model="editForm.subdomain" required />
          </div>

          <div v-if="modalError" class="status-text error">{{ modalError }}</div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeEditModal" :disabled="loadingEdit">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="loadingEdit || !editForm.brand_id">
              <span v-if="loadingEdit" class="spinner"></span>
              {{ loadingEdit ? 'Guardando...' : 'Guardar cambios' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="isDeleteModalOpen" class="modal-overlay" @click.self="closeDeleteModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeDeleteModal" aria-label="Cerrar modal">×</button>
        <h3>{{ deletingTenant.active ? 'Confirmar eliminación' : 'Confirmar reactivación' }}</h3>
        <p>
          {{ deletingTenant.active ? 'Esta acción desactivará' : 'Esta acción reactivará' }}
          el tenant <strong>{{ deletingTenant.brand?.name || deletingTenant.subdomain }}</strong>.
        </p>

        <div v-if="modalError" class="status-text error">{{ modalError }}</div>

        <div class="modal-actions">
          <button type="button" class="btn-secondary" @click="closeDeleteModal" :disabled="loadingDelete">Cancelar</button>
          <button
            type="button"
            :class="[deletingTenant.active ? 'btn-danger' : 'btn-success']"
            @click="confirmDelete"
            :disabled="loadingDelete"
          >
            <span v-if="loadingDelete" class="spinner"></span>
            {{ loadingDelete ? (deletingTenant.active ? 'Eliminando...' : 'Reactivando...') : (deletingTenant.active ? 'Eliminar' : 'Reactivar') }}
          </button>
        </div>
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
import { ref, onMounted } from 'vue';
import { getJSON, postJSON, putJSON, deleteJSON } from '../../services/api.js';

const tenants = ref([]);
const brands = ref([]);
const loading = ref(true);
const error = ref('');
const isCreateModalOpen = ref(false);
const isEditModalOpen = ref(false);
const isDeleteModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const editingTenant = ref({});
const deletingTenant = ref({});
const editForm = ref({ subdomain: '', brand_id: '' });
const createForm = ref({ subdomain: '', brand_id: '' });
const loadingEdit = ref(false);
const createLoading = ref(false);
const loadingDelete = ref(false);
const modalError = ref('');
const createError = ref('');

async function loadBrands() {
  try {
    brands.value = await getJSON('/brands');
  } catch (err) {
    console.error('No se pudieron cargar las brands:', err);
  }
}

async function loadTenants() {
  try {
    tenants.value = await getJSON('/tenants');
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los tenants.';
  } finally {
    loading.value = false;
  }
}

function openCreateModal() {
  isCreateModalOpen.value = true;
  createError.value = '';
  createForm.value = { subdomain: '', brand_id: '' };
}

function closeCreateModal() {
  isCreateModalOpen.value = false;
  createError.value = '';
}

function showAlert(message) {
  alertText.value = message;
  isAlertOpen.value = true;
}

function closeAlert() {
  isAlertOpen.value = false;
  alertText.value = '';
}

async function submitCreate() {
  createLoading.value = true;
  createError.value = '';

  try {
    await postJSON('/tenants', createForm.value);
    await loadTenants();
    closeCreateModal();
  } catch (err) {
    showAlert(err.message || 'Error al crear el tenant.');
  } finally {
    createLoading.value = false;
  }
}

function openEditModal(tenant) {
  editingTenant.value = tenant;
  editForm.value = {
    subdomain: tenant.subdomain || '',
    brand_id: tenant.brand?.id || ''
  };
  modalError.value = '';
  isEditModalOpen.value = true;
}

function closeEditModal() {
  isEditModalOpen.value = false;
  editingTenant.value = {};
  editForm.value = { subdomain: '' };
  modalError.value = '';
}

async function submitEdit() {
  if (!editingTenant.value.id) return;
  loadingEdit.value = true;
  modalError.value = '';

  try {
    await putJSON(`/tenants/${editingTenant.value.id}`, {
      subdomain: editForm.value.subdomain,
      brand_id: editForm.value.brand_id || undefined
    });
    await loadTenants();
    closeEditModal();
  } catch (err) {
    showAlert(err.message || 'Error al guardar los cambios.');
  } finally {
    loadingEdit.value = false;
  }
}

function openDeleteModal(tenant) {
  deletingTenant.value = tenant;
  modalError.value = '';
  isDeleteModalOpen.value = true;
}

function closeDeleteModal() {
  isDeleteModalOpen.value = false;
  deletingTenant.value = {};
  modalError.value = '';
}

async function confirmDelete() {
  if (!deletingTenant.value.id) return;
  loadingDelete.value = true;
  modalError.value = '';

  try {
    if (deletingTenant.value.active) {
      await deleteJSON(`/tenants/${deletingTenant.value.id}`);
    } else {
      await putJSON(`/tenants/${deletingTenant.value.id}`, {
        active: true,
      });
    }
    await loadTenants();
    closeDeleteModal();
  } catch (err) {
    showAlert(
      err.message ||
        (deletingTenant.value.active
          ? 'Error al eliminar el tenant.'
          : 'Error al reactivar el tenant.')
    );
  } finally {
    loadingDelete.value = false;
  }
}

onMounted(async () => {
  await Promise.all([loadBrands(), loadTenants()]);
});
</script>
