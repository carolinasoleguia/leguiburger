<template>
  <section>
    <PageHeading title="Marcas" description="La identidad de tus negocios y los locales que forman parte de cada marca.">
      <button class="btn-primary" @click="openCreateModal">Nueva marca</button>
    </PageHeading>

    <div class="card">
      <StatePanel v-if="loading" kind="loading" />
      <div v-else-if="error" class="status-text error" role="alert">{{ error }}</div>
      <div v-else-if="brands.length === 0" class="status-text">Todavía no hay marcas registradas.</div>
      <div v-else class="table-scroll"><table class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Nombre</th>
            <th>CUIT</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(brand, index) in brands" :key="brand.id">
            <td>{{ index + 1 }}</td>
            <td>{{ brand.name }}</td>
            <td>{{ brand.tax_id }}</td>
            <td class="table-actions">
              <button class="btn-secondary" @click="openEditModal(brand)">Editar</button>
              <button class="btn-danger" @click="deleteBrand(brand.id)">Eliminar</button>
            </td>
          </tr>
        </tbody>
      </table></div>
    </div>

    <div v-if="isCreateModalOpen || isEditModalOpen" class="modal-overlay" @click.self="closeModal">
      <div v-dialog-care class="modal-card">
        <button type="button" class="modal-close" @click="closeModal" aria-label="Cerrar modal">×</button>
        <h3>{{ isEditModalOpen ? 'Editar marca' : 'Nueva marca' }}</h3>
        <p>{{ isEditModalOpen ? 'Actualiza los datos de la marca.' : 'Registra una marca para asociarla a tenants.' }}</p>

        <form v-form-care @submit.prevent="isEditModalOpen ? submitEdit() : submitCreate()">
          <div class="input-group">
            <label>Nombre</label>
            <input type="text" v-model="activeForm.name" required />
          </div>

          <div class="input-group">
            <label>Tax ID</label>
            <input type="text" v-model="activeForm.tax_id" required />
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeModal" :disabled="modalLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="modalLoading">
              <span v-if="modalLoading" class="spinner"></span>
              {{ modalLoading ? 'Guardando...' : isEditModalOpen ? 'Guardar cambios' : 'Crear marca' }}
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
import PageHeading from "../../components/PageHeading.vue";
import StatePanel from "../../components/StatePanel.vue";
import { confirmAction } from "../../services/confirmation.js";
import { ref, onMounted, computed } from 'vue';
import { getJSON, postJSON, putJSON, deleteJSON } from '../../services/api.js';

const brands = ref([]);
const loading = ref(true);
const error = ref('');
const isCreateModalOpen = ref(false);
const isEditModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const editingBrand = ref(null);
const modalLoading = ref(false);
const createForm = ref({ name: '', tax_id: '' });
const editForm = ref({ name: '', tax_id: '' });

const activeForm = computed(() => (isEditModalOpen.value ? editForm.value : createForm.value));
const modalError = ref('');

async function loadBrands() {
  try {
    brands.value = await getJSON('/brands');
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar las brands.';
  } finally {
    loading.value = false;
  }
}

function openCreateModal() {
  isCreateModalOpen.value = true;
  isEditModalOpen.value = false;
  modalError.value = '';
  createForm.value = { name: '', tax_id: '' };
}

function openEditModal(brand) {
  isEditModalOpen.value = true;
  isCreateModalOpen.value = false;
  editingBrand.value = brand;
  editForm.value = { name: brand.name, tax_id: brand.tax_id };
}

function closeModal() {
  isCreateModalOpen.value = false;
  isEditModalOpen.value = false;
  editingBrand.value = null;
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
  modalLoading.value = true;
  modalError.value = '';

  try {
    await postJSON('/brands', createForm.value);
    await loadBrands();
    closeModal();
  } catch (err) {
    showAlert(err.message || 'Error al crear la marca.');
  } finally {
    modalLoading.value = false;
  }
}

async function submitEdit() {
  if (!editingBrand.value) return;
  modalLoading.value = true;
  modalError.value = '';

  try {
    await putJSON(`/brands/${editingBrand.value.id}`, editForm.value);
    await loadBrands();
    closeModal();
  } catch (err) {
    showAlert(err.message || 'Error al editar la marca.');
  } finally {
    modalLoading.value = false;
  }
}

async function deleteBrand(id) {
  const confirmed = await confirmAction('¿Seguro que querés eliminar esta marca?');
  if (!confirmed) return;
  try {
    await deleteJSON(`/brands/${id}`);
    await loadBrands();
  } catch (err) {
    showAlert(err.message || 'No se pudo eliminar la brand.');
  }
}

onMounted(loadBrands);
</script>
