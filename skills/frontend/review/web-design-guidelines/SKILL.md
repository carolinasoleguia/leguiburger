---
name: web-design-guidelines
description: "AI Toque: Revisar archivos de UI contra Web Interface Guidelines, con versión/fuente y límites de evidencia explícitos."
---

# web-design-guidelines: AI Toque

## Alcance y selección

Revisar archivos de UI contra Web Interface Guidelines, con versión/fuente y límites de evidencia explícitos.

Antes de aplicar el procedimiento completo, leer el
[contrato común](../../../ai-toque-execution-context.md) y el
[adaptador frontend de AI Toque](../../ai-toque-frontend-context.md). Para Al Toque,
[web-ui-design](../../web-ui-design/SKILL.md) mantiene los límites del producto.
El adaptador prevalece sobre supuestos genéricos del original: no activa
otras skills, no instala plugins, no cambia el stack ni concede permisos.

Conservar el procedimiento completo; seleccionar en ejecución lo aplicable,
no aplicable en el alcance y pendiente por herramienta/evidencia. Una variante
estética es una alternativa elegida por el brief, no una política transversal.

## Procedencia y recursos

Fuente oficial: https://github.com/vercel-labs/agent-skills, commit `063bee94c3f4df8453406c830b0a7df0f2860278`,
archivo `skills/web-design-guidelines/SKILL.md`.
[Snapshot exacto](../../received/vercel-agent-skills-063bee94c3f4/received-profile.md) y [licencia MIT](../../received/vercel-agent-skills-063bee94c3f4/source-README.md).
El cuerpo siguiente se conserva completo; el frontmatter portable y el contexto
local preceden al original. Los snapshots anteriores permanecen en `received/`.
Resolver recursos relativos desde este directorio canónico, no desde las entradas
de `.agents/skills/` o `.claude/skills/`.

## Perfil especializado completo

# Web Interface Guidelines

Review files for compliance with Web Interface Guidelines.

## How It Works

1. Fetch the latest guidelines from the source URL below
2. Read the specified files (or prompt user for files/pattern)
3. Check against all rules in the fetched guidelines
4. Output findings in the terse `file:line` format

## Guidelines Source

Fetch fresh guidelines before each review:

```
https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md
```

Use WebFetch to retrieve the latest rules. The fetched content contains all the rules and output format instructions.

## Usage

When a user provides a file or pattern argument:
1. Fetch guidelines from the source URL above
2. Read the specified files
3. Apply all rules from the fetched guidelines
4. Output findings using the format specified in the guidelines

If no files specified, ask the user which files to review.
