<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand-logo">🍔 Leguiburger SaaS</div>
      <nav>
        <router-link to="/admin" class="nav-link" exact-active-class="active">Home</router-link>
        <router-link to="/admin/employees" class="nav-link" active-class="active">Gestión de empleados</router-link>
        <router-link to="/admin/production" class="nav-link" active-class="active">Gestión de producción</router-link>
      </nav>
      <button class="btn-logout" @click="logout">Cerrar sesión</button>
    </aside>

    <main class="content-pane">
      <header class="page-header">
        <h1>Panel de Administrador</h1>
        <p>Controla empleados y producción del tenant <strong>{{ tenantId }}</strong>.</p>
      </header>
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '../../composables/useAuth.js';

const router = useRouter();
const auth = useAuth();

function logout() {
  auth.logout();
  router.push('/login');
}
</script>
