<script setup>
import { onMounted, ref } from "vue";
import DashboardShell from "../../components/DashboardShell.vue";
import { useMembership } from "../../composables/useMembership.js";
const membership = useMembership(),
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
  <DashboardShell role="admin"
    ><section class="membership-page card operation-form" :aria-busy="loading">
      <p>Mi negocio</p>
      <h1>
        {{
          membership.access.value?.operational
            ? "Tu membresía"
            : "Renová para seguir operando"
        }}
      </h1>
      <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
      <p v-else-if="loading" role="status">Consultando la vigencia…</p>
      <template v-else>
        <p v-if="membership.access.value?.state === 'trial'">
          Estás en la prueba gratuita de siete días: una tienda, un
          administrador y hasta dos terminales de caja y pedidos.
        </p>
        <p v-else-if="!membership.access.value?.operational">
          Tu período de acceso terminó. Elegí una membresía para habilitar
          nuevamente la administración y las terminales de tus locales.
        </p>
        <p v-if="membership.access.value?.valid_until">
          Acceso hasta
          {{
            new Date(membership.access.value.valid_until).toLocaleString(
              "es-AR",
            )
          }}.
        </p>
        <p>
          Clásico incluye una tienda y dos terminales. Premium incluye cinco
          terminales por tienda, inventario y costos, con un cargo por cada
          tienda adicional.
        </p>
        <router-link to="/payments" class="btn-primary"
          >Ver planes y renovar</router-link
        >
        <p>Los datos y el historial de tu negocio se conservan.</p>
      </template>
      <button
        type="button"
        class="btn-secondary"
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
  max-width: 48rem;
  margin: 0 auto;
}
.membership-page h1 {
  font-size: clamp(1.5rem, 4vw, 2rem);
}
</style>
