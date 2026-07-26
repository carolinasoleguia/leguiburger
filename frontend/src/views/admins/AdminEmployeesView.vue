<template>
  <section>
    <div class="section-header">
      <div class="section-title">
        <h2>Gestión de empleados</h2>
        <p>Administra los diferentes roles de empleado y sus datos de acceso.</p>
      </div>
      <button class="btn-primary" @click="openCreateModal" :disabled="!selectedTenantId">Nuevo empleado</button>
    </div>

    <div class="card">
      <div class="input-group">
        <label>Tenant</label>
        <select v-model="selectedTenantId" @change="loadEmployees">
          <option value="">Seleccionar tenant</option>
          <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
            {{ tenant.subdomain }}
          </option>
        </select>
      </div>
      <div v-if="!selectedTenantId" class="status-text">Seleccioná un tenant para administrar empleados.</div>
      <div v-if="loading" class="centered-loading">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="employees.length === 0" class="status-text">No hay empleados registrados.</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Nombre</th>
            <th>Email</th>
            <th>Rol</th>
            <th>Teléfono</th>
            <th>Activo</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(employee, index) in employees" :key="employee.id">
            <td>{{ index + 1 }}</td>
            <td>{{ employee.first_name }} {{ employee.last_name }}</td>
            <td>{{ employee.email }}</td>
            <td>{{ employee.role }}</td>
            <td>{{ employee.phone || '-' }}</td>
            <td>{{ employee.is_active ? 'Sí' : 'No' }}</td>
            <td class="table-actions">
              <button class="btn-secondary" @click="openEditModal(employee)">Editar</button>
              <button :class="[employee.is_active ? 'btn-danger' : 'btn-success']" @click="toggleActive(employee)">
                {{ employee.is_active ? 'Desactivar' : 'Reactivar' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="isCreateModalOpen" class="modal-overlay" @click.self="closeCreateModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeCreateModal">×</button>
        <h3>Nuevo empleado</h3>
        <p>Registra un empleado con el rol adecuado para el tenant.</p>

        <form @submit.prevent="submitCreate">
          <div class="input-grid">
            <div class="input-group">
              <label>Nombre</label>
              <input type="text" v-model="createForm.first_name" required />
            </div>
            <div class="input-group">
              <label>Apellido</label>
              <input type="text" v-model="createForm.last_name" required />
            </div>
          </div>

          <div class="input-grid">
            <div class="input-group">
              <label>Email</label>
              <input type="email" v-model="createForm.email" required />
            </div>
            <div class="input-group">
              <label>Teléfono</label>
              <input type="text" v-model="createForm.phone" />
            </div>
          </div>

          <div class="input-grid">
            <div class="input-group">
              <label>Rol</label>
              <select v-model="createForm.role" required>
                <option value="employee">employee</option>
                <option value="cashier">cashier</option>
                <option value="kitchen">kitchen</option>
                <option value="admin">admin</option>
              </select>
            </div>
            <div class="input-group">
              <label>Contraseña</label>
              <input type="password" v-model="createForm.password" required />
            </div>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeCreateModal" :disabled="actionLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="actionLoading">
              <span v-if="actionLoading" class="spinner"></span>
              {{ actionLoading ? 'Guardando...' : 'Crear empleado' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="isEditModalOpen" class="modal-overlay" @click.self="closeEditModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeEditModal">×</button>
        <h3>Editar empleado</h3>
        <p>Actualiza los datos del empleado seleccionado.</p>

        <form @submit.prevent="submitEdit">
          <div class="input-grid">
            <div class="input-group">
              <label>Nombre</label>
              <input type="text" v-model="editForm.first_name" required />
            </div>
            <div class="input-group">
              <label>Apellido</label>
              <input type="text" v-model="editForm.last_name" required />
            </div>
          </div>

          <div class="input-grid">
            <div class="input-group">
              <label>Email</label>
              <input type="email" v-model="editForm.email" required />
            </div>
            <div class="input-group">
              <label>Teléfono</label>
              <input type="text" v-model="editForm.phone" />
            </div>
          </div>

          <div class="input-grid">
            <div class="input-group">
              <label>Rol</label>
              <select v-model="editForm.role" required>
                <option value="employee">employee</option>
                <option value="cashier">cashier</option>
                <option value="kitchen">kitchen</option>
                <option value="admin">admin</option>
              </select>
            </div>
            <div class="input-group">
              <label>Contraseña (dejar vacío para no cambiar)</label>
              <input type="password" v-model="editForm.password" />
            </div>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeEditModal" :disabled="actionLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="actionLoading">
              <span v-if="actionLoading" class="spinner"></span>
              {{ actionLoading ? 'Guardando...' : 'Guardar cambios' }}
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
import { getJSON, postJSON, putJSON, deleteJSON, apiFetch } from '../../services/api.js';
import { useRoute } from 'vue-router';
import { useAuth } from '../../composables/useAuth.js';

const route = useRoute();
const tenantId = route.params.tenantId;

const employees = ref([]);
const loading = ref(true);
const error = ref('');
const isCreateModalOpen = ref(false);
const isEditModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const actionLoading = ref(false);
const createForm = ref({
  first_name: '',
  last_name: '',
  email: '',
  phone: '',
  role: 'employee',
  password: ''
});
const editForm = ref({
  id: '',
  first_name: '',
  last_name: '',
  email: '',
  phone: '',
  role: 'employee',
  password: ''
});

async function loadEmployees() {
  loading.value = true;
  error.value = '';
  try {
    const headers = {
      'Content-Type': 'application/json',
      ...(localStorage.getItem('token') ? { Authorization: `Bearer ${localStorage.getItem('token')}` } : {}),
      ...(selectedTenantId ? { 'X-Tenant-ID': selectedTenantId } : {})
    };
    const res = await apiFetch(`/employees`, { method: 'GET', headers });
    if (!res.ok) {
      const payload = await res.json().catch(() => ({}));
      throw new Error(payload.message || 'Error al cargar empleados.');
    }
    employees.value = await res.json();
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los empleados.';
  } finally {
    loading.value = false;
  }
}

async function loadTenants() {
  const auth = useAuth();
  const b = auth.user.value?.brand_id || auth.user.value?.brandID || '';
  try {
    const url = b ? `/tenants?brand_id=${b}` : '/tenants';
    tenants.value = await getJSON(url);
  } catch (err) {
    console.error('No se pudieron cargar tenants:', err);
  }
}

function openCreateModal() {
  isCreateModalOpen.value = true;
  createForm.value = {
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    role: 'employee',
    password: ''
  };
}

function closeCreateModal() {
  isCreateModalOpen.value = false;
}

function openEditModal(employee) {
  editForm.value = {
    id: employee.id,
    first_name: employee.first_name,
    last_name: employee.last_name,
    email: employee.email,
    phone: employee.phone || '',
    role: employee.role,
    password: ''
  };
  isEditModalOpen.value = true;
}

function closeEditModal() {
  isEditModalOpen.value = false;
}

function closeAlert() {
  isAlertOpen.value = false;
  alertText.value = '';
}

async function submitCreate() {
  if (!selectedTenantId.value) {
    alertText.value = 'Seleccioná un tenant antes de crear empleados.';
    isAlertOpen.value = true;
    return;
  }
  actionLoading.value = true;
  try {
    await fetch('/api/employees', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': selectedTenantId.value,
        ...(localStorage.getItem('token') ? { Authorization: `Bearer ${localStorage.getItem('token')}` } : {})
      },
      body: JSON.stringify(createForm.value)
    }).then(async (res) => {
      if (!res.ok) {
        const payload = await res.json().catch(() => ({}));
        throw new Error(payload.message || 'Error al crear empleado.');
      }
      return res.json();
    });
    await loadEmployees();
    closeCreateModal();
  } catch (err) {
    alertText.value = err.message || 'Error al crear empleado.';
    isAlertOpen.value = true;
  } finally {
    actionLoading.value = false;
  }
}

async function submitEdit() {
  if (!selectedTenantId.value) {
    alertText.value = 'Seleccioná un tenant antes de editar empleados.';
    isAlertOpen.value = true;
    return;
  }
  actionLoading.value = true;
  try {
    await fetch(`/api/employees/${editForm.value.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': selectedTenantId.value,
        ...(localStorage.getItem('token') ? { Authorization: `Bearer ${localStorage.getItem('token')}` } : {})
      },
      body: JSON.stringify({
        first_name: editForm.value.first_name,
        last_name: editForm.value.last_name,
        email: editForm.value.email,
        phone: editForm.value.phone,
        role: editForm.value.role,
        password: editForm.value.password
      })
    }).then(async (res) => {
      if (!res.ok) {
        const payload = await res.json().catch(() => ({}));
        throw new Error(payload.message || 'Error al editar empleado.');
      }
      return res.json();
    });
    await loadEmployees();
    closeEditModal();
  } catch (err) {
    alertText.value = err.message || 'Error al editar empleado.';
    isAlertOpen.value = true;
  } finally {
    actionLoading.value = false;
  }
}

async function toggleActive(employee) {
  if (!selectedTenantId.value) {
    alertText.value = 'Seleccioná un tenant antes de actualizar empleados.';
    isAlertOpen.value = true;
    return;
  }
  actionLoading.value = true;
  try {
    await fetch(`/api/employees/${employee.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': selectedTenantId.value,
        ...(localStorage.getItem('token') ? { Authorization: `Bearer ${localStorage.getItem('token')}` } : {})
      },
      body: JSON.stringify({
        first_name: employee.first_name,
        last_name: employee.last_name,
        email: employee.email,
        phone: employee.phone,
        role: employee.role,
        is_active: !employee.is_active
      })
    }).then(async (res) => {
      if (!res.ok) {
        const payload = await res.json().catch(() => ({}));
        throw new Error(payload.message || 'Error al actualizar estado.');
      }
      return res.json();
    });
    await loadEmployees();
  } catch (err) {
    alertText.value = err.message || 'Error al actualizar estado.';
    isAlertOpen.value = true;
  } finally {
    actionLoading.value = false;
  }
}

onMounted(async () => {
  await loadTenants();
});
</script>

<style scoped>
.input-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}
@media (max-width: 800px) {
  .input-grid {
    grid-template-columns: 1fr;
  }
}
.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}
</style>
