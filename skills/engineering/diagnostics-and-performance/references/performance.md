# Rendimiento Python

Leer el [contexto del repo](../../../../AGENTS.md). Definir carga, latencia,
memoria o throughput objetivo antes de cambiar el algoritmo o la concurrencia.

- Separar espera de Supabase/proveedor, CPU, parseo, generación PDF y transporte.
  Un perfil CPU no explica por sí solo el tiempo de una llamada remota.
- Usar `cProfile`, `pstats`, `tracemalloc` o mediciones del sistema disponibles
  con fixtures representativas y sin datos privados.
- Medir tamaños y límites de PDF/imágenes/Excel, picos de memoria y bloqueo
  del event loop. Considerar gates y capacidad por proceso/réplica.
- Conservar límites de payload, autorización y leases al optimizar streaming,
  caching o batches. No cachear respuestas privadas entre talleres.
- Comparar antes/después bajo la misma carga y verificar que no cambia el
  número de efectos, orden fiscal o recuperación ante fallo.

Aplicar una mejora al cuello demostrado y correr regresiones pertinentes.
Documentar condiciones y límites de la medición; no prometer ahorro o escala
de producción a partir de una microprueba o un proveedor simulado.
