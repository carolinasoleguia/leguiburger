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
            <th>Nombre</th>
            <th>Email</th>
            <th>Tenant</th>
            <th>Rol</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="admin in admins" :key="admin.id">
            <td>{{ admin.first_name }} {{ admin.last_name }}</td>
            <td>{{ admin.email }}</td>
            <td>{{ admin.tenant_id }}</td>
            <td>{{ admin.role }}</td>
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
import { getJSON, postJSON } from '../../services/api.js';

const admins = ref([]);
const tenants = ref([]);
const loading = ref(true);
const error = ref('');
const isCreateModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const createLoading = ref(false);
const createError = ref('');
const createForm = ref({
  tenant_id: '',
  first_name: '',
  last_name: '',
  email: '',
  password: '',
  phone: '',
  role: 'employee'
});

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
    await postJSON('/employees', createForm.value);
    await loadAdmins();
    closeCreateModal();
  } catch (err) {
    showAlert(err.message || 'Error al crear el administrador.');
  } finally {
    createLoading.value = false;
  }
}

onMounted(async () => {
  await Promise.all([loadTenants(), loadAdmins()]);
});
</script>
