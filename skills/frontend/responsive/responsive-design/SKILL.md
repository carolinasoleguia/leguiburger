---
name: responsive-design
description: "AI Toque: Implement modern responsive layouts using container queries, fluid typography, CSS Grid, and mobile-first breakpoint strategies. Use when building adaptive interfaces, implementing fluid layouts, or creating component-level responsive behavior."
---

## Alcance y selección — AI Toque

Especialidad completa y reutilizable entre productos. Leer primero el
[contrato de ejecución](../../../ai-toque-execution-context.md), que adapta
los supuestos del original y prevalece sobre sus recomendaciones genéricas,
y después el contexto del producto auditado o desarrollado.

Elegir esta skill por su resultado específico; leer el cuerpo completo
y las referencias pertinentes. Decidir aplicabilidad durante la ejecución,
sin borrar capacidades futuras ni activar otras guías automáticamente.
Las convenciones del producto, en especial `python-guidelines`, gobiernan
las decisiones locales. No instalar herramientas, imponer frameworks,
delegar ni ampliar permisos por cargar este perfil.

El [original recibido completo](../../../originals/second-import/ui-design/skills/responsive-design/original-skill.md) conserva los bytes y
metadata originales para trazabilidad. No es otra capa de instrucciones.
El [manifest de preservación](../../../library-preservation.json) registra
integridad de cuerpos, referencias, templates y recursos.

### Recursos completos de la especialidad

- [references/breakpoint-strategies.md](references/breakpoint-strategies.md)
- [references/container-queries.md](references/container-queries.md)
- [references/details.md](references/details.md)
- [references/fluid-layouts.md](references/fluid-layouts.md)

Seleccionar los recursos correspondientes al modo solicitado. Los templates
son ejemplos preservados; adaptarlos y validarlos antes de utilizarlos.

## Perfil especializado completo

# Responsive Design

Master modern responsive design techniques to create interfaces that adapt seamlessly across all screen sizes and device contexts.

## When to Use This Skill

- Implementing mobile-first responsive layouts
- Using container queries for component-based responsiveness
- Creating fluid typography and spacing scales
- Building complex layouts with CSS Grid and Flexbox
- Designing breakpoint strategies for design systems
- Implementing responsive images and media
- Creating adaptive navigation patterns
- Building responsive tables and data displays

## Detailed patterns and worked examples

Detailed pattern documentation lives in `references/details.md`. Read that file when the navigation tier above is insufficient.

## Best Practices

1. **Mobile-First**: Start with mobile styles, enhance for larger screens
2. **Content Breakpoints**: Set breakpoints based on content, not devices
3. **Fluid Over Fixed**: Use fluid values for typography and spacing
4. **Container Queries**: Use for component-level responsiveness
5. **Test Real Devices**: Simulators don't catch all issues
6. **Performance**: Optimize images, lazy load off-screen content
7. **Touch Targets**: Maintain 44x44px minimum on mobile
8. **Logical Properties**: Use inline/block for internationalization

## Common Issues

- **Horizontal Overflow**: Content breaking out of viewport
- **Fixed Widths**: Using px instead of relative units
- **Viewport Height**: 100vh issues on mobile browsers
- **Font Size**: Text too small on mobile
- **Touch Targets**: Buttons too small to tap accurately
- **Aspect Ratio**: Images squishing or stretching
- **Z-Index Stacking**: Overlays breaking on different screens
