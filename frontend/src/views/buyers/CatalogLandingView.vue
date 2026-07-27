<template>
  <section class="catalog-landing">
    <div class="card">
      <h1>Próximamente</h1>
      <p>Catálogo de hamburguesas</p>
      <p>Esta página está disponible sin inicio de sesión.</p>
    </div>
  </section>
</template>

<script setup>
</script>

<style scoped>
.catalog-landing {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 70vh;
}
.card {
  max-width: 480px;
  padding: 2rem;
  text-align: center;
  border-radius: 12px;
  box-shadow: 0 18px 40px rgba(0, 0, 0, 0.08);
  background: #fff;
}
.card h1 {
  margin-bottom: 1rem;
  font-size: 2rem;
}
.card p {
  color: #444;
  font-size: 1.1rem;
  margin: 0.75rem 0;
}
</style>
