---
name: diagnostics-and-performance
description: "Investigar fallos o rendimiento de Al Toque y mejorar observabilidad. Elegir reproduccion, logs o medicion segun el problema."
---

# Diagnóstico y rendimiento

## Alcance y selección

- Usar para un fallo observado, medición de rendimiento o cambio de
  observabilidad en web, API o worker.
- Cubre evidencia, causa y medición; no impone cambios de arquitectura,
  implementación Python o SQL por el hecho de investigar un problema.

Leer el [contexto del repo](../../../AGENTS.md), definir el síntoma/objetivo y
seleccionar solo el procedimiento que ayude a resolverlo:

- [Reproducción y causa](references/debugging.md): seguir un flujo entre web,
  API, worker y proveedores y aislar la primera diferencia comprobable.
- [Observabilidad](references/observability.md): correlación segura, logs,
  heartbeat, readiness y diagnóstico entre procesos.
- [CPU, memoria y latencia](references/performance.md): perfilar cargas
  representativas y comparar antes/después sin cambiar garantías del dominio.

Separar evidencia local y del servidor; logs, mocks y perfiles no demuestran
por sí solos estado de proveedores o SQL aplicado. Para un plan SQL específico,
usar `sql-optimization-patterns`, sin ejecutar también un perfil Python.
Corregir la causa demostrada con las guías del componente afectado y verificar
la regresión. No instalar SDKs de observabilidad o probar efectos reales
como consecuencia automática de una investigación.

## Especialidades originales completas

Esta guía integra el contexto de Al Toque; no tiene un original
independiente recortado que deba reemplazarse. Para el modo afectado,
leer el perfil completo correspondiente y el
[contrato común de AI Toque](../../ai-toque-execution-context.md).
Las referencias locales complementan esos perfiles; no los sustituyen.
No cargar todos los modos ni establecer convenciones competidoras.

| Decisión | Perfil completo |
| --- | --- |
| Depuración y reproducción | [debugging-strategies](../debugging-strategies/SKILL.md) |
| Observabilidad Python | [python-observability](../../python/python-observability/SKILL.md) |
| CPU, memoria y profiling Python | [python-performance-optimization](../../python/python-performance-optimization/SKILL.md) |
| Rendimiento del sistema | [backend-development-performance-engineer](../../backend/backend-development-performance-engineer/SKILL.md) |
