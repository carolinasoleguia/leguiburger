---
name: monorepo-management
description: "Modificar limites, contratos, scripts o builds del monorepo Next/FastAPI/worker de Al Toque."
---

# Mantenimiento del monorepo

## Alcance y selección

Usar al cambiar límites entre procesos, contratos compartidos, builds o CI.
Cubre arquitectura del repositorio; el diseño interno de un módulo Python
corresponde a `python-guidelines`, sin reorganizar todo el monorepo.

Leer el [contexto del repo](../../../../AGENTS.md),
`docs/arquitectura/v2-only-cutover.md` y los scripts raíz de `package.json`.

- Conservar npm workspaces para `apps/web` y locks de dependencias Python.
  Cada aplicación debe poder construirse/desplegarse de forma independiente.
- Mantener reglas e integraciones en API y jobs durables en worker. No
  introducir imports de archivos de otro deploy como dependencia de runtime.
- Al cambiar contratos, localizar productor y todos los consumidores, incluidas
  rutas Next, handlers internos, eventos y catálogos duplicados por build.
- La auditoría `apps/web/scripts/audit-boundaries.mjs` y los tests de
  infraestructura comprueban límites; ajustar checks solo si el límite
  cambia dentro del alcance acordado.
- Revisar Dockerfiles, Compose, `.dockerignore`, variables de build/runtime,
  CI y fingerprint cuando cambien estructura o dependencias.
- Documentar el grafo de dependencias al extraer código compartido: qué proceso
  lo consume, cómo entra a cada build y quién mantiene su contrato/versionado.
  Compartir contratos no implica compartir implementación de negocio en Next.
- Para acelerar CI, medir primero; aprovechar caches de npm/pip basadas en locks
  y separar tareas independientes. No cachear secretos ni reutilizar evidencia
  de release con otro fingerprint.
- Si se propone verificar solo componentes afectados, incluir consumidores
  transitivos, contratos, SQL y configuración de build. Mantener la verificación
  completa exigida para el cierre de un cambio transversal.

Proponer cambios mínimos compatibles con los tres deploys. No migrar a otra
herramienta de monorepo para aplicar esta guía. Ejecutar las verificaciones
de los procesos afectados; los cambios transversales requieren el cierre
descrito en `docs/operacion/despliegue.md`.
