<script setup>
import { ref, watch } from "vue";
import { apiFetch } from "../services/api.js";
const props = defineProps({ tenant: String });
const published = ref(false),
  ordering = ref(false),
  loading = ref(true),
  busy = ref(false),
  error = ref(""),
  notice = ref(""),
  qr = ref("");
let revision = 0;
const link = ref("");
async function read() {
  const current = ++revision;
  loading.value = true;
  error.value = "";
  qr.value = "";
  published.value = false;
  notice.value = "";
  link.value = new URL(
    "/catalogo/" + props.tenant,
    window.location.origin,
  ).href;
  try {
    const response = await apiFetch("/menu/settings", {
      headers: { "X-Tenant-ID": props.tenant },
    });
    const data = await response.json();
    if (!response.ok)
      throw new Error(data.message || "No pudimos consultar la publicación");
    if (current !== revision) return;
    if (data.path) link.value = new URL(data.path, window.location.origin).href;
    published.value = !!data.published;
    ordering.value = !!data.ordering_enabled;
    if (published.value) await makeQR(current);
  } catch (e) {
    if (current === revision) error.value = e.message;
  } finally {
    if (current === revision) loading.value = false;
  }
}
async function makeQR(current = revision) {
  const { default: QRCode } = await import("qrcode");
  const png = await QRCode.toDataURL(link.value, {
    width: 512,
    margin: 4,
    errorCorrectionLevel: "M",
  });
  if (current === revision) qr.value = png;
}
let pendingKey = "",
  pendingBody = "";
async function toggle(receiving = false) {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  notice.value = "";
  const current = revision;
  const body = JSON.stringify(
    receiving
      ? { ordering_enabled: !ordering.value }
      : { published: !published.value },
  );
  if (body !== pendingBody) {
    pendingBody = body;
    pendingKey = crypto.randomUUID();
  }
  try {
    const response = await apiFetch("/menu/settings", {
      method: "PATCH",
      headers: { "X-Tenant-ID": props.tenant, "Idempotency-Key": pendingKey },
      body,
    });
    const data = await response.json();
    if (!response.ok)
      throw new Error(data.message || "No pudimos guardar la publicación");
    pendingBody = "";
    if (current !== revision) return;
    published.value = data.published;
    ordering.value = !!data.ordering_enabled;
    if (published.value) await makeQR(current);
    else qr.value = "";
    notice.value = receiving
      ? ordering.value
        ? "Pedidos online habilitados"
        : "Pedidos online pausados"
      : published.value
        ? "Carta publicada"
        : "Carta retirada de la vista pública";
  } catch (e) {
    if (current === revision) error.value = e.message;
  } finally {
    busy.value = false;
  }
}
async function copy() {
  try {
    await navigator.clipboard.writeText(link.value);
    notice.value = "Enlace copiado";
  } catch {
    notice.value = "Podés seleccionar y copiar el enlace que se muestra aqu?";
  }
}
watch(() => props.tenant, read, { immediate: true });
</script>
<template>
  <section
    class="card menu-publication"
    :aria-busy="loading || busy"
    aria-labelledby="menu-publication-title"
  >
    <div>
      <p class="eyebrow">La carta de este local</p>
      <h3 id="menu-publication-title">Compartí tu menú</h3>
      <p>
        Publicá las fotos, descripciones, categorías y precios que ya
        configuraste para esta sucursal. También se mostrarán su dirección,
        contacto y horarios.
      </p>
      <p v-if="loading" role="status">Consultando publicación…</p>
      <p v-else>
        <strong>{{ published ? "Publicada" : "Sin publicar" }}</strong> · Los
        costos, recetas y cantidades de stock quedan privados
      </p>
      <div class="operation-actions">
        <button
          class="btn-primary"
          type="button"
          :disabled="loading || busy || !!error"
          @click="toggle(false)"
        >
          {{
            busy
              ? "Guardando…"
              : published
                ? "Dejar de publicar"
                : "Publicar carta"
          }}</button><a
          v-if="published"
          class="btn-secondary"
          :href="link"
          target="_blank"
          rel="noopener"
          >Ver carta</a
        ><button
          v-if="published"
          class="btn-secondary"
          type="button"
          @click="copy"
        >
          Copiar enlace
        </button>
      </div>
      <p v-if="published">
        {{
          ordering
            ? "Recibiendo pedidos online. Revisá Caja y pedidos para aceptarlos."
            : "La carta está visible, pero no recibe pedidos online."
        }}
      </p>
      <button
        v-if="published"
        class="btn-secondary"
        type="button"
        :disabled="loading || busy || !!error"
        @click="toggle(true)"
      >
        {{ ordering ? "Pausar pedidos online" : "Habilitar pedidos online" }}
      </button>
      <p v-if="published" class="menu-share-url">{{ link }}</p>
      <p v-if="error" class="error-msg" role="alert">
        {{ error }}
        <button type="button" class="text-button" @click="read">
          Reintentar
        </button>
      </p>
      <p v-if="notice" role="status">{{ notice }}</p>
    </div>
    <div class="menu-qr">
      <template v-if="published && qr"
        ><img
          :src="qr"
          width="160"
          height="160"
          alt="Código QR de la carta de esta sucursal"
        /><a :href="qr" download="carta-leguiburger.png" class="text-link"
          >Descargar QR</a
        ></template
      ><span v-else>El QR aparece al publicar</span>
    </div>
  </section>
</template>
<style scoped>
.menu-publication {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 180px;
  gap: 24px;
  margin-block: 24px;
  min-height: 284px;
}
.menu-publication h3 {
  margin: 0 0 12px;
}
.menu-publication p {
  color: var(--text-muted);
  font-size: 13px;
}
.menu-qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 208px;
  text-align: center;
  font-size: 12px;
  color: var(--text-muted);
}
.menu-qr img {
  background: white;
  border-radius: 8px;
}
.menu-share-url {
  overflow-wrap: anywhere;
  user-select: all;
}
@media (max-width: 600px) {
  .menu-publication {
    grid-template-columns: 1fr;
  }
  .menu-qr {
    min-height: 208px;
  }
}
</style>
