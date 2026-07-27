<template>
  <section class="admin-home">
    <div class="welcome-card">
      <h1>Bienvenido al panel de Administrador</h1>
      <p>Desde aquí puedes gestionar empleados y revisar la producción diaria de tu tenant.</p>
      <ul>
        <li>Revisar y editar registros diarios de producción.</li>
        <li>Crear, actualizar y desactivar empleados.</li>
        <li>Monitorear el rendimiento del equipo en tu local.</li>
      </ul>
      <p>Usa el menú lateral para navegar entre las secciones.</p>
    </div>
  </section>
</template>

<script setup>
</script>

<style scoped>
.admin-home {
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
