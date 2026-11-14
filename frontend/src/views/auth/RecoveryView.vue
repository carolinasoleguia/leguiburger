<script setup>
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import BrandLogo from "../../components/BrandLogo.vue";
import { authRequest } from "../../services/session.js";
const route = useRoute();
const mode = computed(() => route.meta.authAction);
const invited = computed(() =>
  ["invite", "business-invite"].includes(mode.value),
);
const currentPassword = ref("");
const token =
  new URLSearchParams(window.location.hash.slice(1)).get("token") || "";
if (token) window.history.replaceState(null, "", window.location.pathname);
const email = ref(""),
  password = ref(""),
  confirmation = ref(""),
  firstName = ref(""),
  lastName = ref("");
const busy = ref(false),
  error = ref(""),
  message = ref("");
const title = computed(() =>
  mode.value === "forgot"
    ? "Recuperá tu acceso."
    : mode.value === "business-invite"
      ? "Tu restaurante empieza acá."
      : mode.value === "invite"
        ? "Sumate al equipo."
        : "Elegí tu nueva contraseña.",
);
async function submit() {
  error.value = "";
  if (
    mode.value !== "forgot" &&
    !currentPassword.value &&
    password.value !== confirmation.value
  ) {
    error.value = "Las contraseñas no coinciden";
    return;
  }
  busy.value = true;
  try {
    const path =
      mode.value === "forgot"
        ? "password/forgot"
        : mode.value === "business-invite"
          ? "business-invitations/accept"
          : mode.value === "invite"
            ? "invitations/accept"
            : "password/reset";
    const body =
      mode.value === "forgot"
        ? { email: email.value }
        : {
            token,
            password: password.value,
            ...(invited.value
              ? {
                  first_name: firstName.value,
                  last_name: lastName.value,
                  ...(currentPassword.value
                    ? { current_password: currentPassword.value }
                    : {}),
                }
              : {}),
          };
    message.value = (await authRequest(path, body)).message;
    password.value = "";
    confirmation.value = "";
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <main class="recovery-page">
    <BrandLogo />
    <section class="card">
      <h1>{{ title }}</h1>
      <p v-if="message" role="status">{{ message }}</p>
      <template v-else
        ><p v-if="mode !== 'forgot' && !token" role="alert">
          Este enlace no es válido. Pedí uno nuevo.
        </p>
        <form v-form-care v-else @submit.prevent="submit">
          <div v-if="mode === 'forgot'" class="input-group">
            <label for="recovery-email">Correo electrónico</label
            ><input
              id="recovery-email"
              v-model="email"
              type="email"
              autocomplete="email"
              required
            />
          </div>
          <template v-else
            ><template v-if="invited"
              ><div class="input-group">
                <label for="first-name">Nombre</label
                ><input
                  id="first-name"
                  v-model="firstName"
                  autocomplete="given-name"
                  maxlength="50"
                  required
                />
              </div>
              <div class="input-group">
                <label for="last-name">Apellido</label
                ><input
                  id="last-name"
                  v-model="lastName"
                  autocomplete="family-name"
                  maxlength="50"
                  required
                /></div
            ></template>
            <div v-if="!currentPassword" class="input-group">
              <label for="new-password">Nueva contraseña</label
              ><input
                id="new-password"
                v-model="password"
                type="password"
                autocomplete="new-password"
                minlength="12"
                maxlength="72"
                required
              /><small
                >Al menos 12 caracteres. Podés usar una frase larga.</small
              >
            </div>
            <div v-if="invited" class="input-group">
              <label for="existing-password"
                >Si ya tenés cuenta en este negocio, tu contraseña actual</label
              ><input
                id="existing-password"
                v-model="currentPassword"
                type="password"
                autocomplete="current-password"
              />
            </div>
            <div v-if="!currentPassword" class="input-group">
              <label for="confirm-password">Repetí la contraseña</label
              ><input
                id="confirm-password"
                v-model="confirmation"
                type="password"
                autocomplete="new-password"
                required
              /></div
          ></template>
          <p v-if="error" role="alert" class="error-msg">{{ error }}</p>
          <button class="btn-primary" :disabled="busy">
            {{
              busy
                ? "Procesando…"
                : mode === "forgot"
                  ? "Enviar enlace"
                  : "Guardar y continuar"
            }}
          </button>
        </form></template
      ><router-link to="/login">Volver al ingreso</router-link>
    </section>
  </main>
</template>

<style scoped>
.recovery-page {
  max-width: 520px;
  margin: 70px auto;
  padding: 24px;
}
.card {
  margin-top: 32px;
  padding: 32px;
}
h1 {
  font-size: 26px;
  margin-bottom: 24px;
}
a {
  display: inline-block;
  margin-top: 24px;
}
small {
  color: var(--text-muted);
}
</style>
