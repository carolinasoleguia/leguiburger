import { reactive, computed } from 'vue';
import { login as authLogin } from '../services/authService.js';

const savedUser = localStorage.getItem('authUser') || localStorage.getItem('employee');
const state = reactive({
  token: localStorage.getItem('token') || '',
  user: savedUser ? JSON.parse(savedUser) : null
});

const isLoggedIn = computed(() => !!state.token && !!state.user);
const userRole = computed(() => state.user?.role || '');
const tenantId = computed(() => state.user?.tenant_id || state.user?.tenantID || '');

function persistState() {
  if (state.token && state.user) {
    localStorage.setItem('token', state.token);
    localStorage.setItem('authUser', JSON.stringify(state.user));
    localStorage.removeItem('employee');
  } else {
    localStorage.removeItem('token');
    localStorage.removeItem('authUser');
    localStorage.removeItem('employee');
  }
}

function setUser(authPayload) {
  state.token = authPayload.token;
  state.user = authPayload.employee || authPayload.user || null;
  persistState();
}

function logout() {
  state.token = '';
  state.user = null;
  persistState();
}

async function login(credentials) {
  const response = await authLogin(credentials);
  setUser(response);
  return response;
}

export function useAuth() {
  return {
    state,
    isLoggedIn,
    userRole,
    tenantId,
    login,
    logout,
    user: computed(() => state.user)
  };
}
