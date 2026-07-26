<template>
  <section class="admin-production">
    <div class="section-header">
      <div>
        <h2>Gestión de producción</h2>
        <p>Seleccioná un tenant para cargar sus insumos.</p>
      </div>
    </div>

    <div class="card">
      <div class="controls-grid">
        <div class="input-group">
          <label>Tenant</label>
          <select v-model="selectedTenant">
            <option value="">-- Seleccionar tenant --</option>
            <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
              {{ tenant.subdomain || tenant.id }}
            </option>
          </select>
        </div>
        <div class="control-actions">
          <button class="btn-primary" @click="loadSupplies" :disabled="!selectedTenant || loading">
            Cargar insumos
          </button>
        </div>
      </div>

      <div v-if="loading" class="centered-loading">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="!selectedTenant" class="status-text">Seleccioná un tenant para ver sus insumos.</div>
      <div v-else-if="supplies.length === 0" class="status-text">No se encontraron insumos para el tenant seleccionado.</div>

      <table v-else class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Nombre</th>
            <th>Stock</th>
            <th>Precio mayorista</th>
            <th>Unidad</th>
            <th>Activo</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(supply, index) in supplies" :key="supply.id">
            <td>{{ index + 1 }}</td>
            <td>{{ supply.name }}</td>
            <td>{{ supply.current_stock }}</td>
            <td>{{ supply.current_wholesale_cost }}</td>
            <td>{{ supply.measurement_unit }}</td>
            <td>{{ supply.is_active ? 'Sí' : 'No' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { apiFetch, getJSON } from '../../services/api.js';
import { useAuth } from '../../composables/useAuth.js';

const auth = useAuth();
const tenants = ref([]);
const selectedTenant = ref('');
const supplies = ref([]);
const loading = ref(false);
const error = ref('');

async function loadTenants() {
  try {
    const brandId = auth.user.value?.brand_id || auth.user.value?.brandID || '';
    const url = brandId ? `/tenants?brand_id=${brandId}` : '/tenants';
    tenants.value = await getJSON(url);
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los tenants.';
  }
}

async function loadSupplies() {
  if (!selectedTenant.value) {
    supplies.value = [];
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    const response = await apiFetch('/supplies', {
      method: 'GET',
      headers: {
        'X-Tenant-ID': selectedTenant.value
      }
    });
    if (!response.ok) {
      const text = await response.text().catch(() => 'Error cargando insumos');
      throw new Error(text || 'Error cargando insumos');
    }
    supplies.value = await response.json();
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los insumos.';
    supplies.value = [];
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await loadTenants();
  if (tenants.value.length > 0) {
    selectedTenant.value = tenants.value[0].id;
    await loadSupplies();
  }
});
</script>

<style scoped>
.admin-production {
  padding: 1.5rem;
}
.controls-grid {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 1rem;
  margin-bottom: 1.5rem;
  align-items: end;
}
.control-actions {
  display: flex;
  justify-content: flex-end;
}
.input-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th,
.data-table td {
  padding: 0.85rem 1rem;
  border: 1px solid rgba(148, 163, 184, 0.16);
  text-align: left;
}
.centered-loading {
  display: flex;
  justify-content: center;
  padding: 2rem 0;
}
.status-text {
  padding: 1.5rem;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.05);
  margin-top: 1rem;
}
.status-text.error {
  color: #f87171;
}
@media (max-width: 720px) {
  .controls-grid {
    grid-template-columns: 1fr;
  }
}
</style>
