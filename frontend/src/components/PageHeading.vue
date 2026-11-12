<script setup>
defineProps({ title: String, description: String });
</script>
<template>
  <header class="section-header">
    <div class="section-title">
      <p class="eyebrow">Tu negocio, a punto</p>
      <h1>{{ title }}</h1>
      <p v-if="description">{{ description }}</p>
    </div>
    <div class="page-heading-actions"><slot /></div>
  </header>
</template>
