<template>
  <div class="app-container">
    <header class="hero">
      <div class="hero-icon">🍔</div>
      <h1 class="brand-title">LEGUIBURGERRRRRRRRRRRRRRRR</h1>
      <p class="hero-subtitle">Production Cost Control & Multi-Tenant E-Commerce Platform</p>
      <div class="author-tag">Developed by the Eguia Brothers</div>
    </header>

    <hr class="section-divider" />

    <section class="about-card">
      <h2>About the Project</h2>
      <p>
        LeguiBurger is a high-performance, minimalist SaaS platform designed specifically for the gastronomic and e-commerce sectors. 
        It solves the complex problem of cost management under fluctuating wholesale prices by automatically calculating strict recipe margins, 
        tracking raw supply availability, and serving multi-brand analytics in real-time.
      </p>
      
      <div class="features-grid">
        <div class="feature-item">
          <div class="feature-emoji">🏢</div>
          <h4>Multi-Tenant Architecture</h4>
          <p>Built from the ground up to support multiple corporate clients/brands isolated on a single infrastructure.</p>
        </div>
        <div class="feature-item">
          <div class="feature-emoji">📈</div>
          <h4>Smart Cost Control</h4>
          <p>Tracks raw materials (supplies) and historical ingredient inflation to compute live profit margins per product.</p>
        </div>
        <div class="feature-item">
          <div class="feature-emoji">⚡</div>
          <h4>Modern Tech Stack</h4>
          <p>Powered by a blistering-fast Go backend, an elegant Vue 3 frontend, and a secure Vercel Postgres relational database.</p>
        </div>
      </div>
    </section>

    <section class="sandbox-card">
      <h3>Live API Connection Test</h3>
      <p class="status-box" :class="{ 'has-data': backendMessage !== 'Awaiting click to fetch...' }">
        {{ backendMessage }}
      </p>
      <button @click="fetchFromBackend" class="cta-button">
        Ping Go Backend
      </button>
    </section>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const backendMessage = ref('Awaiting click to fetch...')

const fetchFromBackend = async () => {
  try {
    const response = await fetch('/api/hello')
    const data = await response.json()
    backendMessage.value = data.message
  } catch (error) {
    backendMessage.value = 'Failed to reach Go: ' + error.message
  }
}
</script>

<style scoped>
/* Base Setup (Dark Minimalist) */
.app-container {
  font-family: 'Inter', system-ui, -apple-system, sans-serif;
  background-color: #0d0d0d;
  color: #f5f5f7;
  min-height: 100vh;
  padding: 40px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

/* Hero Section */
.hero {
  text-align: center;
  max-width: 600px;
  margin-top: 20px;
}

.hero-icon {
  font-size: 4rem;
  margin-bottom: 10px;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

.brand-title {
  font-size: 2.5rem;
  font-weight: 800;
  letter-spacing: -0.05em;
  margin: 0;
  color: #ffffff;
}

.hero-subtitle {
  font-size: 1.1rem;
  color: #8e8e93;
  margin: 10px 0 20px 0;
  line-height: 1.4;
}

.author-tag {
  display: inline-block;
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  background-color: #1c1c1e;
  color: #ff9f0a;
  padding: 6px 16px;
  border-radius: 20px;
  font-weight: 600;
}

.section-divider {
  width: 100%;
  max-width: 800px;
  border: 0;
  height: 1px;
  background: linear-gradient(to right, transparent, #2c2c2e, transparent);
  margin: 40px 0;
}

/* About Card & Features Grid */
.about-card {
  background-color: #1c1c1e;
  border: 1px solid #2c2c2e;
  border-radius: 16px;
  padding: 35px;
  max-width: 800px;
  text-align: left;
  box-shadow: 0 4px 30px rgba(0, 0, 0, 0.5);
}

.about-card h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-top: 0;
  color: #ffffff;
}

.about-card p {
  color: #aeaeb2;
  line-height: 1.6;
  font-size: 1rem;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 20px;
  margin-top: 30px;
}

.feature-item {
  background-color: #2c2c2e;
  padding: 20px;
  border-radius: 12px;
  border: 1px solid #3a3a3c;
}

.feature-emoji {
  font-size: 1.5rem;
  margin-bottom: 8px;
}

.feature-item h4 {
  margin: 0 0 8px 0;
  color: #ffffff;
  font-size: 1.05rem;
}

.feature-item p {
  margin: 0;
  font-size: 0.85rem;
  color: #aeaeb2;
  line-height: 1.4;
}

/* Live API Sandbox */
.sandbox-card {
  margin-top: 40px;
  text-align: center;
  max-width: 400px;
  width: 100%;
}

.sandbox-card h3 {
  font-size: 1.1rem;
  font-weight: 600;
  color: #8e8e93;
  margin-bottom: 15px;
}

.status-box {
  background-color: #1c1c1e;
  border: 1px solid #2c2c2e;
  padding: 14px;
  border-radius: 8px;
  font-family: monospace;
  font-size: 0.9rem;
  color: #aeaeb2;
  margin-bottom: 15px;
  transition: all 0.3s ease;
}

.status-box.has-data {
  color: #30d158;
  border-color: rgba(48, 209, 88, 0.3);
  background-color: rgba(48, 209, 88, 0.05);
}

.cta-button {
  background-color: #ff9f0a;
  color: #ffffff;
  border: none;
  padding: 12px 24px;
  font-size: 0.95rem;
  font-weight: 600;
  border-radius: 8px;
  cursor: pointer;
  transition: transform 0.2s, background-color 0.2s;
}

.cta-button:hover {
  background-color: #ffb340;
  transform: translateY(-1px);
}

.cta-button:active {
  transform: translateY(1px);
}
</style>