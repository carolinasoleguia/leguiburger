import { computed } from 'vue';
import { session, setSession, logoutSession, authRequest, recordActivity } from '../services/session.js';

export function useAuth() {
  return {
    state: session,
    isLoggedIn: computed(() => Boolean(session.token && session.user)),
    userRole: computed(() => session.user?.role || ''),
    tenantId: computed(() => session.user?.tenant_id || ''),
    user: computed(() => session.user),
    async login(credentials) {
      // Finish any earlier offline logout before accepting a new session cookie.
      if (localStorage.getItem('legui-logout-pending')) {
        await authRequest('logout'); localStorage.removeItem('legui-logout-pending');
      }
      const payload = await (navigator.locks ? navigator.locks.request('legui-auth-cookie', () => authRequest('login', credentials)) : authRequest('login', credentials));
      if (payload.token) { setSession(payload); recordActivity(); }
      return payload;
    },
    completeMFA(payload) { setSession(payload); recordActivity(); },
    logout: logoutSession,
  };
}
