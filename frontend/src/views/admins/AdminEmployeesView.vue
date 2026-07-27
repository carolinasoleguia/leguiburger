<template>
  <section>
    <div class="section-header">
      <div class="section-title">
        <h2>Gestión de empleados</h2>
        <p>Administra los diferentes roles de empleado y sus datos de acceso.</p>
      </div>
      <button class="btn-primary" @click="openCreateModal">Nuevo empleado</button>
    </div>

    <div class="card">
      <div class="filter-row">
        <div class="input-group">
          <label>Filtrar por tenant</label>
          <select v-model="selectedTenantFilter">
            <option value="">Todos los tenants</option>
            <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
              {{ tenant.subdomain || tenant.brand?.name || tenant.id }}
            </option>
          </select>
        </div>
        <button class="btn-secondary" @click="clearFilter" v-if="selectedTenantFilter">Limpiar filtro</button>
      </div>

      <div v-if="loading" class="centered-loading">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="filteredEmployees.length === 0" class="status-text">No hay empleados registrados para el filtro seleccionado.</div>

      <table v-else class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Nombre</th>
            <th>Email</th>
            <th>Rol</th>
            <th>Teléfono</th>
            <th>Tenant</th>
            <th>Activo</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(employee, index) in filteredEmployees" :key="employee.id">
            <td>{{ index + 1 }}</td>
            <td>{{ employee.first_name }} {{ employee.last_name }}</td>
            <td>{{ employee.email }}</td>
            <td>{{ employee.role }}</td>
            <td>{{ employee.phone || '-' }}</td>
            <td>{{ employee.tenant?.subdomain || employee.tenant?.brand?.name || employee.tenant_id || '-' }}</td>
            <td>{{ employee.is_active ? 'Sí' : 'No' }}</td>
            <td class="table-actions">
              <button class="btn-secondary" @click="openEditModal(employee)">Editar</button>
              <button :class="[employee.is_active ? 'btn-danger' : 'btn-success']" @click="confirmToggleActive(employee)">
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
              <label>Tenant</label>
              <select v-model="createForm.tenant_id" required>
                <option value="" disabled>Seleccioná tenant</option>
                <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
                  {{ tenant.brand?.name || tenant.subdomain || tenant.id }}
                </option>
              </select>
            </div>
            <div class="input-group">
              <label>Rol</label>
              <select v-model="createForm.role" required>
                <option value="employee">employee</option>
                <option value="cashier">cashier</option>
                <option value="kitchen">kitchen</option>
                <option value="admin">admin</option>
              </select>
            </div>
          </div>

          <div class="input-grid">
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
              <label>Tenant</label>
              <select v-model="editForm.tenant_id" required>
                <option value="" disabled>Seleccioná tenant</option>
                <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
                  {{ tenant.brand?.name || tenant.subdomain || tenant.id }}
                </option>
              </select>
            </div>
            <div class="input-group">
              <label>Rol</label>
              <select v-model="editForm.role" required>
                <option value="employee">employee</option>
                <option value="cashier">cashier</option>
                <option value="kitchen">kitchen</option>
                <option value="admin">admin</option>
              </select>
            </div>
          </div>

          <div class="input-grid">
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

    <div v-if="isConfirmModalOpen" class="modal-overlay" @click.self="closeConfirmModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeConfirmModal">×</button>
        <h3>Confirmar acción</h3>
        <p>¿Estás seguro/a que querés {{ confirmAction }} al empleado <strong>{{ editForm.first_name }} {{ editForm.last_name }}</strong>?</p>
        <div class="modal-actions">
          <button type="button" class="btn-secondary" @click="closeConfirmModal">Cancelar</button>
          <button type="button" class="btn-danger" @click="confirmToggle">Sí, {{ confirmAction }}</button>
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
import { computed, ref, onMounted } from 'vue';
import { getJSON, postJSON, putJSON, deleteJSON, apiFetch } from '../../services/api.js';
import { useAuth } from '../../composables/useAuth.js';

const auth = useAuth();
const employees = ref([]);
const tenants = ref([]);
const selectedTenantFilter = ref('');
const loading = ref(true);
const error = ref('');
const isCreateModalOpen = ref(false);
const isEditModalOpen = ref(false);
const isConfirmModalOpen = ref(false);
const isAlertOpen = ref(false);
const alertText = ref('');
const confirmAction = ref('');
const actionLoading = ref(false);
const createForm = ref({
  first_name: '',
  last_name: '',
  email: '',
  phone: '',
  tenant_id: '',
  role: 'employee',
  password: ''
});
const editForm = ref({
  id: '',
  first_name: '',
  last_name: '',
  email: '',
  phone: '',
  tenant_id: '',
  role: 'employee',
  password: ''
});

