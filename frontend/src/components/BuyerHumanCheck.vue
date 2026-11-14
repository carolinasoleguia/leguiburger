<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue";
const props = defineProps({ tenant: { type: String, required: true } });
const emit = defineEmits(["verified", "retry"]);
const title = ref(null);
const target = ref(null),
  loading = ref(true),
  busy = ref(false),
  error = ref("");
const configured = ref(false),
  token = ref("");
let widget,
  alive = true;
const controller = new AbortController();
function loadWidget() {
  if (window.turnstile) return Promise.resolve();
  return new Promise((resolve, reject) => {
    let script = document.querySelector("#checkout-turnstile");
    const timer = setTimeout(
      () =>
        reject(
          new Error(
            "La verificación tardó demasiado. Reintentá en unos minutos.",
          ),
        ),
      12000,
    );
    const done = () => {
      clearTimeout(timer);
      window.turnstile
        ? resolve()
        : reject(new Error("No pudimos iniciar la verificación."));
    };
    if (!script) {
      script = document.createElement("script");
      script.id = "checkout-turnstile";
      script.src =
        "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
      script.async = true;
      script.defer = true;
      script.addEventListener("load", done, { once: true });
      script.addEventListener(
        "error",
        () => {
          clearTimeout(timer);
          script.remove();
          reject(new Error("No pudimos cargar la verificación."));
        },
        { once: true },
      );
      document.head.appendChild(script);
    } else script.addEventListener("load", done, { once: true });
  });
}
async function start() {
  loading.value = true;
  error.value = "";
  token.value = "";
  if (widget !== undefined) {
    window.turnstile?.remove(widget);
    widget = undefined;
  }
  try {
    const response = await fetch(
      `/api/public/menu/${props.tenant}/human-check`,
      { credentials: "same-origin", signal: controller.signal },
    );
    const config = await response.json();
    if (!response.ok)
      throw new Error(config.message || "No pudimos iniciar la verificación.");
    if (!alive) return;
    configured.value = !!config.configured;
    if (!config.configured) return;
    await loadWidget();
    await nextTick();
    if (!alive) return;
    widget = window.turnstile.render(target.value, {
      sitekey: config.site_key,
      action: config.action,
      cData: config.cdata,
      theme: "auto",
      size: "flexible",
      callback: (value) => {
        if (alive) token.value = value;
      },
      "expired-callback": () => {
        if (alive) token.value = "";
      },
      "error-callback": () => {
        if (alive) {
          token.value = "";
          error.value =
            "No pudimos completar la verificación. Volvé a intentarlo.";
        }
      },
    });
  } catch (e) {
    if (alive && e.name !== "AbortError") error.value = e.message;
  } finally {
    if (alive) loading.value = false;
  }
}
async function verify() {
  if (!token.value || busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    const response = await fetch(
      `/api/public/menu/${props.tenant}/human-check`,
      {
        method: "POST",
        credentials: "same-origin",
        signal: controller.signal,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: token.value }),
      },
    );
    const data = await response.json();
    if (!response.ok)
      throw new Error(data.message || "No pudimos verificar el intento.");
    if (alive) emit("verified");
  } catch (e) {
    if (alive && e.name !== "AbortError") {
      error.value = e.message;
      token.value = "";
      if (widget !== undefined) window.turnstile?.reset(widget);
    }
  } finally {
    if (alive) busy.value = false;
  }
}
onMounted(() => {
  title.value?.focus();
  start();
});
onBeforeUnmount(() => {
  alive = false;
  controller.abort();
  if (widget !== undefined) window.turnstile?.remove(widget);
});
</script>
<template>
  <section
    class="buyer-human-check"
    aria-labelledby="buyer-human-title"
    :aria-busy="loading || busy"
  >
    <h3 id="buyer-human-title" ref="title" tabindex="-1">
      Una verificación antes de continuar
    </h3>
    <p>
      Detectamos varios intentos desde esta conexión. Tu carrito y tus datos
      siguen guardados en esta pantalla.
    </p>
    <p v-if="loading" role="status">Preparando la verificación…</p>
    <p v-else-if="!configured">
      La verificación no está disponible por el momento. Esperá 15 minutos o
      contactá al local para hacer tu pedido.
    </p>
    <button
      v-if="!loading && !configured"
      type="button"
      class="btn-secondary"
      @click="emit('retry')"
    >
      Volver a intentar la compra
    </button>
    <div ref="target" />
    <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
    <button
      v-if="configured"
      type="button"
      class="btn-primary"
      :disabled="!token || busy || loading"
      @click="verify"
    >
      {{ busy ? "Verificando…" : "Verificar y continuar" }}
    </button>
    <button
      v-if="error && !configured"
      type="button"
      class="btn-secondary"
      :disabled="busy || loading"
      @click="start"
    >
      Reintentar verificación
    </button>
  </section>
</template>
<style scoped>
.buyer-human-check {
  padding: 16px;
  margin: 12px 0;
  border: 1px solid var(--control-border);
  border-radius: 12px;
  background: var(--surface);
}
h3 {
  margin: 0 0 8px;
  font-size: 16px;
}
p {
  margin: 8px 0;
  line-height: 1.5;
  font-size: 13px;
  color: var(--text-muted);
}
button {
  margin-top: 12px;
  min-height: 44px;
}
</style>
