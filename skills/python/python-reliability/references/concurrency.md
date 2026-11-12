# Concurrencia Python

Leer el [contexto del repo](../../../../AGENTS.md). La API combina asyncio con
adaptadores de `urllib` ejecutados mediante `asyncio.to_thread`; el worker
tiene un ciclo síncrono. Partir del patrón del módulo afectado.

- Mantener I/O bloqueante fuera del event loop. Poner timeout en el transporte
  además del límite de espera de la coroutine.
- Cancelar `to_thread` o agotar `wait_for` no interrumpe necesariamente el hilo
  ni demuestra que una operación externa no ocurrió.
- Limitar concurrencia donde se crea el trabajo, considerar threads y colas
  y revisar el alcance por proceso de gates/semaphores.
- La exclusión distribuida depende de leases/RPC, no de un `asyncio.Lock`
  local. Conservar ownership y renovación durante la operación.
- Recoger excepciones de tareas y cerrar recursos. No lanzar tareas sueltas
  para trabajos que deban sobrevivir al proceso; usar el outbox existente.

Probar timeout, cancelación, concurrencia y liberación de recursos con fixtures
locales. No convertir el worker ni todos los clientes a async como efecto
secundario de una optimización. Usar sintaxis compatible con Python 3.11.
