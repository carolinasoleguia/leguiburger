# Observabilidad de API y worker

Leer el [contexto del repo](../../../../AGENTS.md). Revisar logging existente,
`apps/api/app/core/ops_alerts.py`, adaptadores de alertas y readiness.

- Correlacionar request/event_id, worker_id, operación, intento y resultado.
  No usar teléfono, JWT, payload completo o URL privada como correlación.
- Mantener límites y sanitización en logs/alertas, incluidos errores que
  vienen de proveedores. Un texto de excepción puede contener secretos.
- Distinguir recepción, persistencia, claim, efecto confirmado, conciliación
  y complete/retry. Un HTTP exitoso no prueba que todo el trabajo terminó.
- Para métricas, controlar cardinalidad; no crear labels por persona/evento.
  Medir backlog, DEAD, latencia, retries y gates con el sistema disponible.
- Readiness debe conservar comprobación privada de grants, heartbeat reciente
  y fingerprint. No rebajarla a un health público genérico.
- Reutilizar logging y transportes actuales. Añadir tracing/SDK solo cuando
  forme parte del alcance y esté configurado en el servicio adecuado.
- Para detección de abuso solicitada, definir evento/dato necesario, ventana/
  umbral y caso legítimo. Probar positivos, falsos positivos y ausencia de fuente;
  no inferir intrusión de un 403 ni instalar SIEM/EDR por una guía importada.
  Una alerta no autoriza bloquear cuentas, rotar secretos o enviar avisos.

Probar campos de diagnóstico, redacción de datos y fallos del envío de alertas.
Reportar qué se observó localmente y qué requiere logs del entorno conectado.
