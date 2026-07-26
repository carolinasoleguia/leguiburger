<template>
  <section class="owner-home">
    <div class="welcome-card">
      <h1>Bienvenido a Leguiburger</h1>
      <p>Como dueño de la marca podés:</p>
      <ul>
        <li>Gestionar marcas y su configuración.</li>
        <li>Administrar locales y su presencia.</li>
        <li>Controlar los administradores con acceso al sistema.</li>
      </ul>
      <p>Usá el menú lateral para acceder a cada sección.</p>
    </div>
  </section>
</template>

<script setup>
</script>

<style scoped>
.owner-home {
  padding: 1.5rem;
}
.welcome-card {
  width: min(100%, 760px);
  padding: 2rem;
  border-radius: 18px;
  background: var(--surface-soft);
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow: 0 16px 32px rgba(0, 0, 0, 0.18);
}
.welcome-card h1 {
  margin-bottom: 1rem;
}
.welcome-card p {
  margin-bottom: 1rem;
  line-height: 1.75;
  color: var(--text);
}
.welcome-card ul {
  margin: 0 0 1rem 1.25rem;
  color: var(--text);
}
.welcome-card li {
  margin-bottom: 0.65rem;
}
</style>
