<script setup>
defineProps({ name: { type: String, default: 'grid' }, size: { type: [Number, String], default: 20 } });
const paths = {
  play: 'M8 5l11 7-11 7V5z',
  pause: 'M8 5v14 M16 5v14',
  mic: 'M9 5a3 3 0 0 1 6 0v7a3 3 0 0 1-6 0V5 M5 10v2a7 7 0 0 0 14 0v-2 M12 19v3 M8 22h8',
  file: 'M14 2H5v20h14V7l-5-5z M14 2v6h5 M8 12h8 M8 16h6',
  send: 'M22 2L9 15 M22 2l-7 20-5-9-9-5 21-6z',
  shield: 'M12 2l9 4v6c0 5-9 10-9 10S3 17 3 12V6l9-4z M8 12l3 3 5-6',
  zap: 'M13 2L3 14h8l-1 8L21 9h-8l1-7z',
  grid: 'M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z',
  burger: 'M4 9a8 6 0 0 1 16 0H4z M3 13h18 M4 17h16v1a3 3 0 0 1-3 3H7a3 3 0 0 1-3-3z M9 6h.01 M14 5h.01',
  store: 'M3 10l2-7h14l2 7 M3 10a3 3 0 0 0 6 0 3 3 0 0 0 6 0 3 3 0 0 0 6 0 M5 13v8h14v-8 M10 21v-6h4v6',
  users: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2 M9 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M17 4a4 4 0 0 1 0 8 M22 21v-2a4 4 0 0 0-3-3.87',
  box: 'M12 3l9 5v9l-9 5-9-5V8l9-5z M3 8l9 5 9-5 M12 13v9 M7.5 5.5l9 5',
  flame: 'M12 2c3 5-1 6 2 9 1-1 2-3 2-4 4 4 5 7 3 11a8 8 0 0 1-14-1C3 12 7 8 8 7c-1 5 2 6 3 6 2-3-1-6 1-11z',
  arrow: 'M5 12h14 M13 6l6 6-6 6',
  logout: 'M9 21H4V3h5 M9 12h12 M17 8l4 4-4 4',
  sun: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M12 2v2 M12 20v2 M2 12h2 M20 12h2 M5 5l1.5 1.5 M17.5 17.5L19 19 M5 19l1.5-1.5 M17.5 6.5L19 5',
  moon: 'M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8z',
  menu: 'M4 6h16 M4 12h16 M4 18h16',
  close: 'M6 6l12 12 M18 6L6 18',
  check: 'M5 12l4 4L19 6',
  chevron: 'M6 9l6 6 6-6',
  clock: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 7v5l3 2',
  eye: 'M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6',
};
</script>
<template><svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="paths[name] || paths.grid" /></svg></template>
