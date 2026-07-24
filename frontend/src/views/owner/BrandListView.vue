<template>
  <section>
    <div class="section-title">
      <h2>Brands</h2>
      <p>Gestiona las marcas asociadas a los tenants.</p>
    </div>

    <div class="card">
      <div v-if="loading" class="status-text">Cargando brands...</div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="brands.length === 0" class="status-text">No hay brands registrados.</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Nombre</th>
            <th>Tax ID</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="brand in brands" :key="brand.id">
            <td>{{ brand.name }}</td>
            <td>{{ brand.tax_id }}</td>
            <td class="table-actions">
              <button class="btn-secondary" @click="editBrand(brand)">Editar</button>
              <button class="btn-danger" @click="deleteBrand(brand.id)">Eliminar</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { getJSON, deleteJSON } from '../../services/api.js';

const brands = ref([]);
const loading = ref(true);
const error = ref('');

async function loadBrands() {
  try {
    brands.value = await getJSON('/brands');
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar las brands.';
  } finally {
    loading.value = false;
  }
}

function editBrand(brand) {
  alert(`Editar brand ${brand.name} aún no está implementado.`);
}

async function deleteBrand(id) {
  const confirmed = confirm('¿Seguro que querés eliminar esta brand?');
  if (!confirmed) return;
  try {
    await deleteJSON(`/brands/${id}`);
    await loadBrands();
  } catch (err) {
    error.value = err.message || 'No se pudo eliminar la brand.';
  }
}

onMounted(loadBrands);
</script>
