<script setup>
import { ref, watch } from "vue";
import WorkspaceFrame from "../../components/WorkspaceFrame.vue";
import LoadingRows from "../../components/LoadingRows.vue";
import ListPager from "../../components/ListPager.vue";
import { useWorkspace, money, when } from "../../composables/useWorkspace.js";
import { confirmAction } from "../../services/confirmation.js";
const shifts = ref([]),
  active = ref(null),
  detail = ref(null),
  entries = ref([]),
  offset = ref(0),
  entryOffset = ref(0),
  currency = ref("ARS"),
  opening = ref(""),
  counted = ref(""),
  closeNote = ref(""),
  movementKind = ref("cash_out"),
  amount = ref(""),
  note = ref("");
const labels = {
  charge: "Cobro en efectivo",
  refund: "Devolución en efectivo",
  cash_in: "Ingreso de efectivo",
  cash_out: "Retiro de efectivo",
};
const w = useWorkspace(async ({ request, tenant, current }) => {
  if (!tenant) return;
  const list = await request("/cash/shifts?limit=25&offset=" + offset.value);
  if (!current()) return;
  shifts.value = list.items || [];
  // Opening another page of history must not lose the active drawer.
  const first = offset.value
    ? await request("/cash/shifts?limit=25&offset=0")
    : list;
  if (!current()) return;
  active.value =
    (first.items || []).find(
      (x) => !x.closed_at && x.currency === currency.value,
    ) || null;
  const id = detail.value?.id || active.value?.id;
  if (id) {
    const [snapshot, ledger] = await Promise.all([
      request("/cash/shifts/" + id),
      request(
        "/cash/shifts/" + id + "/entries?limit=25&offset=" + entryOffset.value,
      ),
    ]);
    if (current()) {
      detail.value = snapshot;
      entries.value = ledger.items || [];
    }
  } else {
    detail.value = null;
    entries.value = [];
  }
});
function decimal(value) {
  return String(value).trim().replace(",", ".");
}
function open() {
  return w.act(async () => {
    const data = await w.request("/cash/shifts", {
      opening_balance: decimal(opening.value),
      currency: currency.value,
    });
    detail.value = data;
    opening.value = "";
  }, "Turno abierto. Los cobros y devoluciones en efectivo se registran automáticamente.");
}
function move() {
  return w.act(async () => {
    await w.request("/cash/shifts/" + active.value.id + "/entries", {
      kind: movementKind.value,
      amount: decimal(amount.value),
      note: note.value,
    });
    amount.value = "";
    note.value = "";
    detail.value = { id: active.value.id };
    entryOffset.value = 0;
  }, "Movimiento de efectivo registrado.");
}
async function close() {
  const accepted = await confirmAction(
    "Se guardarán el efectivo contado y la diferencia con los movimientos registrados. El cierre no se podrá editar.",
    "Cerrar este turno de caja",
  );
  if (!accepted) return;
  return w.act(async () => {
    detail.value = await w.request(
      "/cash/shifts/" + active.value.id + "/close",
      { counted_balance: decimal(counted.value), note: closeNote.value },
    );
    counted.value = "";
    closeNote.value = "";
    entryOffset.value = 0;
  }, "Caja cerrada. El arqueo quedó guardado con su historial.");
}
function select(item) {
  detail.value = { id: item.id };
  entryOffset.value = 0;
  w.refresh();
}
function page(n) {
  offset.value = n;
  w.refresh();
}
function entriesPage(n) {
  entryOffset.value = n;
  w.refresh();
}
watch(w.tenant, () => {
  offset.value = 0;
  entryOffset.value = 0;
  active.value = null;
  detail.value = null;
  entries.value = [];
  opening.value = "";
  counted.value = "";
  amount.value = "";
  closeNote.value = "";
  note.value = "";
});
watch(currency, () => {
  detail.value = null;
  entryOffset.value = 0;
  w.refresh();
});
</script>
<template>
  <WorkspaceFrame
    :workspace="w"
    title="Turnos de caja"
    description="Abrí el turno, registrá ingresos y retiros, y cerrá con el efectivo contado."
  >
    <div class="cash-intro">
      <router-link class="btn-secondary" to="/orders"
        >Ir a pedidos y cobros</router-link
      ><label
        >Moneda<select v-model="currency" :disabled="w.busy.value">
          <option>ARS</option>
          <option>USD</option>
        </select></label
      ><span role="status">{{
        w.loading.value
          ? "Consultando caja…"
          : active
            ? "Turno abierto"
            : "Sin turno abierto"
      }}</span>
    </div>
    <p class="inline-notice">
      El arqueo incluye solo efectivo. Tarjetas, transferencias y Mercado Pago
      se consultan en Cobros. Desde la primera apertura, necesitás un turno
      abierto para registrar efectivo en esa moneda; los cobros anteriores
      conservan su historial.
    </p>
    <div class="cash-summary" aria-label="Arqueo seleccionado">
      <article class="card">
        <h3>Efectivo inicial</h3>
        <strong>{{
          money(detail?.opening_balance || 0, detail?.currency || currency)
        }}</strong>
      </article>
      <article class="card">
        <h3>Cobros / devoluciones</h3>
        <strong>{{
          money(detail?.cash_charged || 0, detail?.currency || currency)
        }}</strong><small
          >{{
            money(detail?.cash_refunded || 0, detail?.currency || currency)
          }}
          devueltos</small
        >
      </article>
      <article class="card">
        <h3>Efectivo esperado</h3>
        <strong>{{
          money(detail?.expected_balance || 0, detail?.currency || currency)
        }}</strong>
      </article>
      <article class="card">
        <h3>Diferencia al cierre</h3>
        <strong>{{
          detail?.closed_at
            ? money(detail.difference, detail.currency)
            : "Pendiente"
        }}</strong><small v-if="detail?.closed_at"
          >Contado: {{ money(detail.counted_balance, detail.currency) }}</small
        >
      </article>
    </div>
    <div class="cash-operation-region">
      <div class="cash-context">
        <template v-if="active && detail?.id !== active.id"
          ><span>Estás consultando otro arqueo.</span
          ><button class="text-button" @click="select(active)">
            Volver al turno abierto
          </button></template
        ><span v-else>{{
          active
            ? "Los movimientos se registran en el turno abierto."
            : "Abrí un turno para comenzar a controlar el efectivo."
        }}</span>
      </div>
      <div class="cash-forms">
        <form
          v-if="!active"
          v-form-care
          class="card operation-form"
          @submit.prevent="open"
        >
          <h2>Abrir turno</h2>
          <p>Contá el efectivo disponible en la caja antes de comenzar.</p>
          <label
            >Efectivo inicial<input
              v-model="opening"
              inputmode="decimal"
              type="text"
              pattern="[0-9]{1,14}([.,][0-9]{1,2})?"
              maxlength="17"
              required /></label
          ><button
            class="btn-primary"
            :disabled="w.busy.value || w.loading.value || !w.tenant.value"
          >
            Abrir caja en {{ currency }}
          </button>
        </form>
        <form
          v-else
          v-form-care
          class="card operation-form"
          @submit.prevent="move"
        >
          <h2>Ingreso o retiro</h2>
          <label
            >Movimiento<select
              v-model="movementKind"
              :disabled="detail?.id !== active.id"
            >
              <option value="cash_out">Retiro de efectivo</option>
              <option value="cash_in">Ingreso de efectivo</option>
            </select></label
          ><label
            >Importe<input
              v-model="amount"
              type="text"
              inputmode="decimal"
              pattern="[0-9]{1,14}([.,][0-9]{1,2})?"
              maxlength="17"
              :disabled="detail?.id !== active.id"
              required /></label
          ><label
            >Motivo<input
              v-model="note"
              required
              minlength="4"
              maxlength="1000"
              :disabled="detail?.id !== active.id"
              placeholder="Cambio para el turno, retiro a caja fuerte…" /></label
          ><button
            class="btn-secondary"
            :disabled="w.busy.value || detail?.id !== active.id"
          >
            Registrar movimiento
          </button>
        </form>
        <form v-form-care class="card operation-form" @submit.prevent="close">
          <h2>Cerrar turno</h2>
          <p>
            {{
              active
                ? `Abierto el ${when(active.opened_at)}. Contá el efectivo para guardar el arqueo.`
                : "El cierre se habilita después de abrir la caja. Se conservará el arqueo de cada turno."
            }}
          </p>
          <label
            >Efectivo contado<input
              v-model="counted"
              type="text"
              inputmode="decimal"
              pattern="[0-9]{1,14}([.,][0-9]{1,2})?"
              maxlength="17"
              :disabled="!active || detail?.id !== active.id"
              required /></label
          ><label
            >Nota de cierre<textarea
              v-model="closeNote"
              maxlength="1000"
              :disabled="!active || detail?.id !== active.id"
              placeholder="Explicá el motivo si hay una diferencia"
            /></label
          ><button
            class="btn-primary"
            :disabled="w.busy.value || !active || detail?.id !== active.id"
          >
            Revisar y cerrar caja
          </button>
        </form>
      </div>
    </div>
    <h2>Historial de turnos</h2>
    <div class="card table-scroll">
      <table class="data-table">
        <thead>
          <tr>
            <th>Apertura</th>
            <th>Cierre</th>
            <th>Moneda</th>
            <th>Contado</th>
            <th>Diferencia</th>
            <th>Detalle</th>
          </tr>
        </thead>
        <tbody>
          <LoadingRows v-if="w.loading.value" :columns="6" />
          <tr v-for="shift in shifts" v-show="!w.loading.value" :key="shift.id">
            <td>{{ when(shift.opened_at) }}</td>
            <td>{{ shift.closed_at ? when(shift.closed_at) : "Abierto" }}</td>
            <td>{{ shift.currency }}</td>
            <td>
              {{
                shift.counted_balance == null
                  ? "Pendiente"
                  : money(shift.counted_balance, shift.currency)
              }}
            </td>
            <td>
              {{
                shift.difference == null
                  ? "Pendiente"
                  : money(shift.difference, shift.currency)
              }}
            </td>
            <td>
              <button
                class="text-button"
                :disabled="w.busy.value || w.loading.value"
                @click="select(shift)"
              >
                Ver arqueo
              </button>
            </td>
          </tr>
          <tr v-if="!shifts.length" v-show="!w.loading.value">
            <td colspan="6">Todavía no hay turnos registrados.</td>
          </tr>
        </tbody>
      </table>
    </div>
    <ListPager
      :offset="offset"
      :count="shifts.length"
      :disabled="w.busy.value || w.loading.value"
      @change="page"
    />
    <section class="cash-ledger">
      <h2>Movimientos del turno del {{ when(detail?.opened_at) }}</h2>
      <p class="cash-closing-note">
        {{
          detail?.closing_note
            ? `Nota de cierre: ${detail.closing_note}`
            : detail?.closed_at
              ? "Sin nota de cierre"
              : "Los movimientos quedarán registrados al operar el turno."
        }}
      </p>
      <div class="card table-scroll">
        <table class="data-table">
          <thead>
            <tr>
              <th>Fecha</th>
              <th>Movimiento</th>
              <th>Importe</th>
              <th>Motivo</th>
            </tr>
          </thead>
          <tbody>
            <LoadingRows v-if="w.loading.value" :columns="4" />
            <tr
              v-for="entry in entries"
              v-show="!w.loading.value"
              :key="entry.id"
            >
              <td>{{ when(entry.created_at) }}</td>
              <td>{{ labels[entry.kind] }}</td>
              <td>{{ money(entry.amount, detail?.currency || currency) }}</td>
              <td>{{ entry.note || "Desde el registro de cobros" }}</td>
            </tr>
            <tr v-if="!entries.length" v-show="!w.loading.value">
              <td colspan="4">Sin movimientos en este turno.</td>
            </tr>
          </tbody>
        </table>
      </div>
      <ListPager
        :offset="entryOffset"
        :count="entries.length"
        :disabled="w.busy.value || w.loading.value"
        @change="entriesPage"
      />
    </section>
  </WorkspaceFrame>
