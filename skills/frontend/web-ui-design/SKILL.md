---
name: web-ui-design
description: "Crear, revisar o probar UI web de Al Toque: React, temas, responsive, accesibilidad y feedback. Elegir solo el modo afectado."
---

# Diseño de la web

## Alcance y selección

- Usar para presentación, accesibilidad o pruebas de flujo del frontend.
- Cubre React/Next/Tailwind y fixtures de navegador; la lógica de negocio,
  implementación Python, contrato REST y SQL siguen sus propias guías.
- Elegir componentes/temas, responsive/feedback, accesibilidad o navegador
  según la tarea. Una corrección visual no exige una auditoría completa.

Leer el [contexto del repo](../../../AGENTS.md) y las
[instrucciones del frontend](../../../apps/web/AGENTS.md). Aplicar esta guía a la
presentación en Next/React/Tailwind; las reglas de negocio siguen en la API.
No incluye desarrollo Python ni pide cambiar el contrato HTTP para mejorar UI.

- Revisar pantalla, consumidores y componentes adyacentes antes de abstraer.
  Conservar la identidad visual del producto y el alcance del cambio.
- Usar React, Tailwind v4, estilos y `next-themes` existentes; comprobar las
  convenciones de la versión instalada de Next. No añadir librerías visuales,
  animaciones o pipelines de tokens para aplicar ejemplos de una guía.
- Mantener semántica, labels, foco visible y feedback accesible en cualquier
  cambio de interacción. Evitar controles que solo funcionan con mouse.
- Reutilizar componentes cuando haya consumidores o una responsabilidad real
  compartida. No asumir que existe `components/ui` ni imponer otra estructura.
- Preservar payload, validación, permisos y resultado de acciones del flujo.
  Un estado visual de éxito necesita resultado confirmado o explicación de
  trabajo pendiente, según el contrato actual.

Leer solo las referencias necesarias para la tarea:

- [Componentes, estilo y temas](references/components-and-themes.md): para
  composición React, variantes, tokens, tipografía, colores o light/dark.
- [Responsive y feedback](references/responsive-and-feedback.md): para
  layout móvil, tablas, adjuntos, loading/error y transiciones.
- [Accesibilidad](references/accessibility.md): para formularios, dialogs,
  teclado, foco, lectores de pantalla o una revisión específica de acceso.
- [Pruebas de navegador](references/browser-tests.md): para crear/mantener
  fixtures Puppeteer o verificar una regresión de flujo observable.

Validar tipos/lint y tests web proporcionales al cambio. Para regresiones de
flujo usar las fixtures Puppeteer existentes; no agregar otro runner. Registrar
qué tamaños, temas, estados y navegación se comprobaron. Una revisión parcial
o un escáner automático no acredita conformidad completa de accesibilidad.

## Especialidades originales completas

Esta guía integra el contexto de Al Toque; no tiene un original
independiente recortado que deba reemplazarse. Para el modo afectado,
leer el perfil completo correspondiente y el
[contrato común de AI Toque](../../ai-toque-execution-context.md).
Las referencias locales complementan esos perfiles; no los sustituyen.
No cargar todos los modos ni establecer convenciones competidoras.

| Decisión | Perfil completo |
| --- | --- |
| Componentes y contratos de UI | [web-component-design](../components/web-component-design/SKILL.md) |
| Sistemas de diseño, tokens y temas | [design-system-patterns](../design-systems/design-system-patterns/SKILL.md) |
| Layout y responsive | [responsive-design](../responsive/responsive-design/SKILL.md) |
| Interacción y animaciones | [interaction-design](../components/interaction-design/SKILL.md) |
| Tipografía, color y composición visual | [visual-design-foundations](../design-systems/visual-design-foundations/SKILL.md) |
| Accesibilidad | [accessibility-compliance](../accessibility/accessibility-compliance/SKILL.md) |
| Pruebas E2E | [e2e-testing-patterns](../../testing/e2e-testing-patterns/SKILL.md) |
| Auditoría detallada de accesibilidad | [accessibility-audit](../accessibility/accessibility-audit/SKILL.md) |
| Creación de componentes | [create-component](../components/create-component/SKILL.md) |
| Revisión de diseño | [design-review](../review/design-review/SKILL.md) |
| Setup de un sistema de diseño | [design-system-setup](../design-systems/design-system-setup/SKILL.md) |

## Landing pages y biblioteca de dirección visual

Para crear una landing desde oferta, audiencia y CTA, usar
[landing-page-design](../landing-pages/landing-page-design/SKILL.md) como guideline reutilizable
de AI Toque. No depende de que Al Toque tenga hoy ese desarrollo. Leer el
[adaptador frontend](../ai-toque-frontend-context.md) para separar estrategia,
estética elegida y restricciones del producto; no trasladar reglas de landing
al dashboard.

Para revisión/planificación amplia de pantallas, elegir
[design-guide](../review/design-guide/SKILL.md). Para patrones visuales genéricos,
[avoid-ai-design](../review/avoid-ai-design/SKILL.md) o
[antislop](../review/antislop/SKILL.md) son procedimientos alternativos.
[design-taste-frontend](../landing-pages/design-taste-frontend/SKILL.md) aporta dirección
artística para páginas públicas y conserva biblioteca/helpers completos.
El [catálogo](../../README.md#importación-frontend-guideline-de-landing-pages-y-especialistas-completos)
incluye estilos, reconstrucción, i18n, temas e imágenes. Elegir el mínimo
conjunto pertinente; no encadenarlos ni imponer sus defaults a UI existente.