const filteredEmployees = computed(() => {
  if (!selectedTenantFilter.value) {
    return employees.value;
  }
  return employees.value.filter((employee) => {
    return (
      employee.tenant_id === selectedTenantFilter.value ||
      employee.tenant?.id === selectedTenantFilter.value ||
      employee.tenant?.subdomain === selectedTenantFilter.value
    );
  });
});

function confirmToggleActive(employee) {
  editForm.value = {
    id: employee.id,
    first_name: employee.first_name,
    last_name: employee.last_name,
    email: employee.email,
    phone: employee.phone || '',
    tenant_id: employee.tenant_id || employee.tenant?.id || '',
    role: employee.role,
    password: ''
  };
  confirmAction.value = employee.is_active ? 'desactivar' : 'reactivar';
  isConfirmModalOpen.value = true;
}

async function toggleActive(employee) {
  actionLoading.value = true;
  try {
    const response = await apiFetch(`/employees/${employee.id}`, {
      method: 'PUT',
      body: JSON.stringify({
        first_name: employee.first_name,
        last_name: employee.last_name,
        email: employee.email,
        phone: employee.phone,
        tenant_id: employee.tenant_id || employee.tenant?.id,
        role: employee.role,
        is_active: !employee.is_active
      })
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}));
      throw new Error(payload.message || 'Error al actualizar estado.');
    }
    await loadEmployees();
  } catch (err) {
    alertText.value = err.message || 'Error al actualizar estado.';
    isAlertOpen.value = true;
  } finally {
    actionLoading.value = false;
  }
}

async function confirmToggle() {
  isConfirmModalOpen.value = false;
  const employee = employees.value.find((e) => e.id === editForm.value.id);
  if (!employee) {
    alertText.value = 'Empleado no encontrado.';
    isAlertOpen.value = true;
    return;
  }
  await toggleActive(employee);
}

async function loadEmployees() {
  loading.value = true;
  error.value = '';
  try {
    const res = await apiFetch('/employees', { method: 'GET' });
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
  try {
    const brandId = auth.user.value?.brand_id || auth.user.value?.brandID || '';
    const url = brandId ? `/tenants?brand_id=${brandId}` : '/tenants';
    tenants.value = await getJSON(url);
  } catch (err) {
    console.error('No se pudieron cargar tenants:', err);
  }
}

function clearFilter() {
  selectedTenantFilter.value = '';
}

function openCreateModal() {
  isCreateModalOpen.value = true;
  createForm.value = {
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    tenant_id: tenants.value[0]?.id || '',
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
    tenant_id: employee.tenant_id || employee.tenant?.id || '',
    role: employee.role,
    password: ''
  };
  isEditModalOpen.value = true;
}

function closeEditModal() {
  isEditModalOpen.value = false;
}

function closeConfirmModal() {
  isConfirmModalOpen.value = false;
  confirmAction.value = '';
}

function closeAlert() {
  isAlertOpen.value = false;
  alertText.value = '';
}

async function submitCreate() {
  if (!createForm.value.tenant_id) {
    alertText.value = 'Seleccioná un tenant para el empleado.';
    isAlertOpen.value = true;
    return;
  }
  actionLoading.value = true;
  try {
    const response = await apiFetch('/employees', {
      method: 'POST',
      body: JSON.stringify(createForm.value)
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}));
      throw new Error(payload.message || 'Error al crear empleado.');
    }
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
  if (!editForm.value.tenant_id) {
    alertText.value = 'Seleccioná un tenant para el empleado.';
    isAlertOpen.value = true;
    return;
  }
  actionLoading.value = true;
  try {
    const response = await apiFetch(`/employees/${editForm.value.id}`, {
      method: 'PUT',
      body: JSON.stringify({
        first_name: editForm.value.first_name,
        last_name: editForm.value.last_name,
        email: editForm.value.email,
        phone: editForm.value.phone,
        tenant_id: editForm.value.tenant_id,
        role: editForm.value.role,
        password: editForm.value.password
      })
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}));
      throw new Error(payload.message || 'Error al editar empleado.');
    }
    await loadEmployees();
    closeEditModal();
  } catch (err) {
    alertText.value = err.message || 'Error al editar empleado.';
    isAlertOpen.value = true;
  } finally {
    actionLoading.value = false;
  }
}

onMounted(async () => {
  await loadTenants();
  await loadEmployees();
});
</script>

<style scoped>
.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: flex-end;
  margin-bottom: 1rem;
}
.input-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}
@media (max-width: 800px) {
  .input-grid,
  .filter-row {
    grid-template-columns: 1fr;
    flex-direction: column;
  }
}
.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}
</style>
