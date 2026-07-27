<template>
  <section class="dashboard-welcome">
    <div class="card">
      <div class="dashboard-header">
        <div>
          <h1>Hola Empleado!</h1>
          <p>Bienvenido a tu panel.</p>
          <p>Tenant: <strong>{{ tenantId }}</strong></p>
        </div>
        <button class="btn-logout" @click="logout">Cerrar sesión</button>
      </div>
      <div class="role-notice">Tu experiencia es distinta a la del administrador.</div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '../../composables/useAuth.js';

const route = useRoute();
const router = useRouter();
const auth = useAuth();
const tenantId = computed(() => route.params.tenantId);

function logout() {
  auth.logout();
  router.push('/login');
}
</script>
