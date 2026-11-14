<script setup>
import AppIcon from './AppIcon.vue';
defineProps({ compact: Boolean });
</script>
<template><router-link to="/" class="brand-lockup" aria-label="LeguiBurger, inicio"><span class="brand-mark"><AppIcon name="burger" :size="25" /></span><span v-if="!compact" class="brand-word">legui<span>burger</span><span class="brand-dot">.</span></span></router-link></template>
