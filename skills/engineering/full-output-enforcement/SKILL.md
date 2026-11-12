---
name: full-output-enforcement
description: "AI Toque: Completar implementaciones y archivos sin placeholders cuando se requiere código completo; no obliga a imprimir todos los archivos en la respuesta."
---

# full-output-enforcement: AI Toque

## Alcance y selección

Completar implementaciones y archivos sin placeholders cuando se requiere código completo; no obliga a imprimir todos los archivos en la respuesta.

Antes de aplicar el procedimiento completo, leer el
[contrato común](../../ai-toque-execution-context.md) y el
[adaptador frontend de AI Toque](../../frontend/ai-toque-frontend-context.md). Para Al Toque,
[web-ui-design](../../frontend/web-ui-design/SKILL.md) mantiene los límites del producto.
El adaptador prevalece sobre supuestos genéricos del original: no activa
otras skills, no instala plugins, no cambia el stack ni concede permisos.

Conservar el procedimiento completo; seleccionar en ejecución lo aplicable,
no aplicable en el alcance y pendiente por herramienta/evidencia. Una variante
estética es una alternativa elegida por el brief, no una política transversal.

## Procedencia y recursos

Original recibido: `taste-skill-main/skills/output-skill/SKILL.md`, nombre `full-output-enforcement`.
[Snapshot exacto](../../frontend/received/taste-skill-main/skills/output-skill/received-profile.md) y [atribución/licencia recibida](../../frontend/received/taste-skill-main/LICENSE).
El cuerpo siguiente está completo; metadata original y permisos declarados
quedan en el snapshot, sin configurar Claude ni Codex.
Resolver `scripts/`, `references/`, `assets/`, `library/`, `locales/`, `themes/`
y `design-md/` desde este directorio canónico, nunca desde la entrada generada.
Los paths históricos compartidos se resuelven con el adaptador frontend.

## Perfil especializado completo

# Full-Output Enforcement

## Baseline

Treat every task as production-critical. A partial output is a broken output. Do not optimize for brevity — optimize for completeness. If the user asks for a full file, deliver the full file. If the user asks for 5 components, deliver 5 components. No exceptions.

## Banned Output Patterns

The following patterns are hard failures. Never produce them:

**In code blocks:** `// ...`, `// rest of code`, `// implement here`, `// TODO`, `/* ... */`, `// similar to above`, `// continue pattern`, `// add more as needed`, bare `...` standing in for omitted code

**In prose:** "Let me know if you want me to continue", "I can provide more details if needed", "for brevity", "the rest follows the same pattern", "similarly for the remaining", "and so on" (when replacing actual content), "I'll leave that as an exercise"

**Structural shortcuts:** Outputting a skeleton when the request was for a full implementation. Showing the first and last section while skipping the middle. Replacing repeated logic with one example and a description. Describing what code should do instead of writing it.

## Execution Process

1. **Scope** — Read the full request. Count how many distinct deliverables are expected (files, functions, sections, answers). Lock that number.
2. **Build** — Generate every deliverable completely. No partial drafts, no "you can extend this later."
3. **Cross-check** — Before output, re-read the original request. Compare your deliverable count against the scope count. If anything is missing, add it before responding.

## Handling Long Outputs

When a response approaches the token limit:

- Do not compress remaining sections to squeeze them in.
- Do not skip ahead to a conclusion.
- Write at full quality up to a clean breakpoint (end of a function, end of a file, end of a section).
- End with:

```
[PAUSED — X of Y complete. Send "continue" to resume from: next section name]
```

On "continue", pick up exactly where you stopped. No recap, no repetition.

## Quick Check

Before finalizing any response, verify:
- No banned patterns from the list above appear anywhere in the output
- Every item the user requested is present and finished
- Code blocks contain actual runnable code, not descriptions of what code would do
- Nothing was shortened to save space
