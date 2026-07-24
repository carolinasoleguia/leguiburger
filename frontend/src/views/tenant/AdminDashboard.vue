<template>
  <section class="dashboard-welcome">
    <div class="card">
      <h1>Panel del Admin</h1>
      <p>Administra productos, ventas y usuarios del tenant <strong>{{ tenantId }}</strong>.</p>
      <div class="quick-actions">
        <router-link :to="`/tenant/${tenantId}/admin`" class="btn-secondary">Vista general</router-link>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();
const tenantId = computed(() => route.params.tenantId);
</script>
