<template>
  <section class="dashboard-welcome">
    <div class="card">
      <div class="dashboard-header">
        <div>
          <h1>Hola Admin!</h1>
          <p>Bienvenido al panel de administración.</p>
        </div>
        <button class="btn-logout" @click="logout">Cerrar sesión</button>
      </div>
    </div>
  </section>
</template>

<script setup>
import { useRouter } from 'vue-router';
import { useAuth } from '../../composables/useAuth.js';

const router = useRouter();
const auth = useAuth();

function logout() {
  auth.logout();
  router.push('/login');
}
</script>
