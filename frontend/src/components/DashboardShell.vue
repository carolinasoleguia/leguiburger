<script setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuth } from "../composables/useAuth.js";
import AppIcon from "./AppIcon.vue";
import BrandLogo from "./BrandLogo.vue";
import ThemeToggle from "./ThemeToggle.vue";
import { restaurantWorkspace } from "../navigation/restaurant.js";
import { platformWorkspace } from "../navigation/platform.js";
import { collaboratorWorkspace } from "../navigation/collaborator.js";
import { getJSON } from "../services/api.js";
import {
  authRequest,
  setSession,
  announceContext,
} from "../services/session.js";
const assignedLocations = ref([]),
  switching = ref(false),
  contextError = ref("");
onMounted(async () => {
  if (workspace.value.locationSwitcher) {
    try {
      assignedLocations.value =
        (await getJSON("/business/locations")).items || [];
    } catch {
      contextError.value = "No pudimos consultar tus locales.";
    }
  }
});
async function changeLocation(event) {
  const targetTenant = event.target.value;
  switching.value = true;
  contextError.value = "";
  try {
    const change = () =>
      authRequest("location", { tenant_id: targetTenant }, auth.state.token);
    const data = await (navigator.locks
      ? navigator.locks.request("legui-auth-cookie", change)
      : change());
    setSession(data);
    announceContext();
    window.location.assign(
      data.user.role === "cashier" ? "/orders" : "/kitchen",
    );
  } catch (e) {
    contextError.value = e.message;
    event.target.value = auth.tenantId.value;
  } finally {
    switching.value = false;
  }
}
const props = defineProps({ role: { type: String, default: "owner" } });
const route = useRoute(),
  router = useRouter(),
  auth = useAuth();
const mobileOpen = ref(false),
  menuButton = ref(null),
  sidebar = ref(null);
const workspace = computed(() =>
  props.role === "owner"
    ? platformWorkspace()
    : props.role === "admin"
      ? restaurantWorkspace()
      : collaboratorWorkspace(auth.userRole.value, auth.tenantId.value),
);
const roleLabel = computed(() => workspace.value.label);
const baseNavigation = computed(() => [workspace.value.home]);
const navigationGroups = computed(() => workspace.value.groups);
const navigation = computed(() => [
  workspace.value.home,
  ...workspace.value.groups.flatMap((group) => group.items),
]);
const openedGroups = ref(new Set(["operation"]));
function toggleGroup(key) {
  openedGroups.value = openedGroups.value.has(key) ? new Set() : new Set([key]);
}
const activeItem = computed(() =>
  [...navigation.value]
    .reverse()
    .find(
      (item) => route.path === item.to || route.path.startsWith(item.to + "/"),
    ),
);
watch(
  () => activeItem.value?.to,
  (to) => {
    const group = navigationGroups.value.find((group) =>
      group.items.some((item) => item.to === to),
    );
    if (group && !openedGroups.value.has(group.key))
      openedGroups.value = new Set([group.key]);
  },
  { immediate: true },
);
const displayName = computed(
  () => auth.user.value?.name || auth.user.value?.email || "Mi cuenta",
);
const initials = computed(() => displayName.value.slice(0, 2).toUpperCase());
watch(
  () => route.path,
  () => {
    mobileOpen.value = false;
  },
);
let previousOverflow = "",
  focusTimer;
