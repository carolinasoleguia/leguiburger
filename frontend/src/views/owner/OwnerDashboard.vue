<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand-logo">🍔 Leguiburger SaaS</div>
      <nav>
        <router-link to="/owner/tenants" class="nav-link" active-class="active">Tenants</router-link>
        <router-link to="/owner/admins" class="nav-link" active-class="active">Admins</router-link>
      </nav>
      <button class="btn-logout" @click="logout">Cerrar sesión</button>
    </aside>

    <main class="content-pane">
      <header class="page-header">
        <h1>Panel de Owner</h1>
        <p>Administra el sistema SaaS, tenants y administradores.</p>
      </header>
      <router-view />
    </main>
  </div>
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
