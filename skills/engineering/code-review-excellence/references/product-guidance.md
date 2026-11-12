---
name: code-review-excellence
description: "Revisar PRs o diffs de Al Toque por regresiones, contratos entre procesos y riesgos de integraciones."
---

# Revisión de cambios

## Alcance y selección

Usar al revisar un PR/diff o priorizar deuda/refactors a partir de evidencia.
Cubre evaluación del cambio; no redefine convenciones Python ni inicia una
reescritura, auditoría o cadena de skills para cada revisión.

Leer el [contexto del repo](../../../../AGENTS.md). Determinar base y alcance del
diff, instrucciones locales y consumidores antes de evaluar el cambio.

- Seguir cambios entre rutas Next, API, worker, contratos y migraciones.
  Comprobar que la web siga delegando reglas de negocio a Python.
- Para cobros/emisión, revisar idempotencia, journals y respuesta incierta;
  para jobs, ownership del claim, lease y confirmación de finalización.
- Para datos, comprobar identidad, permisos del taller, grants y autorización
  antes del acceso con service role.
- Si el diff modifica una frontera sensible (entrada externa, permiso, archivo,
  proveedor, dato privado o IA/tools), usar el modo pertinente de `security-review`
  para esa decisión. No duplicar sus checklists ni exigir un scanner para cada PR.
- Valorar tests por comportamiento y fallos que detectan. No exigir una nueva
  abstracción, framework o porcentaje de cobertura por preferencia.
- Un hallazgo debe señalar archivo, escenario que lo activa, consecuencia
  y corrección posible. Separar una duda de un defecto demostrado.
- Para deuda técnica, priorizar por riesgo/frecuencia de cambio y proponer
  una unidad abordable con consumidores y regresiones identificados. Separar
  un refactor que preserva comportamiento de un cambio funcional del mismo flujo.
- Mantener contratos y reglas mientras se extraen módulos; no usar una revisión
  como motivo para reescribir componentes que no intervienen en el problema.

Priorizar problemas que bloqueen el uso esperado. Informar checks ejecutados
y límites de la evidencia. Una revisión no autoriza publicar comentarios,
aplicar SQL, desplegar ni modificar cuentas externas por su cuenta.