watch(mobileOpen, async (open) => {
  clearTimeout(focusTimer);
  if (open) {
    previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    await nextTick();
    focusTimer = setTimeout(() => {
      if (mobileOpen.value) sidebar.value?.querySelector("button")?.focus();
    }, 240);
  } else {
    document.body.style.overflow = previousOverflow;
    await nextTick();
    menuButton.value?.focus();
  }
});
onBeforeUnmount(() => {
  clearTimeout(focusTimer);
  if (mobileOpen.value) document.body.style.overflow = previousOverflow;
});
function trapFocus(event) {
  if (!mobileOpen.value) return;
  if (event.key === "Escape") {
    mobileOpen.value = false;
    return;
  }
  if (event.key !== "Tab") return;
  const elements = sidebar.value.querySelectorAll("a[href], button");
  const first = elements[0],
    last = elements[elements.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}
async function logout() {
  try {
    await auth.logout();
  } finally {
    router.push("/login");
  }
}
</script>
<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">Ir al contenido</a>
    <div
      v-if="mobileOpen"
      class="sidebar-backdrop"
      @click="mobileOpen = false"
    />
    <aside
      ref="sidebar"
      class="sidebar"
      :class="{ 'is-open': mobileOpen }"
      :role="mobileOpen ? 'dialog' : undefined"
      :aria-modal="mobileOpen || undefined"
      aria-label="Navegación de LeguiBurger"
      @keydown="trapFocus"
    >
      <div class="sidebar-brand">
        <BrandLogo /><button
          class="icon-button mobile-only"
          aria-label="Cerrar menú"
          @click="mobileOpen = false"
        >
          <AppIcon name="close" />
        </button>
      </div>
      <p class="sidebar-caption">Tu negocio, a punto.</p>
      <div class="workspace-label">
        <span class="live-dot" />{{ roleLabel }}
      </div>
      <span class="nav-heading">Espacio de trabajo</span>
      <label
        v-if="workspace.locationSwitcher"
        class="input-group sidebar-location-context"
        >Local de trabajo<select
          :value="auth.tenantId.value"
          :disabled="switching || assignedLocations.length < 2"
          @change="changeLocation"
        >
          <option v-if="!assignedLocations.length" :value="auth.tenantId.value">
            Consultando locales…
          </option>
          <option
            v-for="item in assignedLocations"
            :key="item.id"
            :value="item.id"
          >
            {{ item.display_name || item.subdomain }}
          </option>
        </select></label
      >
      <p v-if="contextError" role="alert">{{ contextError }}</p>
      <nav aria-label="Secciones del sistema">
        <router-link
          :to="baseNavigation[0].to"
          class="nav-link"
          :class="{ active: activeItem?.to === baseNavigation[0].to }"
          :aria-current="
            activeItem?.to === baseNavigation[0].to ? 'page' : undefined
          "
          ><AppIcon :name="baseNavigation[0].icon" :size="18" /><span>{{
            baseNavigation[0].label
          }}</span></router-link
        >
        <div
          v-for="group in navigationGroups"
          :key="group.key"
          class="sidebar-nav-group"
        >
          <button
            type="button"
            class="nav-group-toggle"
            :aria-expanded="openedGroups.has(group.key)"
            :aria-controls="`${role}-nav-${group.key}`"
            @click="toggleGroup(group.key)"
          >
            <AppIcon :name="group.icon" :size="18" /><span>{{
              group.label
            }}</span
            ><AppIcon name="chevron" :size="14" class="nav-group-chevron" />
          </button>
          <Transition name="nav-group"
            ><div
              v-if="openedGroups.has(group.key)"
              :id="`${role}-nav-${group.key}`"
              class="nav-group-items"
            >
              <router-link
                v-for="item in group.items"
                :key="item.to"
                :to="item.to"
                class="nav-link"
                :class="{ active: activeItem?.to === item.to }"
                :aria-current="activeItem?.to === item.to ? 'page' : undefined"
                ><span>{{ item.label }}</span
                ><span v-if="activeItem?.to === item.to" class="nav-active-dot"
              /></router-link></div
          ></Transition>
        </div>
      </nav>
      <div class="sidebar-bottom">
        <div class="sidebar-note">
          <AppIcon name="flame" />
          <p>Todo listo para<br /><strong>un buen servicio.</strong></p>
        </div>
        <div class="sidebar-account">
          <span class="avatar">{{ initials }}</span>
          <div>
            <strong>{{ displayName }}</strong><small>{{ roleLabel }}</small>
          </div>
        </div>
        <button class="btn-logout" @click="logout">
          <AppIcon name="logout" :size="18" />Cerrar sesión
        </button>
      </div>
    </aside>
    <div class="workspace-main" :inert="mobileOpen || undefined">
      <header class="workspace-header">
        <div class="workspace-breadcrumb">
          <button
            ref="menuButton"
            class="icon-button mobile-only"
            aria-label="Abrir menú"
            :aria-expanded="mobileOpen"
            @click="mobileOpen = true"
          >
            <AppIcon name="menu" /></button
          ><span>Mi negocio</span><span class="breadcrumb-slash">/</span
          ><strong>{{
            activeItem?.label || route.meta.title || "Mi espacio"
          }}</strong>
        </div>
        <div class="workspace-header-actions">
          <span class="workspace-role">{{
            workspace.accountLabel
          }}</span><ThemeToggle /><span class="avatar avatar-small">{{
            initials
          }}</span>
        </div>
      </header>
      <main id="main-content" class="content-pane" tabindex="-1">
        <slot
          ><router-view v-slot="{ Component, route }"
            ><Transition name="workspace-route" mode="out-in"
              ><component
                :is="Component"
                :key="route.path" /></Transition></router-view
        ></slot>
      </main>
      <footer class="workspace-footer">
        <span>leguiburger.</span><span>Hecho para el ritmo de tu cocina.</span>
      </footer>
    </div>
  </div>
</template>
