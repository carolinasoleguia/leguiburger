<script setup>
import { onMounted, ref } from "vue";
import BrandLogo from "../../components/BrandLogo.vue";
import ThemeToggle from "../../components/ThemeToggle.vue";
import { shopperRequest } from "../../composables/useShopper.js";
const items = ref([]),
  loading = ref(true),
  error = ref("");
async function load() {
  loading.value = true;
  error.value = "";
  try {
    items.value = (await shopperRequest("/api/public/restaurants")).items || [];
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
onMounted(load);
</script>
<template>
  <main class="restaurants-public">
    <header>
      <router-link to="/"><BrandLogo /></router-link>
      <div>
        <router-link to="/mi-cuenta" class="text-link">Mi cuenta</router-link
        ><ThemeToggle />
      </div>
    </header>
    <section>
      <p class="eyebrow">El antojo empieza acá</p>
      <h1>Elegí tu próxima favorita.</h1>
      <p>
        Explorá las cartas de los locales. Podés pedir con tu cuenta o sin
        registrarte.
      </p>
      <p v-if="loading" role="status">Cargando restaurantes…</p>
      <p v-if="error" role="alert">
        {{ error }}
        <button class="btn-secondary" @click="load">Reintentar</button>
      </p>
      <p v-if="!loading && !error && !items.length" class="card">
        Todavía no hay cartas publicadas.
      </p>
      <div class="restaurant-grid">
        <article v-for="local in items" :key="local.id" class="card">
          <small>{{ local.location }}</small>
          <h2>{{ local.business }}</h2>
          <p>
            {{
              local.address || "Consultá los datos de la sucursal en su carta."
            }}
          </p>
          <router-link class="btn-primary" :to="`/catalogo/${local.id}`"
            >Ver carta</router-link
          >
        </article>
      </div>
    </section>
  </main>
</template>
<style scoped>
.restaurants-public {
  min-height: 100dvh;
  background: var(--bg);
  color: var(--text);
}
header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px max(20px, calc((100% - 1200px) / 2));
  border-bottom: 1px solid var(--border);
}
header > div {
  display: flex;
  gap: 20px;
  align-items: center;
}
section {
  max-width: 1240px;
  margin: auto;
  padding: 56px 20px;
}
h1 {
  font-size: clamp(36px, 5vw, 56px);
  letter-spacing: -0.04em;
}
.restaurant-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 24px;
  margin-top: 36px;
}
.restaurant-grid article {
  padding: 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.restaurant-grid h2 {
  margin: 0;
}
.restaurant-grid p,
small {
  color: var(--text-muted);
  line-height: 1.6;
}
.restaurant-grid a {
  margin-top: auto;
}
@media (max-width: 850px) {
  .restaurant-grid {
    grid-template-columns: 1fr;
  }
}
</style>
