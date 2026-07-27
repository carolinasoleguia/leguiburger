<template>
  <section class="admin-landing">
    <div class="card">
      <div class="dashboard-header">
        <div>
          <h1>Bienvenido Admin</h1>
          <p>Seleccioná el comercio (tenant) que querés administrar hoy.</p>
        </div>
        <button class="btn-logout" @click="logout">Cerrar sesión</button>
      </div>

      <div class="admin-landing-content">
        <div v-if="loading" class="centered-loading">
          <span class="spinner-large"></span>
        </div>

        <div v-else-if="error" class="status-text error">{{ error }}</div>

        <div v-else>
          <div v-if="filteredTenants.length === 0" class="status-text">
            No se encontraron tenants disponibles para tu marca.
          </div>

          <div class="tenant-grid" v-else>
            <div
              v-for="tenant in filteredTenants"
              :key="tenant.id"
              class="tenant-card"
            >
              <div>
                <h3>{{ tenant.subdomain }}</h3>
                <p>Marca: {{ tenant.brand.name || tenant.brand_id || 'Sin marca' }}</p>
              </div>
              <button class="btn-primary" @click="selectTenant(tenant.id)">Ingresar</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useAuth } from '../../composables/useAuth.js';
import { getJSON } from '../../services/api.js';

const router = useRouter();
const auth = useAuth();
const loading = ref(true);
const error = ref('');
const tenants = ref([]);

const brandId = computed(() => auth.user.value?.brand_id || auth.user.value?.brandID || '');

const filteredTenants = computed(() => {
  if (!brandId.value) {
    return tenants.value;
  }
  return tenants.value.filter((tenant) => tenant.brand?.id === brandId.value || tenant.brand_id === brandId.value);
});

async function loadTenants() {
  loading.value = true;
  error.value = '';
  try {
    const b = brandId.value;
    const url = b ? `/tenants?brand_id=${b}` : '/tenants';
    tenants.value = await getJSON(url);
  } catch (err) {
    error.value = err.message || 'No se pudieron cargar los tenants.';
  } finally {
    loading.value = false;
  }
}

function logout() {
  auth.logout();
  router.push('/login');
}

function selectTenant(tenantId) {
  router.push(`/tenant/${tenantId}/admin`);
}

onMounted(loadTenants);
</script>

<style scoped>
.admin-landing {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.admin-landing-content {
  margin-top: 28px;
}

.tenant-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin-top: 24px;
}

.tenant-card {
  padding: 24px;
  border-radius: 18px;
  background: rgba(15, 23, 42, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-height: 160px;
}

.tenant-card h3 {
  margin: 0 0 10px;
}

.tenant-card p {
  margin: 0;
  color: var(--text-muted);
}

@media (max-width: 820px) {
  .tenant-grid {
    grid-template-columns: 1fr;
  }
}
</style>
