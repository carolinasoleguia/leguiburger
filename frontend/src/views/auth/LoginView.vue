<template>
  <div class="auth-page">
    <aside class="auth-story"><BrandLogo /><div class="auth-story-copy"><p class="eyebrow">BIENVENIDO A TU NEGOCIO</p><h1>El sabor es tuyo.<br /><em>El orden,<br />también.</em></h1><p>Tu marca, tus locales y tu equipo. Todo listo para una buena jornada.</p><div class="auth-story-list"><span><AppIcon name="store" :size="18" />Cada local, conectado.</span><span><AppIcon name="burger" :size="18" />Tu carta, siempre a punto.</span><span><AppIcon name="flame" :size="18" />La cocina, en orden.</span></div></div><p class="auth-story-foot">LeguiBurger · Hecho para el ritmo de tu cocina.</p></aside>
    <main class="auth-form-side"><div class="auth-top"><router-link to="/" class="auth-back"><AppIcon name="arrow" :size="16" />Volver al inicio</router-link><ThemeToggle /></div><div class="login-card"><div class="auth-mobile-brand"><BrandLogo /></div><div class="brand-header"><p class="eyebrow">TU ESPACIO DE TRABAJO</p><h2>Qué bueno verte.</h2><p>Ingresá con tu cuenta para seguir con tu negocio.</p></div>
      <form @submit.prevent="handleLogin"><div class="input-group"><label for="email">Correo electrónico</label><input id="email" type="email" v-model="email" required autocomplete="username" placeholder="vos@tuhamburgueseria.com" :disabled="loading" /></div><div class="input-group"><label for="password">Contraseña</label><div class="password-field"><input id="password" :type="showPassword ? 'text' : 'password'" v-model="password" required autocomplete="current-password" placeholder="Tu contraseña" :disabled="loading" /><button type="button" class="icon-button" @click="showPassword = !showPassword" :aria-label="showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'" :aria-pressed="showPassword"><AppIcon name="eye" :size="18" /></button></div></div><p v-if="errorMessage" class="error-msg" role="alert">{{ errorMessage }}</p><button type="submit" class="btn-submit" :disabled="loading"><span v-if="loading" class="spinner" />{{ loading ? 'Ingresando…' : 'Ingresar a mi negocio' }}<AppIcon v-if="!loading" name="arrow" :size="17" /></button></form>
    </div><p class="auth-form-foot">Una buena gestión también es un ingrediente.</p></main>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import BrandLogo from '../../components/BrandLogo.vue';
import ThemeToggle from '../../components/ThemeToggle.vue';
import AppIcon from '../../components/AppIcon.vue';
const showPassword = ref(false);
import { useRouter } from 'vue-router';
import { useAuth } from '../../composables/useAuth.js';

const auth = useAuth();
const router = useRouter();
const email = ref('');
const password = ref('');
const loading = ref(false);
const errorMessage = ref('');

async function handleLogin() {
  errorMessage.value = '';
  loading.value = true;

  try {
    await auth.login({ email: email.value, password: password.value });
    navigateToRole();
  } catch (error) {
    errorMessage.value = error.message || 'Error al iniciar sesión';
  } finally {
    loading.value = false;
  }
}

function navigateToRole() {
  const role = auth.userRole.value;
  if (role === 'owner') {
    router.push('/owner');
    return;
  }
  if (role === 'admin') {
    router.push('/admin');
    return;
  }
  if (role === 'employee') {
    const tenantId = auth.tenantId.value;
    if (tenantId) {
      router.push(`/tenant/${tenantId}/employee`);
      return;
    }
    router.push('/unauthorized');
    return;
  }
  router.push('/unauthorized');
}

</script>
