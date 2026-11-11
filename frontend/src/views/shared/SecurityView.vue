<script setup>
import { onMounted, ref } from 'vue';
import DashboardShell from '../../components/DashboardShell.vue';
import { useAuth } from '../../composables/useAuth.js';
import { apiFetch, getJSON, postJSON } from '../../services/api.js';
const auth = useAuth(), sessions = ref([]), error = ref(''), busy = ref(false);
const currentPassword = ref(''), password = ref(''), confirmation = ref('');
const shellRole = auth.userRole.value === 'super_admin' ? 'owner' : ['owner', 'admin'].includes(auth.userRole.value) ? auth.userRole.value : 'employee';
async function load() { try { sessions.value = await getJSON('/auth/sessions') || []; } catch { error.value = 'No pudimos cargar las sesiones'; } }
async function revoke(item) {
  busy.value = true; error.value = '';
  try { const response = await apiFetch(`/auth/sessions/${item.id}`, { method: 'DELETE' }); if (!response.ok) throw new Error('No se pudo cerrar la sesión'); if (item.current) await auth.logout(); else await load(); }
  catch (e) { error.value = e.message; } finally { busy.value = false; }
}
async function changePassword() {
  error.value = ''; if (password.value !== confirmation.value) { error.value = 'Las contraseñas no coinciden'; return; }
  busy.value = true;
  try { await postJSON('/auth/password/change', { current_password: currentPassword.value, password: password.value }); await auth.logout(); }
  catch (e) { error.value = e.message; } finally { currentPassword.value = ''; password.value = ''; confirmation.value = ''; busy.value = false; }
}
async function logoutAll() { busy.value = true; try { await postJSON('/auth/logout-all', {}); await auth.logout(); } catch (e) { error.value = e.message; } finally { busy.value = false; } }
onMounted(load);
</script>

<template><DashboardShell :role="shellRole"><section><div class="section-header"><div class="section-title"><h2>Seguridad de tu cuenta</h2><p>Administrá tus accesos y tu contraseña.</p></div></div><p v-if="error" class="error-msg" role="alert">{{ error }}</p><div class="security-grid"><section class="card"><h3>Sesiones activas</h3><p>Se cierran después de 30 minutos sin actividad.</p><ul class="session-list"><li v-for="item in sessions" :key="item.id"><div><strong>{{ item.current ? 'Esta sesión' : 'Otro dispositivo' }}</strong><p>{{ item.user_agent || 'Dispositivo sin identificar' }}</p><small>Última actividad: {{ new Date(item.last_seen_at).toLocaleString('es-AR') }}</small></div><button class="btn-secondary" :disabled="busy" @click="revoke(item)">Cerrar</button></li></ul><button class="btn-danger" :disabled="busy" @click="logoutAll">Cerrar todas las sesiones</button></section><section class="card"><h3>Cambiar contraseña</h3><p>Este cambio cierra todas tus sesiones.</p><form @submit.prevent="changePassword"><div class="input-group"><label for="current-password">Contraseña actual</label><input id="current-password" v-model="currentPassword" type="password" autocomplete="current-password" required /></div><div class="input-group"><label for="security-password">Nueva contraseña</label><input id="security-password" v-model="password" type="password" autocomplete="new-password" minlength="12" maxlength="72" required /></div><div class="input-group"><label for="security-confirm">Repetí la nueva contraseña</label><input id="security-confirm" v-model="confirmation" type="password" autocomplete="new-password" required /></div><button class="btn-primary" :disabled="busy">Guardar contraseña</button></form></section></div></section></DashboardShell></template>

<style scoped>.security-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }.card { padding: 24px; }.session-list { list-style: none; padding: 0; }.session-list li { display: flex; justify-content: space-between; gap: 16px; padding: 20px 0; border-bottom: 1px solid var(--border); }.session-list p { max-width: 340px; overflow-wrap: anywhere; font-size: 12px; }small { color: var(--text-muted); }@media(max-width:900px){.security-grid{grid-template-columns:1fr;}}</style>
