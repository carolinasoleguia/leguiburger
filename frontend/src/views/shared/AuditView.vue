<script setup>
import DashboardSkeleton from "../../components/DashboardSkeleton.vue";
import { onMounted, ref } from 'vue';
import DashboardShell from '../../components/DashboardShell.vue';
import { useAuth } from '../../composables/useAuth.js';
import { apiFetch } from '../../services/api.js';
const auth = useAuth(), items = ref([]), offset = ref(0), error = ref(''), busy = ref(false);
const shellRole = auth.userRole.value === 'admin' ? 'admin' : 'owner';
const labels = { 'auth.password_verified': 'Ingreso con contraseña', 'auth.mfa_verified': 'Segundo factor verificado', 'auth.mfa_recovery_used': 'Ingreso con código de recuperación', 'auth.password_changed': 'Contraseña actualizada', 'auth.password_reset': 'Contraseña recuperada', 'auth.session_revoked': 'Sesión cerrada', 'auth.all_sessions_revoked': 'Todas las sesiones cerradas', 'team.invited': 'Invitación enviada', 'team.invitation_accepted': 'Integrante incorporado', 'team.invitation_revoked': 'Invitación revocada' };
async function load() {
  error.value = ''; busy.value = true;
  try { const response = await apiFetch(`/audit?limit=50&offset=${offset.value}`); const body = await response.json(); if (!response.ok) throw new Error(body.message); items.value = body.items || []; }
  catch (e) { error.value = e.message || 'No pudimos cargar la actividad'; } finally { busy.value = false; }
}
function page(delta) { offset.value = Math.max(0, offset.value + delta); load(); }
onMounted(load);
</script>

<template><DashboardShell :role="shellRole"><section><div class="section-header"><div class="section-title"><h2>Actividad y auditoría</h2><p>Consultá los cambios de tu equipo y los accesos a las cuentas.</p></div><button class="btn-secondary" :disabled="busy" @click="load">Actualizar</button></div><p v-if="error" role="alert" class="error-msg">{{ error }}</p><DashboardSkeleton v-if="busy" view="table" label="Cargando actividad"/><div v-show="!busy" class="card audit-table"><table class="data-table"><thead><tr><th>Cuándo</th><th>Quién</th><th>Acción</th><th>Recurso</th></tr></thead><tbody><tr v-for="item in items" :key="item.id"><td>{{ new Date(item.occurred_at).toLocaleString('es-AR') }}</td><td>{{ item.actor_name }}</td><td>{{ labels[item.event_name] || item.event_name }}</td><td class="resource-id">{{ item.target_id || '—' }}</td></tr><tr v-if="!items.length"><td colspan="4">{{ busy ? 'Cargando actividad…' : 'Todavía no hay actividad para mostrar.' }}</td></tr></tbody></table></div><div class="pagination"><button class="btn-secondary" :disabled="busy || offset === 0" @click="page(-50)">Anterior</button><span>Página {{ offset / 50 + 1 }}</span><button class="btn-secondary" :disabled="busy || items.length < 50" @click="page(50)">Siguiente</button></div></section></DashboardShell></template>

<style scoped>.audit-table { overflow-x: auto; }.resource-id { font-size: 11px; max-width: 220px; overflow-wrap: anywhere; }.pagination { display: flex; align-items: center; justify-content: flex-end; gap: 20px; margin-top: 24px; }</style>
