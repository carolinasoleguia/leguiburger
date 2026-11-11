<script setup>
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import BrandLogo from '../../components/BrandLogo.vue';
import ThemeToggle from '../../components/ThemeToggle.vue';
import AppIcon from '../../components/AppIcon.vue';
import { useAuth } from '../../composables/useAuth.js';
import { authRequest } from '../../services/session.js';
const router = useRouter(), route = useRoute(), auth = useAuth();
const email = ref(''), password = ref(''), showPassword = ref(false), loading = ref(false), errorMessage = ref('');
const challenge = ref(''), secret = ref(''), code = ref(''), recovery = ref(false), recoveryCodes = ref([]);
const notice = route.query.reason === 'idle' ? 'Cerramos tu sesión por inactividad.' : route.query.reason === 'account-closed' ? 'Tu cuenta fue desactivada y tus sesiones fueron cerradas.' : route.query.reason === 'expired' ? 'Tu sesión venció. Ingresá nuevamente.' : '';
function navigate() {
  const role = auth.userRole.value;
  router.push(['owner', 'super_admin'].includes(role) ? '/owner' : role === 'admin' ? '/admin' : `/tenant/${auth.tenantId.value}/employee`);
}
async function submit() {
  errorMessage.value = ''; loading.value = true;
  try {
    if (!challenge.value) {
      const payload = await auth.login({ email: email.value, password: password.value });
      password.value = '';
      if (!payload.mfa_required) { navigate(); return; }
      challenge.value = payload.challenge;
      if (payload.enrollment_required) secret.value = (await authRequest('mfa/enroll', {}, challenge.value)).secret;
    } else {
      const payload = await authRequest('mfa/verify', recovery.value ? { recovery_code: code.value } : { code: code.value }, challenge.value);
      auth.completeMFA(payload); code.value = ''; secret.value = ''; challenge.value = '';
      if (payload.recovery_codes) recoveryCodes.value = payload.recovery_codes; else navigate();
    }
  } catch (error) { errorMessage.value = error.message; }
  finally { loading.value = false; }
}
function downloadCodes() {
  const blob = new Blob([`LeguiBurger — códigos de recuperación\nCada código funciona una sola vez. Guardalos en un lugar seguro.\n\n${recoveryCodes.value.join('\n')}`], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob), link = document.createElement('a'); link.href = url; link.download = 'leguiburger-recuperacion.txt'; link.click(); URL.revokeObjectURL(url);
}
</script>

<template>
  <div class="auth-page">
    <aside class="auth-story"><BrandLogo /><div class="auth-story-copy"><p class="eyebrow">BIENVENIDO A TU NEGOCIO</p><h1>El sabor es tuyo.<br /><em>El orden,<br />también.</em></h1><p>Tu marca, tus locales y tu equipo. Todo listo para una buena jornada.</p><div class="auth-story-list"><span><AppIcon name="store" :size="18" />Cada local, conectado.</span><span><AppIcon name="burger" :size="18" />Tu carta, siempre a punto.</span><span><AppIcon name="flame" :size="18" />La cocina, en orden.</span></div></div><p class="auth-story-foot">LeguiBurger · Hecho para el ritmo de tu cocina.</p></aside>
    <main class="auth-form-side"><div class="auth-top"><router-link to="/" class="auth-back">Volver al inicio</router-link><ThemeToggle /></div>
      <div class="login-card"><div class="auth-mobile-brand"><BrandLogo /></div>
        <template v-if="recoveryCodes.length"><h2>Guardá tus códigos.</h2><p>Te permiten entrar si perdés el autenticador. Cada uno funciona una sola vez y se muestran únicamente ahora.</p><pre class="recovery-codes">{{ recoveryCodes.join('\n') }}</pre><button class="btn-secondary" @click="downloadCodes">Descargar códigos</button><button class="btn-submit" @click="recoveryCodes = []; navigate()">Ya los guardé, continuar</button></template>
        <template v-else>
          <div class="brand-header"><p class="eyebrow">TU ESPACIO DE TRABAJO</p><h2>{{ challenge ? 'Protegé tu acceso.' : 'Qué bueno verte.' }}</h2><p>{{ challenge ? 'La administración de la app requiere un segundo factor.' : 'Ingresá con tu cuenta para seguir con tu negocio.' }}</p></div>
          <p v-if="notice && !challenge" role="status">{{ notice }}</p>
          <form v-form-care @submit.prevent="submit">
            <template v-if="!challenge"><div class="input-group"><label for="email">Correo electrónico</label><input id="email" v-model="email" type="email" required autocomplete="username" :disabled="loading" /></div><div class="input-group"><label for="password">Contraseña</label><div class="password-field"><input id="password" v-model="password" :type="showPassword ? 'text' : 'password'" required autocomplete="current-password" :disabled="loading" /><button type="button" class="icon-button" @click="showPassword = !showPassword" :aria-label="showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'"><AppIcon name="eye" :size="18" /></button></div></div><router-link to="/forgot-password">Olvidé mi contraseña</router-link></template>
            <template v-else><div v-if="secret" class="mfa-setup"><p>Agregá una cuenta en tu aplicación de autenticación e ingresá esta clave:</p><code>{{ secret }}</code><p>Elegí códigos por tiempo, de 6 dígitos. Después ingresá el código que muestra tu aplicación.</p></div><div class="input-group"><label for="mfa-code">{{ recovery ? 'Código de recuperación' : 'Código del autenticador' }}</label><input id="mfa-code" v-model="code" :inputmode="recovery ? 'text' : 'numeric'" :maxlength="recovery ? 100 : 6" autocomplete="one-time-code" required :disabled="loading" /></div><button v-if="!secret" class="btn-secondary" type="button" @click="recovery = !recovery; code = ''">{{ recovery ? 'Usar autenticador' : 'Usar un código de recuperación' }}</button></template>
            <p v-if="errorMessage" class="error-msg" role="alert">{{ errorMessage }}</p><button class="btn-submit" :disabled="loading">{{ loading ? 'Verificando…' : challenge ? 'Verificar y continuar' : 'Ingresar a mi negocio' }}</button>
          </form>
        </template>
      </div><p class="auth-form-foot">Una buena gestión también es un ingrediente.</p>
    </main>
  </div>
</template>

<style scoped>
.mfa-setup { margin: 20px 0; padding: 16px; border: 1px solid var(--border); border-radius: 12px; }
.mfa-setup code { display: block; overflow-wrap: anywhere; user-select: all; font-size: 16px; }
.recovery-codes { padding: 16px; overflow: auto; font-size: 13px; }
.btn-submit { margin-top: 20px; }
</style>
