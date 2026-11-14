import { reactive } from "vue";
export const shopper = reactive({ profile: null, ready: false });
export async function shopperRequest(path, body, method = "POST") {
  const response = await fetch(path, {
    method: body === undefined ? "GET" : method,
    credentials: "same-origin",
    signal: AbortSignal.timeout(20000),
    headers: { "Content-Type": "application/json" },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  const result = await response.json();
  if (!response.ok) {
    const e = new Error(result.message || "No pudimos completar la solicitud.");
    e.status = response.status;
    throw e;
  }
  return result;
}
export async function loadShopper() {
  try {
    shopper.profile = await shopperRequest("/api/shopper/me");
  } catch (e) {
    if (e.status !== 401) throw e;
    shopper.profile = null;
  } finally {
    shopper.ready = true;
  }
  return shopper.profile;
}