</template>
<style scoped>
.cash-intro {
  display: flex;
  align-items: end;
  flex-wrap: wrap;
  gap: 16px;
  margin-block: 24px;
}
.cash-intro label {
  display: grid;
  gap: 8px;
  font-size: 12px;
}
.cash-intro select {
  min-height: 44px;
  padding: 10px;
  border: 1px solid var(--control-border);
  border-radius: 8px;
  background: var(--field);
  color: var(--text);
}
.cash-intro > span {
  font-size: 13px;
  color: var(--text-muted);
  padding: 12px;
}
.cash-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-block: 24px;
}
.cash-summary .card {
  min-height: 160px;
  padding: 24px;
}
.cash-summary h3 {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0 0 20px;
  min-height: 32px;
}
.cash-summary strong {
  font-size: clamp(20px, 2vw, 28px);
  font-variant-numeric: tabular-nums;
  display: block;
}
.cash-summary small {
  color: var(--text-muted);
  font-size: 12px;
}
.cash-context {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  color: var(--text-muted);
  font-size: 13px;
  flex-wrap: wrap;
}
.cash-forms {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}
.cash-forms .operation-form {
  min-height: 452px;
  margin: 0;
  align-content: start;
}
.cash-forms .operation-form p {
  font-size: 13px;
  min-height: 44px;
  color: var(--text-muted);
}
.cash-forms button {
  margin-top: auto;
}
.cash-ledger > h2 {
  min-height: 48px;
  line-height: 24px;
}
.cash-closing-note {
  height: 64px;
  overflow: auto;
  line-height: 20px;
  color: var(--text-muted);
  font-size: 13px;
}
.cash-ledger > .table-scroll {
  height: 360px;
  overflow: auto;
  scrollbar-gutter: stable;
}
.cash-ledger {
  margin-block: 32px;
}
@media (max-width: 1100px) {
  .cash-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 720px) {
  .cash-ledger > .table-scroll {
    height: 320px;
  }
  .cash-forms {
    grid-template-columns: 1fr;
  }
  .cash-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .cash-summary .card {
    padding: 16px;
  }
  .cash-summary strong {
    font-size: 20px;
    overflow-wrap: anywhere;
  }
}
</style>
