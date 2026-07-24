<template>
  <section>
    <div class="section-title">
      <h2>Tenants</h2>
      <p>Gestiona los negocios que contratan el sistema.</p>
    </div>

    <div class="card">
      <div v-if="loading" class="status-text">Cargando tenants...</div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="tenants.length === 0" class="status-text">No hay tenants registrados.</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Nombre</th>
            <th>Subdominio</th>
            <th>Tax ID</th>
            <th>Estado</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="tenant in tenants" :key="tenant.id">
            <td>{{ tenant.name }}</td>
            <td>{{ tenant.subdomain }}</td>
            <td>{{ tenant.tax_id }}</td>
            <td>{{ tenant.active ? 'Activo' : 'Inactivo' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { getJSON } from '../../services/api.js';

const tenants = ref([]);
const loading = ref(true);
const error = ref('');

async function loadTenants() {
  try {
    tenants.value = await getJSON('/tenants');
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los tenants.';
  } finally {
    loading.value = false;
  }
}

onMounted(loadTenants);
</script>
