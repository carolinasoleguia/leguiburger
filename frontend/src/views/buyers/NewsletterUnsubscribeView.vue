<script setup>
import { ref } from "vue";
import BrandLogo from "../../components/BrandLogo.vue";
import ThemeToggle from "../../components/ThemeToggle.vue";
const token = new URLSearchParams(location.hash.slice(1)).get("token") || "";
history.replaceState(null, "", location.pathname);
const busy = ref(false),
  done = ref(false),
  error = ref("");
async function unsubscribe() {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    const r = await fetch("/api/public/newsletter/unsubscribe", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    });
    const data = await r.json();
    if (!r.ok) throw new Error(data.message || "No pudimos guardar la baja.");
    done.value = true;
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <main class="unsubscribe-page">
    <header>
      <router-link to="/"><BrandLogo /></router-link><ThemeToggle />
    </header>
    <section class="card operation-form">
      <h1>
        {{ done ? "Suscripción cancelada" : "Dejar de recibir novedades" }}
      </h1>
      <p v-if="done" role="status">
        No recibirás nuevas campañas de esta marca. Podés seguir comprando y
        recibir los avisos de tus pedidos.
      </p>
      <template v-else
        ><p>
          Esta baja se aplica a la marca que te envió el correo. Tus otras
          preferencias se conservan.
        </p>
        <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
        <button
          class="btn-primary"
          :disabled="busy || !token"
          @click="unsubscribe"
        >
          {{ busy ? "Guardando…" : "Confirmar baja" }}
        </button>
        <p v-if="!token">
          El enlace está incompleto. Podés gestionar tus suscripciones desde Mi
          cuenta.
        </p></template
      ><router-link class="btn-secondary" to="/mi-cuenta"
        >Ir a mi cuenta</router-link
      >
    </section>
  </main>
</template>
<style scoped>
.unsubscribe-page {
  max-width: 700px;
  margin: 0 auto;
  padding: 2rem 1.25rem;
}
.unsubscribe-page header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 3rem;
}
</style>
