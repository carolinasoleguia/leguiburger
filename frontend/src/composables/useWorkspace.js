import { computed, onMounted, ref, watch } from "vue";
import { useAuth } from "./useAuth.js";
import { apiFetch, getJSON } from "../services/api.js";

export function useWorkspace(load) {
  const auth = useAuth(),
    tenant = ref(auth.tenantId.value || ""),
    tenants = ref([]);
  const loading = ref(true),
    busy = ref(false),
    error = ref(""),
    notice = ref("");
  const platform = computed(() =>
    ["owner", "super_admin"].includes(auth.userRole.value),
  );
  const manager = computed(
    () => platform.value || auth.userRole.value === "admin",
  );
  const cashier = computed(
    () => platform.value || auth.userRole.value === "cashier",
  );
  const shellRole = computed(() =>
    platform.value ? "owner" : manager.value ? "admin" : "employee",
  );
  const keys = new Map();
  let ready = false,
    revision = 0;
  async function request(
    path,
    body,
    method = body === undefined ? "GET" : "POST",
  ) {
    const scope = tenant.value,
      key = `${scope}:${method}:${path}:${JSON.stringify(body)}`;
    if (method !== "GET" && !keys.has(key)) keys.set(key, crypto.randomUUID());
    const response = await apiFetch(path, {
      method,
      signal: AbortSignal.timeout(25000),
      headers: {
        ...(scope ? { "X-Tenant-ID": scope } : {}),
        ...(method !== "GET" ? { "Idempotency-Key": keys.get(key) } : {}),
      },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok)
      throw new Error(
        data.message ||
          "No pudimos completar la operación. Reintentá en unos momentos.",
      );
    if (method !== "GET") keys.delete(key);
    return data;
  }
  async function refresh() {
    const current = ++revision;
    loading.value = true;
    error.value = "";
    try {
      if (!ready) {
        if (manager.value) {
          tenants.value = ((await getJSON("/tenants")) || []).filter(
            (t) => t.active !== false,
          );
          tenant.value = tenants.value[0]?.id || "";
        }
        ready = true;
      }
      await load({
        request,
        tenant: tenant.value,
        platform: platform.value,
        manager: manager.value,
        current: () => current === revision,
      });
    } catch (e) {
      if (current === revision) error.value = e.message;
    } finally {
      if (current === revision) loading.value = false;
    }
  }
  async function act(fn, message = "Cambios guardados.") {
    if (busy.value) return;
    busy.value = true;
    error.value = "";
    notice.value = "";
    try {
      await fn();
      notice.value = message;
      await refresh();
    } catch (e) {
      error.value = e.message;
    } finally {
      busy.value = false;
    }
  }
  watch(tenant, (_next, previous) => {
    notice.value = "";
    if (ready && previous) refresh();
  });
  onMounted(refresh);
  return {
    auth,
    tenant,
    tenants,
    loading,
    busy,
    error,
    notice,
    platform,
    manager,
    cashier,
    shellRole,
    request,
    refresh,
    act,
  };
}
export const money = (value, currency = "ARS") =>
  value == null
    ? "Sin datos"
    : new Intl.NumberFormat("es-AR", { style: "currency", currency }).format(
        value,
      );
export const when = (value) =>
  value
    ? new Date(value).toLocaleString("es-AR", {
        timeZone: "America/Argentina/Buenos_Aires",
      })
    : "—";
export const statusLabels = {
  pending: "Pendiente",
  accepted: "Aceptado",
  preparing: "En preparación",
  ready: "Listo",
  dispatched: "En camino",
  delivered: "Entregado",
  canceled: "Cancelado",
  rejected: "Rechazado",
};
