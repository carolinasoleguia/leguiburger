<script setup>
import { useTheme } from '../composables/useTheme.js';
import AppIcon from './AppIcon.vue';
const { theme, toggleTheme } = useTheme();
</script>
<template><button type="button" class="icon-button theme-toggle" @click="toggleTheme" :aria-label="theme === 'dark' ? 'Activar tema claro' : 'Activar tema oscuro'" :title="theme === 'dark' ? 'Tema claro' : 'Tema oscuro'"><AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" /></button></template>
