---
name: projection-patterns
description: "AI Toque: Build read models and projections from event streams. Use when implementing CQRS read sides, building materialized views, or optimizing query performance in event-sourced systems."
---

## Alcance y selección — AI Toque

Especialidad completa y reutilizable entre productos. Leer primero el
[contrato de ejecución](../../ai-toque-execution-context.md), que adapta
los supuestos del original y prevalece sobre sus recomendaciones genéricas,
y después el contexto del producto auditado o desarrollado.

Elegir esta skill por su resultado específico; leer el cuerpo completo
y las referencias pertinentes. Decidir aplicabilidad durante la ejecución,
sin borrar capacidades futuras ni activar otras guías automáticamente.
Las convenciones del producto, en especial `python-guidelines`, gobiernan
las decisiones locales. No instalar herramientas, imponer frameworks,
delegar ni ampliar permisos por cargar este perfil.

El [original recibido completo](../../originals/second-import/backend-development/skills/projection-patterns/original-skill.md) conserva los bytes y
metadata originales para trazabilidad. No es otra capa de instrucciones.
El [manifest de preservación](../../library-preservation.json) registra
integridad de cuerpos, referencias, templates y recursos.

### Recursos completos de la especialidad

- [references/details.md](references/details.md)

Seleccionar los recursos correspondientes al modo solicitado. Los templates
son ejemplos preservados; adaptarlos y validarlos antes de utilizarlos.

## Perfil especializado completo

# Projection Patterns

Comprehensive guide to building projections and read models for event-sourced systems.

## When to Use This Skill

- Building CQRS read models
- Creating materialized views from events
- Optimizing query performance
- Implementing real-time dashboards
- Building search indexes from events
- Aggregating data across streams

## Core Concepts

### 1. Projection Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Event Store │────►│ Projector   │────►│ Read Model  │
│             │     │             │     │ (Database)  │
│ ┌─────────┐ │     │ ┌─────────┐ │     │ ┌─────────┐ │
│ │ Events  │ │     │ │ Handler │ │     │ │ Tables  │ │
│ └─────────┘ │     │ │ Logic   │ │     │ │ Views   │ │
│             │     │ └─────────┘ │     │ │ Cache   │ │
└─────────────┘     └─────────────┘     └─────────────┘
```

### 2. Projection Types

| Type           | Description                 | Use Case               |
| -------------- | --------------------------- | ---------------------- |
| **Live**       | Real-time from subscription | Current state queries  |
| **Catchup**    | Process historical events   | Rebuilding read models |
| **Persistent** | Stores checkpoint           | Resume after restart   |
| **Inline**     | Same transaction as write   | Strong consistency     |

## Templates and detailed worked examples

Full template library and detailed worked examples live in `references/details.md`. Read that file when you need the concrete templates.

## Best Practices

### Do's

- **Make projections idempotent** - Safe to replay
- **Use transactions** - For multi-table updates
- **Store checkpoints** - Resume after failures
- **Monitor lag** - Alert on projection delays
- **Plan for rebuilds** - Design for reconstruction

### Don'ts

- **Don't couple projections** - Each is independent
- **Don't skip error handling** - Log and alert on failures
- **Don't ignore ordering** - Events must be processed in order
- **Don't over-normalize** - Denormalize for query patterns
