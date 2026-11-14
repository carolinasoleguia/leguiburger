# Amenazas y requisitos

Leer el [contexto del repo](../../../../AGENTS.md) y el documento del dominio
relevante. Elegir un flujo, actores y activos reales antes de modelar riesgos.

- Trazar browser, Next, API, worker, Supabase y proveedores con sus fronteras:
  quién aporta identidad, quién decide permiso y dónde se ejecuta el efecto.
- Identificar credenciales, datos privados, membresía/taller, presupuestos,
  jobs y efectos fiscales/financieros del alcance solicitado.
- Revisar STRIDE según el flujo: suplantación (JWT/firmas/claims), alteración
  (IDs/body/estado), repudio (correlación persistida), divulgación (RLS/Storage/
  enlaces/modelo), denegación (tamaños/concurrencia/backlog) y elevación
  (service role/RPC/rol y objeto comprobados por el servidor).
- Incluir replay, resultado incierto y prompt injection solo si la entrada
  no confiable puede influir en decisiones o herramientas reales.
- Si un árbol aclara caminos de ataque, separar condiciones necesarias AND
  de alternativas OR y señalar controles que interrumpen cada camino.
  No inventar acceso a infraestructura ni probabilidades sin evidencia.
- Traducir riesgos en condiciones verificables: otro taller denegado, firma
  inválida rechazada, duplicado sin doble efecto, secreto fuera del bundle,
  archivo excesivo rechazado o resultado incierto conciliable.
- Asociar requisito a componente y test/evidencia de staging, distinguiendo
  controles existentes, propuestos y supuestos del esquema/despliegue real.
- Considerar minimización de datos de logs/modelo y revocación/retención
  según el dominio; no asumir requisitos regulatorios sin establecer alcance.

Entregar tabla o diagrama útil para ese flujo, prioridades y pendientes de
validación. No reproducir ataques ni ejecutar cambios remotos para completar
un documento de amenazas.
