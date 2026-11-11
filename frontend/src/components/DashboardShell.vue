<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '../composables/useAuth.js';
import AppIcon from './AppIcon.vue';
import BrandLogo from './BrandLogo.vue';
import ThemeToggle from './ThemeToggle.vue';
const props = defineProps({ role: { type: String, default: 'owner' } });
const route = useRoute(), router = useRouter(), auth = useAuth();
const mobileOpen = ref(false), menuButton = ref(null), sidebar = ref(null);
const roleLabel = computed(() => ({ owner: 'Administración general', admin: 'Gestión de la marca', employee: 'Mi espacio de trabajo' }[props.role]));
const navigation = computed(() => props.role === 'owner' ? [
  { label: 'Resumen', to: '/owner', icon: 'grid' }, { label: 'Marcas', to: '/owner/brands', icon: 'burger' },
  { label: 'Locales', to: '/owner/tenants', icon: 'store' }, { label: 'Administradores', to: '/owner/admins', icon: 'users' },
  { label: 'Cobros', to: '/payments', icon: 'grid' },
  { label: 'Invitaciones', to: '/team/invitations', icon: 'users' }, { label: 'Actividad', to: '/audit', icon: 'grid' }, { label: 'Seguridad', to: '/account/security', icon: 'grid' },
] : props.role === 'admin' ? [
  { label: 'Resumen', to: '/admin', icon: 'grid' }, { label: 'Equipo', to: '/admin/employees', icon: 'users' },
  { label: 'Producción', to: '/admin/production', icon: 'flame' }, { label: 'Catálogo', to: '/admin/catalog', icon: 'burger' },
  { label: 'Cobros', to: '/payments', icon: 'grid' },
  { label: 'Invitaciones', to: '/team/invitations', icon: 'users' }, { label: 'Actividad', to: '/audit', icon: 'grid' }, { label: 'Seguridad', to: '/account/security', icon: 'grid' },
] : [{ label: 'Mi espacio', to: `/tenant/${auth.tenantId.value}/employee`, icon: 'grid' }, ...(auth.userRole.value === 'cashier' ? [{ label: 'Cobros', to: '/payments', icon: 'grid' }] : []), { label: 'Seguridad', to: '/account/security', icon: 'grid' }]);
const activeItem = computed(() => [...navigation.value].reverse().find(item => route.path === item.to || route.path.startsWith(item.to + '/')));
const displayName = computed(() => auth.user.value?.name || auth.user.value?.email || 'Mi cuenta');
const initials = computed(() => displayName.value.slice(0, 2).toUpperCase());
watch(() => route.path, () => { mobileOpen.value = false; });
let previousOverflow = '', focusTimer;
watch(mobileOpen, async open => {
  clearTimeout(focusTimer);
  if (open) {
    previousOverflow = document.body.style.overflow; document.body.style.overflow = 'hidden';
    await nextTick();
    focusTimer = setTimeout(() => { if (mobileOpen.value) sidebar.value?.querySelector('button')?.focus(); }, 240);
  } else {
    document.body.style.overflow = previousOverflow;
    await nextTick(); menuButton.value?.focus();
  }
});
onBeforeUnmount(() => { clearTimeout(focusTimer); if (mobileOpen.value) document.body.style.overflow = previousOverflow; });
function trapFocus(event) {
  if (!mobileOpen.value) return;
  if (event.key === 'Escape') { mobileOpen.value = false; return; }
  if (event.key !== 'Tab') return;
  const elements = sidebar.value.querySelectorAll('a[href], button');
  const first = elements[0], last = elements[elements.length - 1];
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
}
async function logout() { try { await auth.logout(); } finally { router.push('/login'); } }
</script>
<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">Ir al contenido</a>
    <div v-if="mobileOpen" class="sidebar-backdrop" @click="mobileOpen = false" />
    <aside ref="sidebar" class="sidebar" :class="{ 'is-open': mobileOpen }" :role="mobileOpen ? 'dialog' : undefined" :aria-modal="mobileOpen || undefined" aria-label="Navegación de LeguiBurger" @keydown="trapFocus">
      <div class="sidebar-brand"><BrandLogo /><button class="icon-button mobile-only" aria-label="Cerrar menú" @click="mobileOpen = false"><AppIcon name="close" /></button></div>
      <p class="sidebar-caption">Tu negocio, a punto.</p>
      <div class="workspace-label"><span class="live-dot" />{{ roleLabel }}</div>
      <span class="nav-heading">ESPACIO DE TRABAJO</span>
      <nav aria-label="Secciones del sistema"><router-link v-for="item in navigation" :key="item.to" :to="item.to" class="nav-link" :class="{ active: activeItem?.to === item.to }" :aria-current="activeItem?.to === item.to ? 'page' : undefined"><AppIcon :name="item.icon" /><span>{{ item.label }}</span><span v-if="activeItem?.to === item.to" class="nav-active-dot" /></router-link></nav>
      <div class="sidebar-bottom"><div class="sidebar-note"><AppIcon name="flame" /><p>Todo listo para<br /><strong>un buen servicio.</strong></p></div><div class="sidebar-account"><span class="avatar">{{ initials }}</span><div><strong>{{ displayName }}</strong><small>{{ roleLabel }}</small></div></div><button class="btn-logout" @click="logout"><AppIcon name="logout" :size="18" />Cerrar sesión</button></div>
    </aside>
    <div class="workspace-main" :inert="mobileOpen || undefined">
      <header class="workspace-header"><div class="workspace-breadcrumb"><button ref="menuButton" class="icon-button mobile-only" aria-label="Abrir menú" :aria-expanded="mobileOpen" @click="mobileOpen = true"><AppIcon name="menu" /></button><span>Mi negocio</span><span class="breadcrumb-slash">/</span><strong>{{ activeItem?.label || 'Mi espacio' }}</strong></div><div class="workspace-header-actions"><span class="workspace-role">{{ role === 'owner' ? 'Propietario' : role === 'admin' ? 'Administrador' : 'Equipo' }}</span><ThemeToggle /><span class="avatar avatar-small">{{ initials }}</span></div></header>
      <main id="main-content" class="content-pane" tabindex="-1"><slot><router-view /></slot></main>
      <footer class="workspace-footer"><span>leguiburger.</span><span>Hecho para el ritmo de tu cocina.</span></footer>
    </div>
  </div>
</template>
