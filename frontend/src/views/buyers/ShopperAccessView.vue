<script setup>
import { onMounted, ref } from "vue";
import BrandLogo from "../../components/BrandLogo.vue";
import ThemeToggle from "../../components/ThemeToggle.vue";
import {
  shopper,
  loadShopper,
  shopperRequest,
} from "../../composables/useShopper.js";
const props = defineProps({ initialAction: String, token: String });
const emit = defineEmits(["complete"]);
const action = ref(props.initialAction || ""),
  mode = ref("login"),
  busy = ref(false),
  loading = ref(true),
  error = ref(""),
  notice = ref("");
const name = ref(""),
  email = ref(""),
  password = ref("");
let actionToken = props.token || "";
async function act(fn, message = "") {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  notice.value = "";
  try {
    await fn();
    if (message) notice.value = message;
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
function submitAuth() {
  return act(async () => {
    if (action.value) {
      await shopperRequest(`/api/shopper/${action.value}`, {
        token: actionToken,
        ...(action.value === "reset" ? { password: password.value } : {}),
      });
      notice.value =
        action.value === "verify"
          ? "Correo confirmado. Ya podés recibir novedades de las marcas que elegiste."
          : "Contraseña actualizada. Ingresá con tu nueva contraseña.";
      action.value = "";
      actionToken = "";
      password.value = "";
      await loadShopper();
      emit("complete", notice.value);
      return;
    }
    if (mode.value === "forgot") {
      const result = await shopperRequest("/api/shopper/forgot", {
        email: email.value,
      });
      notice.value = result.message;
      return;
    }
    shopper.profile = await shopperRequest(`/api/shopper/${mode.value}`, {
      email: email.value,
      password: password.value,
      ...(mode.value === "register" ? { name: name.value } : {}),
    });
    shopper.ready = true;
    password.value = "";
    emit("complete");
  });
}
onMounted(async () => {
  try {
    if (action.value && !actionToken) {
      action.value = "";
      if (!shopper.ready) await loadShopper();
      error.value =
        "El enlace está incompleto. Volvé a abrir el correo original.";
      emit("complete", "", error.value);
    } else if (action.value === "verify") await submitAuth();
    else if (!shopper.ready) await loadShopper();
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
});
</script>
<template>
  <div class="shopper-account">
    <header class="shopper-top">
      <router-link to="/" aria-label="LeguiBurger, inicio"
        ><BrandLogo
      /></router-link>
      <div>
        <router-link to="/restaurantes" class="text-link"
          >Explorar cartas</router-link
        ><ThemeToggle />
      </div>
    </header>
    <main class="shopper-main">
      <section class="shopper-welcome">
        <p class="eyebrow">Tu próxima favorita te espera</p>
        <h1>Todas tus hamburgueserías.<br />Una sola cuenta.</h1>
        <p>
          Guardá tus pedidos, seguí cada entrega y encontrá beneficios de las
          marcas que te gustan.
        </p>
        <router-link to="/restaurantes" class="text-link"
          >También podés comprar sin registrarte →</router-link
        >
      </section>
      <section class="card shopper-auth" :aria-busy="loading || busy">
        <h2>
          {{
            action === "reset"
              ? "Elegí una nueva contraseña"
              : mode === "register"
                ? "Creá tu cuenta"
                : mode === "forgot"
                  ? "Recuperá tu cuenta"
                  : "Qué bueno verte de nuevo"
          }}
        </h2>
        <p v-if="loading" role="status">Consultando tu cuenta…</p>
        <div v-else-if="action === 'verify'">
          <p>Solicitá un nuevo enlace desde tu perfil si el anterior venció.</p>
          <button
            class="btn-secondary"
            @click="
              action = '';
              emit('complete');
            "
          >
            Volver a mi cuenta
          </button>
        </div>
        <form v-else @submit.prevent="submitAuth" class="shopper-form">
          <label v-if="mode === 'register' && !action"
            >Tu nombre<input
              v-model="name"
              autocomplete="name"
              required
              minlength="2"
              maxlength="100"
          /></label>
          <label v-if="!action"
            >Correo<input
              v-model="email"
              autocomplete="email"
              type="email"
              required
              maxlength="254"
          /></label>
          <label v-if="mode !== 'forgot' || action"
            >Contraseña<input
              v-model="password"
              :autocomplete="
                mode === 'register' || action
                  ? 'new-password'
                  : 'current-password'
              "
              type="password"
              :minlength="mode === 'register' || action ? 12 : undefined"
              required
              maxlength="72"
          /></label>
          <p v-if="mode === 'register'" class="shopper-muted">
            Usá al menos 12 caracteres. Las novedades de cada marca se activan
            por separado, si vos querés.
          </p>
          <button class="btn-primary" :disabled="busy">
            {{
              busy
                ? "Un momento…"
                : action
                  ? "Guardar contraseña"
                  : mode === "register"
                    ? "Crear cuenta"
                    : mode === "forgot"
                      ? "Solicitar enlace"
                      : "Ingresar"
            }}
          </button>
          <button
            v-if="!action"
            type="button"
            class="text-button"
            @click="
              mode = mode === 'login' ? 'register' : 'login';
              error = '';
              notice = '';
            "
          >
            {{
              mode === "login"
                ? "¿Primera vez? Creá tu cuenta"
                : "Ya tengo una cuenta"
            }}
          </button>
          <button
            v-if="mode === 'login' && !action"
            type="button"
            class="text-button"
            @click="mode = 'forgot'"
          >
            Olvidé mi contraseña
          </button>
        </form>
      </section>
      <p v-if="error" role="alert" class="error-msg shopper-feedback">
        {{ error }}
      </p>
      <p v-if="notice" role="status" class="shopper-feedback">{{ notice }}</p>
    </main>
  </div>
</template>
<style scoped src="../../styles/shopper-account.css" />
