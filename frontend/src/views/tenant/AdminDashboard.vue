<template>
  <section class="dashboard-welcome">
    <div class="card">
      <div class="dashboard-header">
        <div>
          <h1>Panel del Admin</h1>
          <p>Administra productos, ventas y usuarios del tenant <strong>{{ tenantId }}</strong>.</p>
        </div>
        <button class="btn-logout" @click="logout">Cerrar sesión</button>
      </div>
      <div class="quick-actions">
        <router-link :to="`/tenant/${tenantId}/admin`" class="btn-secondary">Vista general</router-link>
      </div>
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
