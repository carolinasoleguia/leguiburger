<script setup>
import { onMounted, ref } from "vue";
import DashboardShell from "../../components/DashboardShell.vue";
import { useMembership } from "../../composables/useMembership.js";
import { useAuth } from "../../composables/useAuth.js";
const membership = useMembership(),
  auth = useAuth(),
  error = ref(""),
  loading = ref(true);
async function refresh() {
  loading.value = true;
  error.value = "";
  try {
    await membership.refresh();
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
onMounted(refresh);
</script>
<template>
  <DashboardShell role="employee"
    ><section class="card operation-form membership-page" :aria-busy="loading">
      <h1>
        {{
          membership.access.value?.operational
            ? "Acceso del local"
            : "El local necesita renovar su membresía"
        }}
      </h1>
      <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
      <p v-else-if="loading" role="status">Consultando el acceso…</p>
      <p v-else>
        Contactá al administrador de tu restaurante para renovar la membresía o
        liberar una terminal. Podrás volver a gestionar caja y pedidos cuando el
        acceso esté habilitado.
      </p>
      <router-link
        v-if="membership.access.value?.operational"
        :to="auth.userRole.value === 'cashier' ? '/orders' : '/kitchen'"
        class="btn-primary"
        >Volver al local</router-link
      >
      <button
        class="btn-secondary"
        type="button"
        :disabled="loading"
        @click="refresh"
      >
        Volver a consultar
      </button>
    </section></DashboardShell
  >
</template>
<style scoped>
.membership-page {
  max-width: 42rem;
  margin: 0 auto;
}
.membership-page h1 {
  font-size: clamp(1.5rem, 4vw, 2rem);
}
</style>
