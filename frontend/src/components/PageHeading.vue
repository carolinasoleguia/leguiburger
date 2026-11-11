<script setup>
defineProps({ title: String, description: String });
</script>
<template>
  <header class="section-header">
    <div class="section-title">
      <p class="eyebrow">TU NEGOCIO, A PUNTO</p>
      <h1>{{ title }}</h1>
      <p v-if="description">{{ description }}</p>
    </div>
    <div class="page-heading-actions"><slot /></div>
  </header>
</template>
