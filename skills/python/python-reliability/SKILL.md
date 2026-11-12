---
name: python-reliability
description: "Modificar ejecucion Python con I/O, concurrencia, reintentos, recursos o jobs outbox de Al Toque. Seleccionar solo el modo afectado."
---

# Ejecución fiable Python

## Alcance y selección

- Usar cuando cambia I/O, timeout/cancelación, retry, manejo de recursos,
  fallos parciales o recuperación de un job durable.
- Cubre garantías de ejecución; las convenciones del código están en
  `python-guidelines` y las pruebas en `python-testing-patterns`.
- Una edición de estilo o DTO no requiere leer todos estos procedimientos.

Leer el [contexto del repo](../../../AGENTS.md) y elegir el modo del cambio:

| Cambio | Referencia |
| --- | --- |
| Event loop, threads, cancelación o límites de concurrencia | [Concurrencia](references/concurrency.md) |
| Timeout/reintento de proveedor o resultado externo incierto | [Reintentos](references/retries.md) |
| Outbox, claim, lease, ACK y recuperación del worker | [Jobs durables](references/jobs.md) |
| Respuestas HTTP, archivos, streaming, imágenes o navegador | [Recursos](references/resources.md) |
| Validación, excepciones o mezcla de éxitos/fallos en un lote | [Fallos parciales](references/failures.md) |

Los modos se complementan solo si el flujo los cruza. El outbox usa PostgreSQL
y el worker existente; no implica adoptar un broker ni otro scheduler.
Un timeout/cancelación no demuestra que un efecto externo no ocurrió.
Conservar ownership, journals y conciliación antes de repetir efectos.

Probar los fallos relevantes con las fixtures actuales. La recuperación real
de pagos, emisión o mensajes requiere ambiente y alcance autorizados.

## Especialidades originales completas

Esta guía integra el contexto de Al Toque; no tiene un original
independiente recortado que deba reemplazarse. Para el modo afectado,
leer el perfil completo correspondiente y el
[contrato común de AI Toque](../../ai-toque-execution-context.md).
Las referencias locales complementan esos perfiles; no los sustituyen.
No cargar todos los modos ni establecer convenciones competidoras.

| Decisión | Perfil completo |
| --- | --- |
| Event loop, concurrencia y cancelación | [async-python-patterns](../async-python-patterns/SKILL.md) |
| Retries y resiliencia | [python-resilience](../python-resilience/SKILL.md) |
| Ownership, context managers y recursos | [python-resource-management](../python-resource-management/SKILL.md) |
| Jobs, colas y recuperación | [python-background-jobs](../python-background-jobs/SKILL.md) |
| Excepciones y fallos parciales | [python-error-handling](../python-error-handling/SKILL.md) |
| Errores entre lenguajes/procesos | [error-handling-patterns](../../engineering/error-handling-patterns/SKILL.md) |
