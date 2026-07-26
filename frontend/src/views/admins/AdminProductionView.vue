<template>
  <section>
    <div class="section-header">
      <div class="section-title">
        <h2>Gestión de producción</h2>
        <p>Registra y revisa los reportes de producción diaria.</p>
      </div>
      <button class="btn-primary" @click="openCreateModal">Nuevo registro</button>
    </div>

    <div class="card">
      <div class="filters-grid">
        <div class="input-group">
          <label>Desde</label>
          <input type="date" v-model="filters.start_date" />
        </div>
        <div class="input-group">
          <label>Hasta</label>
          <input type="date" v-model="filters.end_date" />
        </div>
        <div class="filter-actions">
          <button class="btn-secondary" @click="resetFilters">Limpiar filtros</button>
          <button class="btn-primary" @click="loadReports">Aplicar filtros</button>
        </div>
      </div>

      <div v-if="loading" class="centered-loading">
        <span class="spinner-large"></span>
      </div>
      <div v-else-if="error" class="status-text error">{{ error }}</div>
      <div v-else-if="reports.length === 0" class="status-text">No se encontraron registros de producción.</div>

      <table v-else class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Fecha</th>
            <th>Medallones</th>
            <th>Panes</th>
            <th>Creado por</th>
            <th>Notas</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(report, index) in reports" :key="report.id">
            <td>{{ index + 1 }}</td>
            <td>{{ report.production_date }}</td>
            <td>{{ report.medallions_produced }}</td>
            <td>{{ report.breads_purchased }}</td>
            <td>{{ report.created_by || '-' }}</td>
            <td>{{ report.notes || '-' }}</td>
            <td class="table-actions">
              <button class="btn-secondary" @click="openEditModal(report)">Editar</button>
              <button class="btn-danger" @click="confirmDelete(report)">Eliminar</button>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="production-summary">
        <div class="summary-card">
          <h3>Total medallones</h3>
          <p>{{ totalMedallions }}</p>
        </div>
        <div class="summary-card">
          <h3>Total panes</h3>
          <p>{{ totalBreads }}</p>
        </div>
      </div>
    </div>

    <div class="card chart-card">
      <h3>Últimos 7 días</h3>
      <div class="chart-grid">
        <div class="chart-block">
          <p>Medallones</p>
          <div class="chart-bar" v-for="point in chartData.medallions" :key="point.date">
            <div class="bar" :style="`height: ${point.value / maxChartValue * 100}%`"></div>
            <span>{{ point.date.slice(5) }}</span>
          </div>
        </div>
        <div class="chart-block">
          <p>Panes</p>
          <div class="chart-bar" v-for="point in chartData.breads" :key="point.date">
            <div class="bar" :style="`height: ${point.value / maxChartValue * 100}%`"></div>
            <span>{{ point.date.slice(5) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="isCreateModalOpen || isEditModalOpen" class="modal-overlay" @click.self="closeModal">
      <div class="modal-card">
        <button type="button" class="modal-close" @click="closeModal">×</button>
        <h3>{{ isEditModalOpen ? 'Editar registro' : 'Nuevo registro' }}</h3>
        <p>{{ isEditModalOpen ? 'Actualiza el registro de producción.' : 'Crea un nuevo registro de producción para el tenant.' }}</p>

        <form @submit.prevent="isEditModalOpen ? submitEdit() : submitCreate()">
          <div class="input-grid">
            <div class="input-group">
              <label>Fecha</label>
              <input type="date" v-model="form.production_date" required />
            </div>
            <div class="input-group">
              <label>Medallones producidos</label>
              <input type="number" min="0" v-model.number="form.medallions_produced" required />
            </div>
            <div class="input-group">
              <label>Panes comprados</label>
              <input type="number" min="0" v-model.number="form.breads_purchased" required />
            </div>
          </div>
          <div class="input-group">
            <label>Notas</label>
            <textarea rows="4" v-model="form.notes"></textarea>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeModal" :disabled="actionLoading">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="actionLoading">
              <span v-if="actionLoading" class="spinner"></span>
              {{ actionLoading ? 'Guardando...' : (isEditModalOpen ? 'Guardar cambios' : 'Crear registro') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="isConfirmDeleteOpen" class="alert-overlay" @click.self="closeDeleteConfirm">
      <div class="alert-card">
        <h3 class="alert-title">Confirmar eliminación</h3>
        <p class="alert-message">¿Eliminar el registro del {{ deleteTarget.production_date }}?</p>
        <div class="modal-actions">
          <button type="button" class="btn-secondary" @click="closeDeleteConfirm">Cancelar</button>
          <button type="button" class="btn-danger" @click="deleteReport" :disabled="actionLoading">
            <span v-if="actionLoading" class="spinner"></span>
            {{ actionLoading ? 'Eliminando...' : 'Eliminar' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { getJSON, postJSON, putJSON, deleteJSON } from '../../services/api.js';
import { useRoute } from 'vue-router';

const route = useRoute();
const tenantId = route.params.tenantId;

const reports = ref([]);
const loading = ref(true);
const error = ref('');
const filters = ref({ start_date: '', end_date: '' });
const isCreateModalOpen = ref(false);
const isEditModalOpen = ref(false);
const isConfirmDeleteOpen = ref(false);
const deleteTarget = ref(null);
const actionLoading = ref(false);
const form = ref({
  id: '',
  production_date: '',
  medallions_produced: 0,
  breads_purchased: 0,
  notes: ''
});

const totalMedallions = computed(() => reports.value.reduce((sum, item) => sum + (item.medallions_produced || 0), 0));
const totalBreads = computed(() => reports.value.reduce((sum, item) => sum + (item.breads_purchased || 0), 0));

const chartData = computed(() => {
  const sorted = [...reports.value].sort((a, b) => a.production_date.localeCompare(b.production_date));
  return {
    medallions: sorted.map(r => ({ date: r.production_date, value: r.medallions_produced || 0 })),
    breads: sorted.map(r => ({ date: r.production_date, value: r.breads_purchased || 0 }))
  };
});

const maxChartValue = computed(() => {
  const values = [...chartData.value.medallions, ...chartData.value.breads].map(item => item.value);
  const max = Math.max(...values, 1);
  return max;
});

async function loadReports() {
  loading.value = true;
  error.value = '';
  try {
    const query = new URLSearchParams();
    if (filters.value.start_date) query.set('start_date', filters.value.start_date);
    if (filters.value.end_date) query.set('end_date', filters.value.end_date);
    reports.value = await getJSON(`/production?${query.toString()}`);
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los reportes de producción.';
  } finally {
    loading.value = false;
  }
}

function resetFilters() {
  filters.value.start_date = '';
  filters.value.end_date = '';
  loadReports();
}

function openCreateModal() {
  form.value = { id: '', production_date: '', medallions_produced: 0, breads_purchased: 0, notes: '' };
  isCreateModalOpen.value = true;
}

function openEditModal(report) {
  form.value = { ...report };
  isEditModalOpen.value = true;
}

function closeModal() {
  isCreateModalOpen.value = false;
  isEditModalOpen.value = false;
}

function confirmDelete(report) {
  deleteTarget.value = report;
  isConfirmDeleteOpen.value = true;
}

function closeDeleteConfirm() {
  isConfirmDeleteOpen.value = false;
  deleteTarget.value = null;
}

async function submitCreate() {
  actionLoading.value = true;
  try {
    await postJSON('/production', { ...form.value, tenant_id: tenantId });
    await loadReports();
    closeModal();
  } catch (err) {
    error.value = err.message || 'Error al crear registro de producción.';
  } finally {
    actionLoading.value = false;
  }
}

async function submitEdit() {
  actionLoading.value = true;
  try {
    await putJSON(`/production/${form.value.id}`, {
      production_date: form.value.production_date,
      medallions_produced: form.value.medallions_produced,
      breads_purchased: form.value.breads_purchased,
      notes: form.value.notes
    });
    await loadReports();
    closeModal();
  } catch (err) {
    error.value = err.message || 'Error al actualizar registro de producción.';
  } finally {
    actionLoading.value = false;
  }
}

async function deleteReport() {
  actionLoading.value = true;
  try {
    await deleteJSON(`/production/${deleteTarget.value.id}`);
    await loadReports();
    closeDeleteConfirm();
  } catch (err) {
    error.value = err.message || 'Error al eliminar registro de producción.';
  } finally {
    actionLoading.value = false;
  }
}

onMounted(loadReports);
</script>

<style scoped>
.filters-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
  margin-bottom: 24px;
}
.filter-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}
.production-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin-top: 24px;
}
.summary-card {
  padding: 24px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
}
.summary-card h3 {
  margin: 0 0 12px;
  color: var(--text-muted);
}
.summary-card p {
  margin: 0;
  font-size: 2.4rem;
  font-weight: 700;
}
.chart-card {
  padding: 24px;
}
.chart-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
  margin-top: 18px;
}
.chart-block {
  padding: 18px;
  border-radius: 18px;
  background: rgba(15, 23, 42, 0.9);
}
.chart-bar {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 10px;
  align-items: end;
  min-height: 220px;
}
.chart-bar .bar {
  width: 100%;
  background: linear-gradient(180deg, rgba(96, 165, 250, 0.5), rgba(37, 99, 235, 0.9));
  border-radius: 12px 12px 0 0;
}
.chart-bar span {
  margin-top: 8px;
  font-size: 0.75rem;
  color: var(--text-muted);
  text-align: center;
}
@media (max-width: 900px) {
  .filters-grid,
  .chart-grid,
  .production-summary {
    grid-template-columns: 1fr;
  }
}
</style>
