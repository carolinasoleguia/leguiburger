# Trabajos durables

Leer el [contexto del repo](../../../../AGENTS.md), `apps/worker/worker/main.py`,
`apps/worker/worker/infrastructure/outbox.py`,
`apps/api/app/infrastructure/supabase_outbox.py` y el handler interno afectado.

- Usar el outbox PostgreSQL existente. Seguir creación/deduplicación, claim,
  dispatch, ejecución en API y complete/retry/defer como un solo contrato.
- Validar event_id y worker_id contra el claim persistido; no aceptar actor,
  teléfono, factura o pago arbitrario del payload interno.
- Conservar heartbeat, fingerprint, expiración/renovación de lease y ownership.
  No confirmar finalización si se perdió el claim.
- Tratar reentregas como normales: la persistencia local no garantiza entrega
  exactamente una vez al proveedor. Antes de repetir un efecto incierto,
  consultar journals/estado y seguir la conciliación del dominio.
- Mantener intentos y estados DEAD recuperables. Un fallo de infraestructura
  no equivale a un trabajo realizado ni a permiso para purgarlo.
- Cambios de evento/RPC requieren compatibilidad entre SQL, API y worker.
  No añadir otro broker, scheduler o consumidor sobre los mismos eventos.

Probar caída después del efecto y antes del ACK, claim perdido, duplicados,
reintento, payload inválido y worker obsoleto. La recuperación real y el orden
de corte se validan en staging según `docs/operacion/despliegue.md`.
