---
name: theme-factory
description: "AI Toque: Proponer temas para artefactos o nuevas superficies; no reemplazar la paleta actual sin un encargo de rediseño."
---

# theme-factory: AI Toque

## Alcance y selección

Proponer temas para artefactos o nuevas superficies; no reemplazar la paleta actual sin un encargo de rediseño.

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

Original recibido: `web-design-skills-main/04-theme-factory/SKILL.md`, nombre `theme-factory`.
[Snapshot exacto](../../received/web-design-skills-main/04-theme-factory/received-profile.md) y [atribución/licencia recibida](../../received/web-design-skills-main/source-notice.md).
El cuerpo siguiente está completo; metadata original y permisos declarados
quedan en el snapshot, sin configurar Claude ni Codex.
Resolver `scripts/`, `references/`, `assets/`, `library/`, `locales/`, `themes/`
y `design-md/` desde este directorio canónico, nunca desde la entrada generada.
Los paths históricos compartidos se resuelven con el adaptador frontend.

## Perfil especializado completo

# Theme Factory Skill

This skill provides a curated collection of professional font and color themes themes, each with carefully selected color palettes and font pairings. Once a theme is chosen, it can be applied to any artifact.

## Purpose

To apply consistent, professional styling to presentation slide decks, use this skill. Each theme includes:
- A cohesive color palette with hex codes
- Complementary font pairings for headers and body text
- A distinct visual identity suitable for different contexts and audiences

## Usage Instructions

To apply styling to a slide deck or other artifact:

1. **Show the theme showcase**: Display the `theme-showcase.pdf` file to allow users to see all available themes visually. Do not make any modifications to it; simply show the file for viewing.
2. **Ask for their choice**: Ask which theme to apply to the deck
3. **Wait for selection**: Get explicit confirmation about the chosen theme
4. **Apply the theme**: Once a theme has been chosen, apply the selected theme's colors and fonts to the deck/artifact

## Themes Available

The following 10 themes are available, each showcased in `theme-showcase.pdf`:

1. **Ocean Depths** - Professional and calming maritime theme
2. **Sunset Boulevard** - Warm and vibrant sunset colors
3. **Forest Canopy** - Natural and grounded earth tones
4. **Modern Minimalist** - Clean and contemporary grayscale
5. **Golden Hour** - Rich and warm autumnal palette
6. **Arctic Frost** - Cool and crisp winter-inspired theme
7. **Desert Rose** - Soft and sophisticated dusty tones
8. **Tech Innovation** - Bold and modern tech aesthetic
9. **Botanical Garden** - Fresh and organic garden colors
10. **Midnight Galaxy** - Dramatic and cosmic deep tones

## Theme Details

Each theme is defined in the `themes/` directory with complete specifications including:
- Cohesive color palette with hex codes
- Complementary font pairings for headers and body text
- Distinct visual identity suitable for different contexts and audiences

## Application Process

After a preferred theme is selected:
1. Read the corresponding theme file from the `themes/` directory
2. Apply the specified colors and fonts consistently throughout the deck
3. Ensure proper contrast and readability
4. Maintain the theme's visual identity across all slides

## Create your Own Theme
To handle cases where none of the existing themes work for an artifact, create a custom theme. Based on provided inputs, generate a new theme similar to the ones above. Give the theme a similar name describing what the font/color combinations represent. Use any basic description provided to choose appropriate colors/fonts. After generating the theme, show it for review and verification. Following that, apply the theme as described above.

## Extended Themes (DESIGN.md Format)

For build pipelines that consume DESIGN.md files (`website-rebuild`, `local-business-rebuild`, `azerbaijan-website-build`), extended versions of each theme are available in the `design-md/` directory. These include full 9-section specifications:

- Complete color palette with semantic roles (not just 4 colors)
- Typography hierarchy table with sizes, weights, line-heights, and letter-spacing
- Component stylings (buttons with states, cards, inputs, navigation)
- Shadow/elevation system with exact CSS values
- Layout principles with spacing scale
- Do's and Don'ts specific to the theme
- Responsive behavior and breakpoints
- Agent prompt examples with exact CSS values

### Usage in Build Pipelines

1. Show the theme showcase as usual
2. Once the user picks a theme, read the DESIGN.md version from `design-md/[theme-name].md`
3. Copy its content to the project root as `DESIGN.md`
4. The build pipeline will consume it during code generation

### Usage for Non-Build Artifacts

For slides, docs, and reports, continue using the simple theme files in `themes/`. The simple format is sufficient for artifact styling and keeps things lightweight.
