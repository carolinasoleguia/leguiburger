<template>
  <section>
    <div class="section-header">
      <div class="section-title">
        <h2>Administradores</h2>
        <p>Revisa los administradores de los tenants registrados.</p>
      </div>
      <button class="btn-primary" @click="openCreateModal">Nuevo administrador</button>
    </div>

    <div class="card">
      <div v-if="loading" class="centered-loading">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="admins.length === 0" class="status-text">No se encontraron administradores.</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Nombre</th>
            <th>Email</th>
            <th>Tenant</th>
            <th>Rol</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(admin, index) in admins" :key="admin.id">
            <td>{{ index + 1 }}</td>
            <td>{{ admin.first_name }} {{ admin.last_name }}</td>
            <td>{{ admin.email }}</td>
            <td>{{ getTenantLabel(admin.tenant_id) }}</td>
            <td>{{ admin.role }}</td>
            <td class="table-actions">
              <button class="btn-secondary" @click="openEditModal(admin)">Editar</button>
              <button
                :class="[admin.is_active ? 'btn-danger' : 'btn-success']"
                @click="openDeleteModal(admin)"
              >
                {{ admin.is_active ? 'Eliminar' : 'Reactivar' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="isCreateModalOpen" class="modal-overlay" @click.self="closeCreateModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeCreateModal" aria-label="Cerrar modal">×</button>
        <h3>Nuevo administrador</h3>
        <p>Registra un administrador para un tenant existente.</p>

        <form @submit.prevent="submitCreate">
          <div class="input-group">
            <label>Tenant</label>
            <select v-model="createForm.tenant_id">
              <option value="">Seleccionar tenant</option>
              <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
                {{ tenant.brand?.name || tenant.subdomain }} — {{ tenant.subdomain }}
              </option>
            </select>
          </div>

          <div class="input-group">
            <label>Nombre</label>
            <input type="text" v-model="createForm.first_name" required />
          </div>

          <div class="input-group">
            <label>Apellido</label>
            <input type="text" v-model="createForm.last_name" required />
          </div>

          <div class="input-group">
            <label>Email</label>
            <input type="email" v-model="createForm.email" required />
          </div>

          <div class="input-group">
            <label>Contraseña</label>
            <input type="password" v-model="createForm.password" required />
          </div>

          <div class="input-group">
            <label>Teléfono</label>
            <input type="text" v-model="createForm.phone" />
          </div>

          <div class="input-group">
            <label>Rol</label>
            <select v-model="createForm.role" required>
              <option value="employee">Empleado</option>
              <option value="admin">Administrador</option>
              <option value="owner">Owner</option>
            </select>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeCreateModal" :disabled="createLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="createLoading">
              <span v-if="createLoading" class="spinner"></span>
              {{ createLoading ? 'Guardando...' : 'Crear administrador' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="isEditModalOpen" class="modal-overlay" @click.self="closeEditModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeEditModal" aria-label="Cerrar modal">×</button>
        <h3>Editar administrador</h3>
        <p>Actualiza los datos del administrador seleccionado.</p>

        <form @submit.prevent="submitEdit">
          <div class="input-group">
            <label>Nombre</label>
            <input type="text" v-model="editForm.first_name" required />
          </div>

          <div class="input-group">
            <label>Apellido</label>
            <input type="text" v-model="editForm.last_name" required />
          </div>

          <div class="input-group">
            <label>Email</label>
            <input type="email" v-model="editForm.email" required />
          </div>

          <div class="input-group">
            <label>Contraseña (dejar vacío para no cambiar)</label>
            <input type="password" v-model="editForm.password" />
          </div>

          <div class="input-group">
            <label>Teléfono</label>
            <input type="text" v-model="editForm.phone" />
          </div>

          <div class="input-group">
            <label>Rol</label>
            <select v-model="editForm.role" required>
              <option value="employee">Empleado</option>
              <option value="admin">Administrador</option>
              <option value="owner">Owner</option>
            </select>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeEditModal" :disabled="loadingEdit">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="loadingEdit">
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
        <h3>{{ deletingAdmin.is_active ? 'Confirmar eliminación' : 'Confirmar reactivación' }}</h3>
        <p>
          {{ deletingAdmin.is_active ? 'Esta acción eliminará' : 'Esta acción reactivará' }}
          al administrador <strong>{{ deletingAdmin.first_name }} {{ deletingAdmin.last_name }}</strong>.
        </p>

        <div class="modal-actions">
          <button type="button" class="btn-secondary" @click="closeDeleteModal" :disabled="loadingDelete">Cancelar</button>
          <button
            type="button"
            :class="[deletingAdmin.is_active ? 'btn-danger' : 'btn-success']"
            @click="confirmDelete"
            :disabled="loadingDelete"
          >
            <span v-if="loadingDelete" class="spinner"></span>
            {{ loadingDelete ? (deletingAdmin.is_active ? 'Eliminando...' : 'Reactivando...') : (deletingAdmin.is_active ? 'Eliminar' : 'Reactivar') }}
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

const admins = ref([]);
const tenants = ref([]);
const loading = ref(true);
const error = ref('');
const isCreateModalOpen = ref(false);
const isEditModalOpen = ref(false);
const isDeleteModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const createLoading = ref(false);
const loadingEdit = ref(false);
const loadingDelete = ref(false);
const createError = ref('');
const editingAdmin = ref(null);
const deletingAdmin = ref({});
const createForm = ref({
  tenant_id: '',
  first_name: '',
  last_name: '',
  email: '',
  password: '',
  phone: '',
  role: 'employee'
});
const editForm = ref({
  first_name: '',
  last_name: '',
  email: '',
  password: '',
  phone: '',
  role: 'employee'
});
const editError = ref('');

async function loadAdmins() {
  try {
    admins.value = await getJSON('/employees');
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los administradores.';
  } finally {
    loading.value = false;
  }
}

async function loadTenants() {
  try {
    tenants.value = await getJSON('/tenants');
  } catch (err) {
    console.error('No se pudieron cargar los tenants:', err);
  }
}

function openCreateModal() {
  isCreateModalOpen.value = true;
  createError.value = '';
  createForm.value = {
    tenant_id: '',
    first_name: '',
    last_name: '',
    email: '',
    password: '',
    phone: '',
    role: 'employee'
  };
}

function closeCreateModal() {
  isCreateModalOpen.value = false;
  createError.value = '';
}

function openEditModal(admin) {
  editingAdmin.value = admin;
  isEditModalOpen.value = true;
  editError.value = '';
  editForm.value = {
    first_name: admin.first_name || '',
    last_name: admin.last_name || '',
    email: admin.email || '',
    password: '',
    phone: admin.phone || '',
    role: admin.role || 'employee'
  };
}

function closeEditModal() {
  isEditModalOpen.value = false;
  editingAdmin.value = null;
  editError.value = '';
}

function openDeleteModal(admin) {
  deletingAdmin.value = admin;
  isDeleteModalOpen.value = true;
}

function closeDeleteModal() {
  isDeleteModalOpen.value = false;
  deletingAdmin.value = {};
}

function showAlert(message) {
  alertText.value = message;
  isAlertOpen.value = true;
}

function closeAlert() {
  isAlertOpen.value = false;
  alertText.value = '';
}

function getTenantLabel(tenantId) {
  if (!tenantId) {
    return 'Global';
  }

  const tenant = tenants.value.find((item) => item.id === tenantId);

  if (!tenant) {
    return tenantId;
  }

  return tenant.brand?.name || tenant.name || tenant.subdomain || tenant.id;
}

async function submitCreate() {
  createLoading.value = true;
  createError.value = '';

  try {
    await postJSON('/employees', createForm.value);
    await loadAdmins();
    closeCreateModal();
  } catch (err) {
    showAlert(err.message || 'Error al crear el administrador.');
  } finally {
    createLoading.value = false;
  }
}

async function submitEdit() {
  if (!editingAdmin.value?.id) return;
  loadingEdit.value = true;
  editError.value = '';

  try {
    await putJSON(`/employees/${editingAdmin.value.id}`, {
      first_name: editForm.value.first_name,
      last_name: editForm.value.last_name,
      email: editForm.value.email,
      password: editForm.value.password || undefined,
      phone: editForm.value.phone,
      role: editForm.value.role
    });
    await loadAdmins();
    closeEditModal();
  } catch (err) {
    showAlert(err.message || 'Error al guardar los cambios.');
  } finally {
    loadingEdit.value = false;
  }
}

async function confirmDelete() {
  if (!deletingAdmin.value?.id) return;
  loadingDelete.value = true;

  try {
    if (deletingAdmin.value.is_active) {
      await deleteJSON(`/employees/${deletingAdmin.value.id}`);
    } else {
      await putJSON(`/employees/${deletingAdmin.value.id}`, { is_active: true });
    }
    await loadAdmins();
    closeDeleteModal();
  } catch (err) {
    closeDeleteModal();
    showAlert(
      err.message ||
        (deletingAdmin.value.is_active ? 'Error al eliminar el administrador.' : 'Error al reactivar el administrador.')
    );
  } finally {
    loadingDelete.value = false;
  }
}

onMounted(async () => {
  await Promise.all([loadTenants(), loadAdmins()]);
});
</script>
