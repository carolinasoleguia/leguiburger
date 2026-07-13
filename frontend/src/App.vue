<template>
  <div class="container">
    <h1>🍔 LeguiBurger App (Vue + Vite)</h1>
    <p class="subtitle">Bienvenido al portfolio de desarrollo en equipo.</p>
    
    <hr class="divider" />

    <div class="card">
      <h3>Respuesta del Servidor de Go:</h3>
      <p class="backend-msg">{{ mensajeBackend }}</p>
      <button @click="llamarAlBackend" class="btn">
        Saludar a Go
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

// Definimos la variable reactiva
const mensajeBackend = ref('Hacé clic para saludar al backend...')

// Función para pegarle al endpoint del monolito de Go
const llamarAlBackend = async () => {
  try {
    const respuesta = await fetch('/api/hello')
    const data = await respuesta.json()
    mensajeBackend.value = data.message
  } catch (error) {
    mensajeBackend.value = 'Error al conectar con Go: ' + error.message
  }
}
</script>

<style scoped>
.container {
  font-family: sans-serif;
  text-align: center;
  padding: 50px;
  background-color: #1a1a1a;
  color: #fff;
  min-height: 100vh;
}
.subtitle {
  color: #aaa;
}
.divider {
  border-color: #333;
  margin: 30px 0;
}
.card {
  background-color: #2a2a2a;
  padding: 20px;
  border-radius: 8px;
  display: inline-block;
}
.backend-msg {
  color: #4caf50;
  font-weight: bold;
}
.btn {
  background-color: #ff9800;
  color: white;
  border: none;
  padding: 10px 20px;
  font-size: 16px;
  border-radius: 5px;
  cursor: pointer;
  font-weight: bold;
}
.btn:hover {
  background-color: #e68a00;
}
</style>