---
name: python-guidelines
description: "Implementar o refactorizar Python de Al Toque: estilo 3.11, tipos, responsabilidades y Settings. Usar para convenciones de codigo."
---

# Guía Python

## Alcance y selección

- Usar al escribir/refactorizar módulos Python, definir sus tipos o modificar
  Settings. Es la referencia única de convenciones de implementación Python.
- Cubre estilo, diseño, organización, tipado y antipatrones del código.
- Contratos HTTP, modelo SQL y presentación web tienen guías propias; no
  reemplazan esta guía ni obligan a reorganizar código Python.

Leer el [contexto del repo](../../../AGENTS.md). Mantener Python 3.11, locks,
estructura y pruebas existentes; aplicar la convención del módulo adyacente.

## Procedimiento

1. Localizar consumidores y frontera: ruta HTTP, caso de uso core/bot,
   adaptador de I/O o dispatch del worker. Conservar reglas de negocio en API.
2. Elegir el cambio mínimo que aísle la decisión/efecto real. Documentar
   invariantes de permisos, idempotencia o estados inciertos cuando corresponda.
3. Leer [implementación y tipos](references/implementation.md) si hace falta
   decidir composición, interfaces, movimiento de módulos o validación de DTO.
4. Leer [configuración](references/configuration.md) solo si cambian Settings,
   variables de entorno, defaults o configuración por servicio.
5. Validar según el comportamiento afectado. Para pruebas usar
   `python-testing-patterns`; para concurrencia/reintentos/jobs,
   `python-reliability`, solo cuando el cambio necesite ese procedimiento.

No introducir ORM, framework de colas, contenedor de inyección, type checker
o formatter global por aplicar esta guía. No imponer clases/interfaces para
cada helper ni reescribir módulos fuera del alcance.

## Especialidades originales completas

Esta guía integra el contexto de Al Toque; no tiene un original
independiente recortado que deba reemplazarse. Para el modo afectado,
leer el perfil completo correspondiente y el
[contrato común de AI Toque](../../ai-toque-execution-context.md).
Las referencias locales complementan esos perfiles; no los sustituyen.
No cargar todos los modos ni establecer convenciones competidoras.

| Decisión | Perfil completo |
| --- | --- |
| Estilo, documentación y linting | [python-code-style](../python-code-style/SKILL.md) |
| Composición, responsabilidades y patrones | [python-design-patterns](../python-design-patterns/SKILL.md) |
| Módulos y estructura | [python-project-structure](../python-project-structure/SKILL.md) |
| Tipado, protocolos y genéricos | [python-type-safety](../python-type-safety/SKILL.md) |
| Settings y configuración | [python-configuration](../python-configuration/SKILL.md) |
| Antipatrones y revisión | [python-anti-patterns](../python-anti-patterns/SKILL.md) |
| Packaging y publicación | [python-packaging](../python-packaging/SKILL.md) |
| Gestión con uv | [uv-package-manager](../uv-package-manager/SKILL.md) |
| Diseño/implementación Python profunda | [python-pro](../python-pro/SKILL.md) |
| FastAPI como especialidad completa | [python-development-fastapi-pro](../python-development-fastapi-pro/SKILL.md) |
| Scaffolding autorizado | [python-scaffold](../python-scaffold/SKILL.md) |
