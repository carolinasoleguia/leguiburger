<script setup>
import BrandLogo from '../../components/BrandLogo.vue';
import ThemeToggle from '../../components/ThemeToggle.vue';
import AppIcon from '../../components/AppIcon.vue';
</script>
<template><div><header class="public-header"><BrandLogo /><ThemeToggle /></header><main class="full-screen-message"><div><span class="brand-mark" style="margin:0 auto 24px"><AppIcon name="burger" /></span><p class="eyebrow">LA CARTA DE LA CASA</p><h1>Estamos preparando algo rico.</h1><p class="status-text">El catálogo público estará disponible próximamente.</p><router-link to="/" class="btn-primary">Volver al inicio<AppIcon name="arrow" :size="17" /></router-link></div></main></div></template>
