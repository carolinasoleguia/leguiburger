import { reactive } from 'vue';

// Access tokens live only in memory; the refresh credential is an HttpOnly cookie.
for (const key of ['token', 'authUser', 'employee']) localStorage.removeItem(key);
export const session = reactive({ token: '', user: null, ready: false });
let refreshPromise;
let bootstrapPromise;
const channel = typeof BroadcastChannel === 'undefined' ? null : new BroadcastChannel('legui-auth');
const activityKey = 'legui-last-activity';
const pendingLogoutKey = 'legui-logout-pending';
const idleMs = 30 * 60 * 1000;

export function setSession(payload) {
  session.token = payload.token;
  session.user = payload.user || payload.employee;
  session.ready = true;
}
export function clearSession() { session.token = ''; session.user = null; }
export async function authRequest(path, body = {}, token = '') {
  const response = await fetch(`/api/auth/${path}`, {
    method: 'POST', credentials: 'same-origin',
    signal: AbortSignal.timeout(15000),
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...(path === 'refresh' && session.user?.id ? { 'X-Legui-Account': session.user.id } : {}) },
    body: JSON.stringify(body),
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(payload.message || 'No se pudo completar la operación');
    error.status = response.status; error.code = payload.code; throw error;
  }
  return payload;
}
export function refreshSession() {
  if (!refreshPromise) {
    const refresh = async () => {
      if (localStorage.getItem(pendingLogoutKey)) throw new Error('El cierre de sesión está pendiente');
      const payload = await authRequest('refresh');
      setSession(payload);
      return payload;
    };
    refreshPromise = (navigator.locks ? navigator.locks.request('legui-auth-cookie', refresh) : refresh())
      .finally(() => { refreshPromise = undefined; });
  }
  return refreshPromise;
}
export async function logoutSession(reason = '') {
  localStorage.setItem(pendingLogoutKey, 'true');
  clearSession(); channel?.postMessage({ type: 'logout' });
  try {
    const revoke = async () => { await authRequest('logout'); localStorage.removeItem(pendingLogoutKey); };
    if (navigator.locks) await navigator.locks.request('legui-auth-cookie', revoke); else await revoke();
  } finally { window.dispatchEvent(new CustomEvent('legui:logout', { detail: reason })); }
}
export function bootstrapSession() {
  if (!bootstrapPromise) bootstrapPromise = (async () => {
    try {
      if (localStorage.getItem(pendingLogoutKey)) { await authRequest('logout'); localStorage.removeItem(pendingLogoutKey); return; }
      const last = Number(localStorage.getItem(activityKey));
      if (last && Date.now() - last >= idleMs) { await logoutSession('idle'); return; }
      await refreshSession();
    } catch { clearSession(); }
    finally { session.ready = true; }
  })();
  return bootstrapPromise;
}
export function recordActivity() { localStorage.setItem(activityKey, String(Date.now())); }
let lastRecorded = 0;
for (const event of ['pointerdown', 'keydown', 'scroll']) window.addEventListener(event, () => {
  if (session.token && Date.now() - lastRecorded > 15000) { lastRecorded = Date.now(); recordActivity(); }
}, { passive: true });
setInterval(() => {
  const last = Number(localStorage.getItem(activityKey));
  if (session.token && last && Date.now() - last >= idleMs) logoutSession('idle').catch(() => {});
}, 10000);
channel?.addEventListener('message', event => {
  if (event.data?.type === 'logout') { clearSession(); window.dispatchEvent(new CustomEvent('legui:logout')); }
});
