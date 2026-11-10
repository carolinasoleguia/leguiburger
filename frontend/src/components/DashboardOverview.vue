<script setup>
import { computed, onMounted, ref } from 'vue';
import { getJSON } from '../services/api.js';
import { useAuth } from '../composables/useAuth.js';
import AppIcon from './AppIcon.vue';
const props = defineProps({ role: { type: String, default: 'owner' } });
const auth = useAuth();
const sections = computed(() => props.role === 'owner' ? [
  { title: 'Marcas', icon: 'burger', path: '/brands', to: '/owner/brands', text: 'La identidad de cada negocio.', action: 'Gestionar marcas' },
  { title: 'Locales', icon: 'store', path: '/tenants', to: '/owner/tenants', text: 'Cada sucursal, en un mismo lugar.', action: 'Ver locales' },
  { title: 'Administradores', icon: 'users', path: '/users', to: '/owner/admins', text: 'Las personas al frente del negocio.', action: 'Administrar accesos' },
] : [
  { title: 'Locales', icon: 'store', path: '/tenants', to: '/admin/catalog/tenants', text: 'Disponibilidad y precios por sucursal.', action: 'Ver sucursales' },
  { title: 'Productos', icon: 'burger', path: '/products', to: '/admin/catalog', text: 'Una carta que invita a volver.', action: 'Abrir catálogo' },
  { title: 'Producción', icon: 'flame', to: '/admin/production', text: 'Registrá lo que pasa en la cocina.', action: 'Ver producción' },
]);
const metrics = ref({}), loading = ref(true), failed = ref([]);
const firstName = computed(() => auth.user.value?.name?.split(' ')[0] || '');
const date = new Intl.DateTimeFormat('es-AR', { weekday: 'long', day: 'numeric', month: 'long' }).format(new Date());
const locals = ref([]);
async function load() {
  loading.value = true; failed.value = [];
  await Promise.all(sections.value.filter(s => s.path).map(async section => {
    try {
      const brand = auth.user.value?.brand_id || auth.user.value?.brandID;
      const query = props.role === 'admin' && brand ? `?brand_id=${encodeURIComponent(brand)}` : '';
      const data = await getJSON(section.path + query);
      if (!Array.isArray(data)) throw new Error('Formato inesperado');
      metrics.value[section.path] = data.length;
      if (section.path === '/tenants') locals.value = data;
    } catch { failed.value.push(section.path); }
  }));
  loading.value = false;
}
onMounted(load);
</script>
<template>
  <section class="overview">
    <header class="overview-heading"><div><p class="eyebrow">TU NEGOCIO, EN UN VISTAZO</p><h1>Todo a punto{{ firstName ? ', ' + firstName : '' }}.</h1><p>Marcas, personas y cocina. Cada cosa en su lugar.</p></div><span class="date-label"><AppIcon name="clock" :size="16" />{{ date }}</span></header>
    <div class="overview-banner"><span class="banner-icon"><AppIcon name="flame" :size="26" /></span><div><strong>El sabor es tuyo. El orden, también.</strong><p>{{ role === 'owner' ? 'Organizá tus marcas, acompañá a tus locales y dale a tu equipo el acceso que necesita.' : 'Prepará el catálogo, organizá al equipo y llevá el registro de la producción de tus locales.' }}</p></div><router-link class="btn-primary" :to="role === 'owner' ? '/owner/tenants' : '/admin/production'">{{ role === 'owner' ? 'Ver mis locales' : 'Ir a producción' }}<AppIcon name="arrow" :size="17" /></router-link></div>
    <div class="metrics-grid" :aria-busy="loading"><router-link v-for="section in sections" :key="section.title" :to="section.to" class="metric-card"><div class="metric-label"><span>{{ section.title }}</span><span class="metric-icon"><AppIcon :name="section.icon" /></span></div><strong class="metric-number">{{ !section.path ? 'Ir a cocina' : loading ? '…' : failed.includes(section.path) ? '—' : metrics[section.path] }}</strong><span class="metric-caption">{{ section.path ? failed.includes(section.path) ? 'No se pudo cargar' : 'Registros en tu negocio' : 'Registro diario de producción' }}</span><span class="metric-link">{{ section.action }}<AppIcon name="arrow" :size="16" /></span></router-link></div>
    <p v-if="failed.length" class="inline-notice" role="status">No pudimos cargar algunos datos del resumen. <button class="text-button" @click="load">Volver a intentar</button></p>
    <div class="overview-columns"><section class="card local-summary"><div class="panel-heading"><div><p class="eyebrow">CERCA DE CADA SUCURSAL</p><h2>Tus locales</h2></div><router-link class="text-link" :to="role === 'owner' ? '/owner/tenants' : '/admin/catalog/tenants'">Ver todos<AppIcon name="arrow" :size="16" /></router-link></div><div v-if="loading" class="empty-state">Cargando tus locales…</div><div v-else-if="failed.includes('/tenants')" class="empty-state"><AppIcon name="store" :size="32" /><h3>No pudimos consultar los locales</h3><button class="btn-secondary" @click="load">Reintentar</button></div><div v-else-if="!locals.length" class="empty-state"><AppIcon name="store" :size="36" /><h3>Acá empieza tu próximo local.</h3><p>Los locales que registres aparecerán en este espacio.</p><router-link v-if="role === 'owner'" to="/owner/tenants" class="btn-secondary">Gestionar locales</router-link></div><div v-else class="local-list"><router-link v-for="local in locals.slice(0, 5)" :key="local.id" :to="role === 'owner' ? '/owner/tenants' : '/admin/catalog/tenants'" class="local-row"><span class="local-icon"><AppIcon name="store" /></span><div><strong>{{ local.name || local.subdomain || 'Local' }}</strong><small>{{ local.brand?.name || 'Sucursal de tu negocio' }}</small></div><AppIcon name="arrow" :size="17" /></router-link></div></section><section class="card quick-panel"><p class="eyebrow">A MANO, SIEMPRE</p><h2>Accesos rápidos</h2><router-link v-for="section in sections" :key="section.to" :to="section.to" class="quick-link"><AppIcon :name="section.icon" /><span><strong>{{ section.title }}</strong><small>{{ section.text }}</small></span><AppIcon name="arrow" :size="17" /></router-link><router-link v-if="role === 'admin'" to="/admin/employees" class="quick-link"><AppIcon name="users" /><span><strong>Equipo</strong><small>Organizá las personas de tu local.</small></span><AppIcon name="arrow" :size="17" /></router-link></section></div>
  </section>
</template>
