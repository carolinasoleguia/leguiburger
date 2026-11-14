---
name: git-advanced-workflows
description: "Gestionar ramas, worktrees o diagnosticar regresiones Git del monorepo Al Toque preservando trabajo local."
---

# Git y worktrees

## Alcance y selección

Usar para ramas, worktrees, commits, sincronización o diagnóstico Git.
Cubre estado e historia del repositorio; no define arquitectura ni estilo.

Leer el [contexto del repo](../../../../AGENTS.md). Inspeccionar estado, rama,
remotos y worktrees; el repo puede estar en un worktree con HEAD separado.

- Para una rama desde una base indicada, comparar referencia local y remota
  actualizada, crear la rama en el worktree apropiado y comprobar su base.
- Una carpeta sin seguimiento en otro worktree no aparece al cambiar de
  rama: copiar solo material solicitado y revisar el diff antes de stage.
- No mover ni sobrescribir trabajo ajeno para limpiar el estado. Usar diff,
  log, blame y bisect cuando ayuden a explicar una regresión.
- Si Windows reporta error de fsmonitor, usar para ese comando
  `git -c core.fsmonitor=false ...`; no modificar configuración global.
- Stage por rutas del cambio. Antes del commit comprobar diff staged,
  secretos, artefactos y validación pertinente.
- Si el usuario pidió push, usar push normal con upstream de la nueva rama.
  Rebase público, force push y destrucción de cambios requieren alcance explícito.

Reportar rama, commit y resultado real del push. Un commit local no demuestra
que el remoto esté actualizado ni autoriza abrir/mergear un PR por sí solo.
