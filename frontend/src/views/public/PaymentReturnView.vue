<script setup>
import BrandLogo from '../../components/BrandLogo.vue';
</script>
<template><main class="payment-return"><BrandLogo /><section class="card"><h1>Gracias por completar el checkout.</h1><p>Estamos verificando el resultado con Mercado Pago. El regreso a esta página no confirma por sí solo que el pago se haya acreditado.</p><p>El estado se actualizará en el negocio cuando termine la verificación.</p><router-link class="btn-primary" to="/payments">Consultar mis cobros</router-link><router-link to="/">Volver al inicio</router-link></section></main></template>
<style scoped>.payment-return { max-width: 580px; margin: 70px auto; padding: 24px; }.card { margin-top: 30px; padding: 32px; }h1 { font-size: 26px; }p { margin: 20px 0; }a { display: inline-block; margin: 12px 16px 0 0; }</style>
