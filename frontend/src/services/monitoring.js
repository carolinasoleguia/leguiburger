import { apiFetch } from "./api.js";
const last = new Map();
export function reportClientError(code) {
  if (!["render", "navigation", "unhandled"].includes(code)) return;
  if (Date.now() - (last.get(code) || 0) < 60000) return;
  last.set(code, Date.now());
  // Never send Error.message/stack, locations, fragments, input or user data.
  apiFetch("/care/errors", {
    method: "POST",
    body: JSON.stringify({ code }),
    signal: AbortSignal.timeout(3000),
  }).catch(() => {});
}
