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

        <button type="submit" class="btn-submit" :disabled="loading">
          <span v-if="loading">Ingresando...</span>
          <span v-else>Iniciar sesión</span>
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

const auth = useAuth();
const router = useRouter();
const email = ref('');
const password = ref('');
const loading = ref(false);
const errorMessage = ref('');

async function handleLogin() {
  errorMessage.value = '';
  loading.value = true;

  try {
    await auth.login({ email: email.value, password: password.value });
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
    router.push('/admin');
    return;
  }
  if (role === 'employee') {
    const tenantId = auth.tenantId.value;
    if (tenantId) {
      router.push(`/tenant/${tenantId}/employee`);
      return;
    }
    router.push('/unauthorized');
    return;
  }
  router.push('/unauthorized');
}

</script>
