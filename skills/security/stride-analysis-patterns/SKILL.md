---
name: stride-analysis-patterns
description: "AI Toque: Apply STRIDE methodology to systematically identify threats. Use when analyzing system security, conducting threat modeling sessions, or creating security documentation."
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

El [original recibido completo](../../originals/first-import/security-scanning/skills/stride-analysis-patterns/original-skill.md) conserva los bytes y
metadata originales para trazabilidad. No es otra capa de instrucciones.
El [manifest de preservación](../../library-preservation.json) registra
integridad de cuerpos, referencias, templates y recursos.

### Recursos completos de la especialidad

- [references/details.md](references/details.md)

Seleccionar los recursos correspondientes al modo solicitado. Los templates
son ejemplos preservados; adaptarlos y validarlos antes de utilizarlos.

## Perfil especializado completo

# STRIDE Analysis Patterns

Systematic threat identification using the STRIDE methodology.

## When to Use This Skill

- Starting new threat modeling sessions
- Analyzing existing system architecture
- Reviewing security design decisions
- Creating threat documentation
- Training teams on threat identification
- Compliance and audit preparation

## Core Concepts

### 1. STRIDE Categories

```
S - Spoofing       → Authentication threats
T - Tampering      → Integrity threats
R - Repudiation    → Non-repudiation threats
I - Information    → Confidentiality threats
    Disclosure
D - Denial of      → Availability threats
    Service
E - Elevation of   → Authorization threats
    Privilege
```

### 2. Threat Analysis Matrix

| Category            | Question                                  | Control Family |
| ------------------- | ----------------------------------------- | -------------- |
| **Spoofing**        | Can attacker pretend to be someone else?  | Authentication |
| **Tampering**       | Can attacker modify data in transit/rest? | Integrity      |
| **Repudiation**     | Can attacker deny actions?                | Logging/Audit  |
| **Info Disclosure** | Can attacker access unauthorized data?    | Encryption     |
| **DoS**             | Can attacker disrupt availability?        | Rate limiting  |
| **Elevation**       | Can attacker gain higher privileges?      | Authorization  |

## Templates and detailed worked examples

Full template library lives in `references/details.md`. Read that file when you need concrete templates for this skill.

## Best Practices

### Do's

- **Involve stakeholders** - Security, dev, and ops perspectives
- **Be systematic** - Cover all STRIDE categories
- **Prioritize realistically** - Focus on high-impact threats
- **Update regularly** - Threat models are living documents
- **Use visual aids** - DFDs help communication

### Don'ts

- **Don't skip categories** - Each reveals different threats
- **Don't assume security** - Question every component
- **Don't work in isolation** - Collaborative modeling is better
- **Don't ignore low-probability** - High-impact threats matter
- **Don't stop at identification** - Follow through with mitigations
