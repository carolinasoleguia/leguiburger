<template>
  <section>
    <div class="section-title">
      <h2>Administradores</h2>
      <p>Revisa los administradores de los tenants registrados.</p>
    </div>

    <div class="card">
      <div v-if="loading" class="status-text">Cargando administradores...</div>
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
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { getJSON } from '../../services/api.js';

const admins = ref([]);
const loading = ref(true);
const error = ref('');

async function loadAdmins() {
  try {
    admins.value = await getJSON('/employees');
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los administradores.';
  } finally {
    loading.value = false;
  }
}

onMounted(loadAdmins);
</script>
