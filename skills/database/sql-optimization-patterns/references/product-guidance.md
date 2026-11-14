---
name: sql-optimization-patterns
description: "Analizar rendimiento de consultas, RPC, indices y planes PostgreSQL/Supabase de Al Toque ante lentitud o carga."
---

# SQL y rendimiento

## Alcance y selección

Usar para una hipótesis de lentitud/carga, consulta, índice o plan SQL.
Cubre rendimiento medido; modelo y procedimiento de cambio tienen guías propias.

Leer el [contexto del repo](../../../../AGENTS.md). Localizar SQL en
`apps/web/supabase/migrations` y llamadas en adaptadores Python.

- Identificar filtros por taller/usuario, joins, orden, paginación, cardinalidad
  y frecuencia; conservar autorización, locks e idempotencia al optimizar.
- Verificar índices existentes y evitar proponer duplicados o índices con
  costo de escritura mayor que el beneficio demostrado.
- Pedir/obtener el plan del entorno autorizado si hay acceso. Sin él,
  describir hipótesis basadas en SQL, no mejoras medidas.
- `EXPLAIN ANALYZE` ejecuta la consulta: para escrituras o RPC con efectos
  usar una fixture/base descartable autorizada. Incluso SELECT puede llamar
  funciones con efectos. Preferir `EXPLAIN` para inspección inicial.
- No ejecutar mantenimiento con locks ni aplicar SQL para una revisión.
  Diseñar migración compatible y considerar RLS/grants y consumidores.
- Para dashboard/informes, distinguir muestras y paginación de agregados
  exactos. Si se propone una vista/materialización, declarar frescura,
  invalidación y permisos; no asumir que un agregado en caché conserva RLS.

Esta guía cubre rendimiento. El modelo de tablas corresponde a
`postgresql-table-design`; la evolución y aplicación del SQL, a
`database-migrations`. No cargar esos procedimientos para una lectura de plan.

Usar `tools/migrations.py --history ruta/historial.json` desde la raíz para
comparar historial exportado; sin historial solo inventaría/parsea archivos.
El directorio no es un esquema inicial ni todo su contenido está pendiente.
Validar planes, concurrencia y aislamiento en staging según
`docs/operacion/despliegue.md` antes de recomendar el cambio operativo.
