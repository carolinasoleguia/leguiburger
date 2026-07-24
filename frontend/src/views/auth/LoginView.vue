<template>
  <div class="login-wrapper">
    <div class="login-card">
      <div class="brand-header">
        <div class="logo-icon">🍔</div>
        <h2>Leguiburger SaaS</h2>
        <p>Iniciar sesión</p>
      </div>

      <form @submit.prevent="handleLogin">
        <div class="input-group">
          <label for="email">Correo electrónico</label>
          <input id="email" type="email" v-model="email" required placeholder="admin@leguiburger.com" />
        </div>

        <div class="input-group">
          <label for="password">Contraseña</label>
          <input id="password" type="password" v-model="password" required placeholder="••••••••" />
        </div>

        <div v-if="tenantSelectionActive" class="input-group">
          <label for="tenantId">Seleccionar tienda</label>
          <select id="tenantId" v-model="tenantId">
            <option value="">Seleccionar tienda</option>
            <option v-for="tenant in tenantOptions" :key="tenant.tenant_id" :value="tenant.tenant_id">
              {{ tenant.label }}
            </option>
          </select>
        </div>

        <button type="submit" class="btn-submit" :disabled="loading || lookupLoading">
          <span v-if="loading || lookupLoading">{{ tenantSelectionActive ? 'Ingresando...' : 'Verificando...' }}</span>
          <span v-else>{{ tenantSelectionActive ? 'Iniciar sesión' : 'Continuar' }}</span>
        </button>

        <p v-if="errorMessage" class="error-msg">{{ errorMessage }}</p>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuth } from '../../composables/useAuth.js';
import { lookupTenantOptions } from '../../services/authService.js';

const auth = useAuth();
const router = useRouter();
const email = ref('');
const password = ref('');
const tenantId = ref('');
const tenantOptions = ref([]);
const tenantSelectionActive = ref(false);
const loading = ref(false);
const lookupLoading = ref(false);
const errorMessage = ref('');

async function handleLogin() {
  errorMessage.value = '';

  if (!tenantSelectionActive.value) {
    lookupLoading.value = true;
    try {
      const result = await lookupTenantOptions({ email: email.value, password: password.value });
      if (result?.choices?.length > 0) {
        tenantOptions.value = result.choices;
        tenantSelectionActive.value = true;
        return;
      }

      await auth.login({ email: email.value, password: password.value });
      navigateToRole();
    } catch (error) {
      errorMessage.value = error.message || 'Error al verificar credenciales';
    } finally {
      lookupLoading.value = false;
    }
    return;
  }

  if (!tenantId.value) {
    errorMessage.value = 'Selecciona la tienda para continuar';
    return;
  }

  loading.value = true;
  try {
    await auth.login({ email: email.value, password: password.value, tenant_id: tenantId.value });
    navigateToRole();
  } catch (error) {
    errorMessage.value = error.message || 'Error al iniciar sesión';
  } finally {
    loading.value = false;
  }
}

function navigateToRole() {
  const role = auth.userRole.value;
  if (role === 'owner') {
    router.push('/owner');
    return;
  }
  if (role === 'admin') {
    router.push(`/tenant/${auth.tenantId.value}/admin`);
    return;
  }
  if (role === 'employee') {
    router.push(`/tenant/${auth.tenantId.value}/employee`);
    return;
  }
  router.push('/unauthorized');
}

</script>
