<script setup>
defineProps({
  columns: { type: Number, required: true },
  rows: { type: Number, default: 5 },
});
</script>
<template>
  <tr v-for="row in rows" :key="row" class="loading-row" aria-hidden="true">
    <td v-for="column in columns" :key="column">
      <span
        class="loading-cell"
        :style="{
          width: `${column === 1 ? 72 : 46 + ((row + column) % 3) * 12}%`,
        }"
      ></span>
    </td>
  </tr>
</template>
