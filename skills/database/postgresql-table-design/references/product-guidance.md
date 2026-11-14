---
name: postgresql-table-design
description: "Disenar o revisar el modelo PostgreSQL de Al Toque: tablas, tipos, claves, restricciones y pertenencia al taller."
---

# Modelo PostgreSQL

## Alcance y selección

Usar para decidir identidad, tipos, relaciones y restricciones de tablas.
Cubre modelo de datos; rendimiento y aplicación de SQL tienen guías propias.

Leer el [contexto del repo](../../../../AGENTS.md) y el
[deepdive de base de datos](../../../../docs/deepdive-base-de-datos.md) cuando el
dominio esté allí analizado. Esta guía cubre decisiones de modelo; rendimiento
y procedimiento de migración tienen sus propias guías.

- Partir de migraciones, tipos, RPC y consumidores actuales. Distinguir el
  esquema inferido de archivos del esquema aplicado; no suponer una base vacía.
- Definir identidad, actor creador, taller propietario, ciclo de vida y unidad
  de consistencia. Una FK de existencia no prueba que dos filas sean del mismo
  taller ni sustituye autorización/RLS.
- Conservar IDs y convenciones compatibles con consumidores existentes.
  Elegir tipos por semántica: exactitud/escala de importes, moneda, instante
  con zona horaria frente a fecha de negocio y estados realmente distintos.
- Definir NOT NULL, CHECK, FK y unicidad según la regla, incluidos nulos,
  borrado lógico y normalización de teléfono/documento/email. Comprobar que
  defaults no impidan un backfill ni conviertan ausencia en dato verificado.
- Revisar efectos ON DELETE/UPDATE sobre jobs, journals y restauración.
  Conservar evidencia de deduplicación/conciliación al definir retención.
- Mantener JSONB para contenido variable cuyo contrato esté claro; no mover
  identidad, ownership o relaciones centrales a JSON para eludir restricciones.
- Valorar índices por consultas y costo de escritura; no crear uno por cada
  FK ni particionar por un umbral universal. Confirmar versión/extensiones
  del PostgreSQL real antes de usar funciones o sintaxis específicas.
- Declarar RLS/grants y privilegios de RPC/vistas como parte del diseño.
  Un modelo propuesto no demuestra aislamiento ni políticas aplicadas.

Entregar DDL propuesto y compatibilidad con productores/consumidores, sin
ejecutarlo por revisar el modelo. Para llevarlo a un cambio SQL usar
`database-migrations`; para planes/mediciones usar `sql-optimization-patterns`.
