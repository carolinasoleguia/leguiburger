<template>
  <section class="admin-production">
    <PageHeading title="Gestión de producción" description="Seleccioná un local para ver y cargar reportes de producción.">
    </PageHeading>

    <div class="card">
      <div class="controls-grid">
        <div class="input-group">
          <label for="production-location">Local</label>
          <select id="production-location" v-model="selectedTenant">
            <option value="">-- Seleccionar local --</option>
            <option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
              {{ tenant.subdomain || tenant.id }}
            </option>
          </select>
        </div>
        <div class="control-actions">
          <button class="btn-primary" @click="openCreateModal" :disabled="!selectedTenant || loading">
            Nuevo reporte
          </button>
        </div>
      </div>

      <StatePanel v-if="loading" kind="loading" />
      <div v-else-if="error" class="status-text error" role="alert">{{ error }}</div>
      <div v-else-if="!selectedTenant" class="status-text">Seleccioná un local para ver los reportes de producción.</div>
      <div v-else-if="reports.length === 0" class="status-text">No se encontraron reportes de producción para el local seleccionado.</div>

      <div v-else class="table-scroll"><table class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Fecha</th>
            <th>Medallones producidos</th>
            <th>Panes comprados</th>
            <th>Notas</th>
            <th>Creado</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(report, index) in reports" :key="report.id">
            <td>{{ index + 1 }}</td>
            <td>{{ report.production_date }}</td>
            <td>{{ report.medallions_produced }}</td>
            <td>{{ report.breads_purchased }}</td>
            <td>{{ report.notes || '-' }}</td>
            <td>{{ report.created_at }}</td>
          </tr>
        </tbody>
      </table></div>
    </div>

    <Transition name="modal"><div v-if="isCreateModalOpen" class="modal-overlay" @click.self="closeCreateModal">
      <div v-dialog-care class="modal-card">
        <button type="button" class="modal-close" @click="closeCreateModal">×</button>
        <h3>Nuevo reporte de producción</h3>
        <p>Agregá un reporte de producción para el local seleccionado.</p>

        <form v-form-care @submit.prevent="submitCreateReport">
          <div class="input-grid">
            <div class="input-group">
              <label>Fecha de producción</label>
              <input type="date" v-model="createForm.production_date" required />
            </div>
            <div class="input-group">
              <label>Medallones producidos</label>
              <input type="number" min="0" v-model.number="createForm.medallions_produced" required />
            </div>
          </div>

          <div class="input-grid">
            <div class="input-group">
              <label>Panes comprados</label>
              <input type="number" min="0" v-model.number="createForm.breads_purchased" required />
            </div>
            <div class="input-group">
              <label>Notas</label>
              <textarea v-model="createForm.notes"></textarea>
            </div>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeCreateModal" :disabled="actionLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="actionLoading || !selectedTenant">
              <span v-if="actionLoading" class="spinner"></span>
              {{ actionLoading ? 'Guardando...' : 'Agregar reporte' }}
            </button>
          </div>
        </form>
      </div>
    </div></Transition>
  </section>
</template>

<script setup>
import PageHeading from "../../components/PageHeading.vue";
import StatePanel from "../../components/StatePanel.vue";
import { ref, onMounted, watch } from 'vue';
import { apiFetch, getJSON } from '../../services/api.js';
import { useAuth } from '../../composables/useAuth.js';

const auth = useAuth();
const tenants = ref([]);
const selectedTenant = ref('');
const reports = ref([]);
const loading = ref(true);
const error = ref('');
const isCreateModalOpen = ref(false);
const actionLoading = ref(false);
const createForm = ref({
  production_date: '',
  medallions_produced: 0,
  breads_purchased: 0,
  notes: ''
});

watch(selectedTenant, async (newTenant) => {
  if (newTenant) {
    await loadReports();
  } else {
    reports.value = [];
  }
});

async function loadTenants() {
  try {
    const brandId = auth.user.value?.brand_id || auth.user.value?.brandID || '';
    const url = brandId ? `/tenants?brand_id=${brandId}` : '/tenants';
    tenants.value = await getJSON(url);
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los locales.';
  } finally {
    if (!tenants.value.length) loading.value = false;
  }
}

async function loadReports() {
  if (!selectedTenant.value) {
    reports.value = [];
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    const response = await apiFetch('/production', {
      method: 'GET',
      headers: {
        'X-Tenant-ID': selectedTenant.value
      }
    });
    if (!response.ok) {
      const text = await response.text().catch(() => 'Error cargando reportes de producción');
      throw new Error(text || 'Error cargando reportes de producción');
    }
    reports.value = await response.json();
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los reportes de producción.';
    reports.value = [];
  } finally {
    loading.value = false;
  }
}

function openCreateModal() {
  if (!selectedTenant.value) {
    error.value = 'Seleccioná un tenant antes de agregar un reporte de producción.';
    return;
  }
  error.value = '';
  createForm.value = {
    production_date: '',
    medallions_produced: 0,
    breads_purchased: 0,
    notes: ''
  };
  isCreateModalOpen.value = true;
}

function closeCreateModal() {
  isCreateModalOpen.value = false;
}

async function submitCreateReport() {
  if (!selectedTenant.value) {
    error.value = 'Seleccioná un tenant antes de agregar un reporte de producción.';
    return;
  }

  actionLoading.value = true;
  error.value = '';
  try {
    const response = await apiFetch('/production', {
      method: 'POST',
      headers: {
        'X-Tenant-ID': selectedTenant.value
      },
      body: JSON.stringify(createForm.value)
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}));
      throw new Error(payload.message || 'Error al crear el reporte de producción');
    }
    await loadReports();
    closeCreateModal();
  } catch (err) {
    error.value = err.message || 'Error al crear el reporte de producción.';
  } finally {
    actionLoading.value = false;
  }
}

onMounted(async () => {
  await loadTenants();
  if (tenants.value.length > 0) {
    selectedTenant.value = tenants.value[0].id;
  }
});
</script>

<style scoped>
.admin-production {width:100%;min-width:0;}
.controls-grid {display:grid;grid-template-columns:minmax(0,1fr) auto;gap:20px;align-items:end;margin-bottom:24px;}
.control-actions {display:flex;justify-content:flex-end;}
.input-group {display:flex;flex-direction:column;gap:8px;}
@media(max-width:650px){.controls-grid{grid-template-columns:1fr}.control-actions{justify-content:flex-start}}
</style>
