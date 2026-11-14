<script setup>
defineProps({
  offset: Number,
  count: Number,
  total: { type: Number, default: null },
  size: { type: Number, default: 25 },
  disabled: Boolean,
});
defineEmits(["change"]);
</script>
<template>
  <nav class="list-pager" aria-label="Páginas de resultados">
    <button
      class="btn-secondary"
      :disabled="disabled || offset === 0"
      @click="$emit('change', Math.max(0, offset - size))"
    >
      Anterior</button
    ><span>Página {{ Math.floor(offset / size) + 1 }}</span
    ><button
      class="btn-secondary"
      :disabled="
        disabled ||
        count < size ||
        (total !== null && offset + size >= total) ||
        offset + size > 10000
      "
      @click="$emit('change', offset + size)"
    >
      Siguiente
    </button>
  </nav>
</template>
