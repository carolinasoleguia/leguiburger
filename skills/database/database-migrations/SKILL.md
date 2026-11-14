---
name: database-migrations
description: "Preparar cambios SQL compatibles de Al Toque: historial Supabase, orden de despliegue, backfills y validacion de staging."
---

# Evolución del SQL

## Alcance y selección

Usar para preparar orden, compatibilidad, backfills y recuperación de SQL.
Cubre evolución del esquema; seleccionar esta guía no autoriza aplicar SQL.

Leer el [contexto del repo](../../../AGENTS.md) y
[cierre de migración](../../../docs/operacion/despliegue.md). Esta guía cubre el
procedimiento de cambio; el modelo y los planes SQL se definen por separado.

1. Identificar versiones aplicadas, baseline y ambiente objetivo. Usar el
   historial exportado con `python tools/migrations.py --history ruta/historial.json`.
   Sin historial, el inventario local solo parsea archivos y no informa qué
   está pendiente. Igualdad de versión sin checksum no acredita contenido.
2. Preparar una migración nueva en `database/v2/supabase/migrations`, con nombre
   de 14 dígitos único, siguiendo la secuencia existente. No reescribir SQL
   aplicado ni ejecutar todo el directorio para crear una base inicial.
3. Explicitar compatibilidad entre SQL, API, worker y web: ampliar primero,
   adaptar lectores/escritores y retirar lo anterior solo después del corte
   validado. No activar dos procesadores sobre los mismos efectos.
4. Para backfills, definir filas candidatas, lotes, reanudación, invariantes,
   locks y validación. Defaults/nulos deben permitir identificar pendientes.
   No declarar rollback completo si cambió información o hubo efectos externos.
5. Evaluar duración/locks, permisos y restricciones de transacción antes de
   elegir DDL. La sintaxis parseada no demuestra que el cambio sea online ni
   compatible con versión/extensiones, grants o datos efectivos.
6. Incluir backup/restauración ensayados, verificaciones de RLS/concurrencia,
   criterios de abortar y procedimiento de recuperación con claims/journals.
   Respetar gates de release y fingerprint.

Validar sintaxis/historial con herramientas locales y las pruebas pertinentes.
Aplicación de SQL, backfills y pruebas de proveedores dependen del alcance
autorizado y del ambiente identificado. No añadir otro sistema de migraciones
ni sustituir el flujo Supabase para preparar un cambio.

## Especialidades originales completas

Esta guía integra el contexto de Al Toque; no tiene un original
independiente recortado que deba reemplazarse. Para el modo afectado,
leer el perfil completo correspondiente y el
[contrato común de AI Toque](../../ai-toque-execution-context.md).
Las referencias locales complementan esos perfiles; no los sustituyen.
No cargar todos los modos ni establecer convenciones competidoras.

| Decisión | Perfil completo |
| --- | --- |
| Preparación de migraciones SQL | [sql-migrations](../sql-migrations/SKILL.md) |
| Observabilidad y seguimiento de migraciones | [migration-observability](../migration-observability/SKILL.md) |
| Administración, backup y recuperación | [database-admin](../database-admin/SKILL.md) |
| Optimización y ejecución online | [database-migrations-database-optimizer](../database-migrations-database-optimizer/SKILL.md) |
