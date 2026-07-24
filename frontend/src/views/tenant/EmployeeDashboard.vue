<template>
  <section class="dashboard-welcome">
    <div class="card">
      <h1>Panel del Empleado</h1>
      <p>Acceso restringido para tareas de caja, inventario y soporte.</p>
      <p>Tenant: <strong>{{ tenantId }}</strong></p>
      <div class="role-notice">Tu experiencia es distinta a la del administrador.</div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();
const tenantId = computed(() => route.params.tenantId);
</script>
