<script setup>
import { ref } from "vue";
import { authRequest } from "../../services/session.js";
import BrandLogo from "../../components/BrandLogo.vue";
const token = new URLSearchParams(location.hash.slice(1)).get("token") || "";
history.replaceState(null, "", location.pathname);
const busy = ref(false),
  message = ref(""),
  error = ref("");
async function confirm() {
  busy.value = true;
  error.value = "";
  try {
    const data = await authRequest("email/confirm", { token });
    message.value = data.completed
      ? "Correo actualizado. Todas las sesiones se cerraron. Iniciá sesión con el correo nuevo."
      : "Esta casilla ya está confirmada. Falta confirmar el enlace enviado a la otra casilla.";
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <main class="public-order">
    <BrandLogo />
    <section class="card">
      <h1>Confirmá tu nuevo correo</h1>
      <p>
        El cambio se completa cuando confirmás los enlaces enviados al correo
        anterior y al nuevo.
      </p>
      <p v-if="message" role="status">{{ message }}</p>
      <p v-if="error" role="alert">{{ error }}</p>
      <button
        v-if="!message"
        class="btn-primary"
        :disabled="busy || !token"
        @click="confirm"
      >
        Confirmar esta casilla
      </button>
      <p><router-link to="/login">Ir al inicio de sesión</router-link></p>
    </section>
  </main>
</template>
