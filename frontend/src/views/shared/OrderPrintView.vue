<script setup>
import { ref, onMounted } from "vue";
import { useRoute } from "vue-router";
import { apiFetch } from "../../services/api.js";
import { money, when, statusLabels } from "../../composables/useWorkspace.js";
const route = useRoute(),
  ticket = ref(null),
  loading = ref(true),
  error = ref("");
const kind = route.query.kind === "kitchen" ? "kitchen" : "receipt";
async function load() {
  loading.value = true;
  error.value = "";
  try {
    const response = await apiFetch(
      `/orders/${encodeURIComponent(route.params.id)}/ticket?kind=${kind}`,
      {
        headers: { "X-Tenant-ID": String(route.query.tenant || "") },
        signal: AbortSignal.timeout(20000),
      },
    );
    const data = await response.json();
    if (!response.ok)
      throw new Error(data.message || "No pudimos cargar el comprobante.");
    ticket.value = data;
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
function print() {
  window.print();
}
onMounted(load);
</script>
<template>
  <main class="ticket-page">
    <nav class="ticket-controls" aria-label="Impresión">
      <button class="btn-primary" :disabled="loading || !ticket" @click="print">
        Imprimir {{ kind === "kitchen" ? "comanda" : "comprobante" }}</button
      ><router-link
        class="btn-secondary"
        :to="kind === 'kitchen' ? '/kitchen' : '/orders'"
        >Volver</router-link
      ><small>Elegí papel de 80 mm en el diálogo de impresión.</small>
    </nav>
    <p v-if="loading" class="ticket-loading" role="status">
      Preparando el comprobante…
    </p>
    <div v-else-if="error" role="alert">
      <p>{{ error }}</p>
      <button class="btn-secondary" @click="load">Reintentar</button>
    </div>
    <article v-else-if="ticket" class="ticket-paper">
      <header>
        <h1>{{ ticket.business }}</h1>
        <p>{{ ticket.location }}</p>
        <h2>
          {{
            kind === "kitchen" ? "COMANDA DE COCINA" : "COMPROBANTE DE PEDIDO"
          }}
        </h2>
        <strong class="ticket-number">#{{ ticket.id.slice(0, 8) }}</strong>
        <p>{{ when(ticket.created_at) }}</p>
        <p>
          {{ statusLabels[ticket.status] }} ·
          {{
            ticket.fulfillment === "delivery"
              ? "Envío a domicilio"
              : "Retiro en el local"
          }}
        </p>
      </header>
      <section v-if="kind === 'receipt' && ticket.name">
        <strong>{{ ticket.name }}</strong>
        <p v-if="ticket.phone">{{ ticket.phone }}</p>
        <p v-if="ticket.address">{{ ticket.address }}</p>
        <p v-if="ticket.zone">
          {{ ticket.zone }} · CP {{ ticket.postal_code }}
        </p>
      </section>
      <section aria-label="Productos">
        <div v-for="(item, i) in ticket.items" :key="i" class="ticket-line">
          <strong>{{ item.quantity }} × {{ item.name }}</strong>
          <p v-for="(extra, j) in item.extras" :key="j">
            + {{ extra.quantity }} {{ extra.name }} por unidad
          </p>
          <p v-if="item.note" class="ticket-note">Nota: {{ item.note }}</p>
        </div>
      </section>
      <section v-if="kind === 'receipt'" class="ticket-totals">
        <p v-if="Number(ticket.discount)">
          Descuento: {{ money(ticket.discount, ticket.currency) }}
        </p>
        <p v-if="Number(ticket.shipping)">
          Envío: {{ money(ticket.shipping, ticket.currency) }}
        </p>
        <strong>Total: {{ money(ticket.total_due, ticket.currency) }}</strong>
        <p>Recibido: {{ money(ticket.total_paid, ticket.currency) }}</p>
        <p>
          Pendiente:
          {{
            money(
              Number(ticket.total_due) - Number(ticket.total_paid),
              ticket.currency,
            )
          }}
        </p>
      </section>
      <footer>Pedido #{{ ticket.id }}</footer>
    </article>
  </main>
</template>
<style>
.ticket-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 2rem 1rem;
}
.ticket-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 2rem;
}
.ticket-paper {
  box-sizing: border-box;
  width: 72mm;
  max-width: 100%;
  margin: auto;
  padding: 4mm;
  background: #fff;
  color: #111;
  font:
    13px/1.45 ui-monospace,
    monospace;
  overflow-wrap: anywhere;
  border: 1px solid #ddd;
}
.ticket-paper header {
  text-align: center;
}
.ticket-paper h1 {
  font-size: 20px;
  margin: 0;
}
.ticket-paper h2 {
  font-size: 14px;
  margin: 1rem 0;
}
.ticket-number {
  font-size: 25px;
}
.ticket-paper p {
  margin: 0.35rem 0;
}
.ticket-paper section {
  border-top: 1px dashed #777;
  padding-top: 3mm;
  margin-top: 3mm;
}
.ticket-line {
  padding-bottom: 3mm;
}
.ticket-note {
  font-weight: 700;
}
.ticket-paper footer {
  font-size: 10px;
  border-top: 1px dashed #777;
  padding-top: 4mm;
  margin-top: 3mm;
}
.ticket-totals > strong {
  font-size: 17px;
}
@media print {
  @page orderTicket {
    size: auto;
    margin: 4mm;
  }
  html:has(.ticket-page),
  body:has(.ticket-page),
  #app:has(.ticket-page) {
    background: #fff !important;
    margin: 0 !important;
    padding: 0 !important;
    min-width: 0 !important;
  }
  .ticket-page {
    padding: 0;
    margin: 0;
    width: 72mm;
    max-width: 72mm;
  }
  .ticket-controls,
  .ticket-loading,
  .app-announcer,
  .confirmation-overlay {
    display: none !important;
  }
  .ticket-paper {
    page: orderTicket;
    border: 0;
    margin: 0;
    padding: 0;
    max-width: 72mm;
  }
  .ticket-line {
    break-inside: avoid;
  }
  .ticket-paper header {
    break-after: avoid;
  }
}
</style>
