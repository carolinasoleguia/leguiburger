# Biblioteca completa de skills de AI Toque

Catálogo compartido de **134 skills** para Claude Code y Codex:
84 especialidades/procedimientos restaurados de las dos primeras importaciones,
11 perfiles completos de seguridad restaurados anteriormente y 6 guías integradas
de Al Toque, más **31 skills completas de diseño** extraídas de siete repos
recibidos en la importación frontend. Los 84 originales incluyen **43 skills, 24 perfiles de agentes y
17 comandos**, convertidos a entradas portables sin recortar sus cuerpos.

Los originales seleccionados están activos completos: Python/FastAPI,
Next/React, datos, arquitectura, CQRS/event stores, multiagente/memoria, packaging
y tooling Python. Su aplicabilidad se decide al ejecutarlos. Se excluyen las
especialidades dedicadas a blockchain, Django, móvil nativo/React Native, Temporal,
GraphQL y Nx/Bazel/Turborepo, por decisión del usuario sobre el stack de AI Toque.

## Entrada y uso

La fuente es `skills/<grupo>/<nombre>/SKILL.md`, con subgrupos opcionales como
`skills/frontend/<subgrupo>/<nombre>/SKILL.md`. Las entradas en
`.agents/skills/<nombre>/SKILL.md` y `.claude/skills/<nombre>/SKILL.md`
apuntan a la misma fuente. Invocar en Codex con `$nombre` y en Claude con
`/nombre`. Los perfiles de agentes son skills completas; no crean subagentes
automáticamente. Los comandos son procedimientos portables; no instalan plugins.

Leer el [contrato común de ejecución](ai-toque-execution-context.md) y el
contexto del producto antes de una especialidad restaurada. Los 11 perfiles
de seguridad mantienen además su
[contrato específico](security/ai-toque-security-context.md). Las entradas
piden leer la fuente completa y seleccionar solo las referencias pertinentes.

## Selección y precedencia sin conflicto

- Elegir el menor conjunto que cubra el resultado pedido, no todas las skills
  de un grupo. Un perfil explícitamente solicitado se lee completo.
- `python-guidelines` es la autoridad local de convenciones. Las guías completas
  de estilo, tipos, estructura, patrones y Settings aportan técnicas dentro de
  esa decisión; no establecen estándares alternativos simultáneos.
- `python-reliability` orienta ejecución; los especialistas desarrollan async,
  recursos, resiliencia, errores y jobs. `python-testing-patterns` cubre pruebas.
- REST/GraphQL, modelo SQL, evolución de migraciones y presentación tienen
  ámbitos distintos. Combinar solo cuando el cambio cruce esas decisiones.
- Modelos, permisos, `PROACTIVELY`, llamadas `Task` y ejemplos de orquestación
  del material original no configuran el host ni autorizan efectos externos.
- Registrar lo aplicable, no aplicable en este alcance y pendiente/no ejecutado.
  La falta de herramientas no es un resultado aprobado ni motivo para recortar.

Las 6 guías integradas conservadas son `python-guidelines`, `python-reliability`,
`diagnostics-and-performance`, `web-ui-design`, `database-migrations` y
`security-review`. Sus modos enlazan los especialistas completos. Las otras
8 guías que ya existían con nombre original ahora contienen su cuerpo completo;
la integración anterior sigue en `references/product-guidance.md`.

## Organización por naturaleza

| Carpeta | Contenido | Skills |
| --- | --- | --- |
| `python/` | Convenciones y ejecución, especialistas Python, FastAPI, packaging y scaffolding. | 21 |
| `security/` | 11 perfiles completos de seguridad, guías originales de auth/amenazas/scanning y procedimientos. | 27 |
| `engineering/` | Revisión, refactorización, Git, monorepo y build systems. | 13 |
| `backend/` | Arquitectura, patrones distribuidos, eventos y desarrollo de features. | 11 |
| `database/` | Diseño, SQL, administración, optimización y migraciones. | 9 |
| `frontend/` | UI, accesibilidad, componentes, landing pages, dirección visual y sistemas de diseño. | 45 |
| `agents/` | Contexto, memoria, mejora y orquestación de agentes. | 4 |
| `api/` | Diseño de contratos REST y referencias completas de API. | 1 |
| `testing/` | E2E, automatización de pruebas y TDD. | 3 |

`originals/` conserva snapshots y metadata recibidos; no es un grupo ejecutable.
Los manifests registran procedencia e integridad. La agrupación de fuentes no
cambia los nombres de invocación y las entradas nativas siguen planas.

Frontend se agrupa también por propósito. `web-ui-design` y el adaptador están
en la raíz de `frontend/`; los especialistas quedan en estos subgrupos:

| Subgrupo frontend | Propósito | Skills |
| --- | --- | --- |
| `landing-pages/` | Landing pages, páginas públicas y reconstrucción web. | 8 |
| `review/` | Planificación, auditoría, rediseño y filtros visuales. | 8 |
| `design-systems/` | Tokens, temas, DESIGN.md y fundamentos visuales. | 7 |
| `styles/` | Direcciones estéticas alternativas. | 4 |
| `visual-assets/` | Boards, imágenes de UI y reproducción desde imágenes. | 3 |
| `responsive/` | Layout web entre tamaños y dispositivos. | 3 |
| `accessibility/` | Accesibilidad, contraste, teclado y auditoría. | 4 |
| `components/` | Componentes, contratos de UI e interacción. | 4 |
| `content/` | Copy y wordings de interfaz/marketing. | 1 |
| `performance/` | Rendimiento frontend, Web Vitals, React y bundles. | 1 |

Agrupar fuentes no agrupa la ejecución: se sigue eligiendo el menor conjunto
pertinente y las entradas nativas permanecen planas, con los mismos nombres.

## Catálogo por decisión

| Tarea/decisión | Skill | Alcance |
| --- | --- | --- |
| Escribir/refactorizar Python | [python-guidelines](python/python-guidelines/SKILL.md) | Estilo 3.11, tipos, responsabilidades, módulos y Settings. |
| Cambiar ejecución Python | [python-reliability](python/python-reliability/SKILL.md) | Concurrencia, retries, outbox, recursos y fallos parciales, por modo. |
| Escribir pruebas Python | [python-testing-patterns](python/python-testing-patterns/SKILL.md) | unittest, mocks y fixtures de API/worker/herramientas. |
| Diseñar contrato REST | [api-design-principles](api/api-design-principles/SKILL.md) | DTO, rutas, status, paginación y compatibilidad de consumidores. |
| Diseñar datos | [postgresql-table-design](database/postgresql-table-design/SKILL.md) | Tipos, claves, restricciones y pertenencia al taller. |
| Evolucionar SQL | [database-migrations](database/database-migrations/SKILL.md) | Historial, orden compatible, backfills y validación de staging. |
| Mejorar una consulta lenta | [sql-optimization-patterns](database/sql-optimization-patterns/SKILL.md) | Planes, índices y mediciones. |
| Crear/revisar/probar frontend | [web-ui-design](frontend/web-ui-design/SKILL.md) | Componentes/temas, responsive/feedback, accesibilidad y navegador. |
| Investigar fallos/rendimiento | [diagnostics-and-performance](engineering/diagnostics-and-performance/SKILL.md) | Reproducción, observabilidad o profiling, por modo. |
| Diseñar/implementar/revisar seguridad | [security-review](security/security-review/SKILL.md) | Aplicación por diff/flujo, autorización, IA/tools, credenciales, amenazas o remediación, por modo. |
| Preparar análisis automático | [sast-configuration](security/sast-configuration/SKILL.md) | Configuración de escáneres y evidencia realmente ejecutada. |
| Revisar PR/diff o deuda técnica | [code-review-excellence](engineering/code-review-excellence/SKILL.md) | Hallazgos sustentados y refactors abordables. |
| Cambiar límites/builds/contratos compartidos | [monorepo-management](engineering/monorepo-management/SKILL.md) | Arquitectura entre web, API y worker y sus deploys. |
| Gestionar rama/worktree/commit/push | [git-advanced-workflows](engineering/git-advanced-workflows/SKILL.md) | Estado e historia de Git, preservando trabajo local. |

## Especialistas completos de seguridad de AI Toque

| Especialidad | Skill | Cuándo elegirla |
| --- | --- | --- |
| Arquitectura de seguridad | [security-architect](security/security-architect/SKILL.md) | Fronteras, amenazas, identidad, datos, defensa en profundidad y decisiones de arquitectura. |
| Ingeniería AppSec | [security-appsec-engineer](security/security-appsec-engineer/SKILL.md) | SDLC, code review, desarrollo seguro, SAST/SCA/DAST y regresiones. |
| Código generado con IA | [security-ai-generated-code-auditor](security/security-ai-generated-code-auditor/SKILL.md) | Secretos, RLS, prompt injection, taint y scan-fix-rescan. |
| Credenciales | [security-secrets-credential-engineer](security/security-secrets-credential-engineer/SKILL.md) | Inventario, scopes, broker, claves dinámicas, rotación y exposición. |
| Controles defensivos | [security-senior-secops](security/security-senior-secops/SKILL.md) | JWT, cookies, roles, CORS, headers, input, secrets y logging según estándar del producto. |
| Arquitectura cloud | [security-cloud-security-architect](security/security-cloud-security-architect/SKILL.md) | IAM, red, IaC, workloads, Kubernetes, supply chain y guardrails de la plataforma real. |
| Compliance | [security-compliance-auditor](security/security-compliance-auditor/SKILL.md) | Framework, controles, período, evidencia y readiness; no certificación inferida. |
| Incidentes | [security-incident-responder](security/security-incident-responder/SKILL.md) | Triage, forense, contención, recuperación y postmortem autorizados. |
| Pentesting | [security-penetration-tester](security/security-penetration-tester/SKILL.md) | Preparación/pruebas activas, cadenas de ataque, reporte y retest con targets y permiso definidos. |
| Detección | [security-threat-detection-engineer](security/security-threat-detection-engineer/SKILL.md) | Sigma/SIEM, fuentes, ATT&CK, hunting, falsos positivos y validación. |
| Inteligencia | [security-threat-intelligence-analyst](security/security-threat-intelligence-analyst/SKILL.md) | IOCs, fuentes, TTPs, confianza, atribución y productos de inteligencia. |

## Catálogo completo de originales restaurados

Cada enlace lleva al contenido activo completo, no a un resumen ni a un snapshot.
Los nombres recibidos con namespace se mantienen para distinguir perfiles de
procedencia diferente. No ejecutar varios auditores o arquitectos por tenerlos
disponibles: seleccionar uno por su alcance y resultado.

### python

| Especialidad | Tipo recibido |
| --- | --- |
| [async-python-patterns](python/async-python-patterns/SKILL.md) | Skill |
| [python-anti-patterns](python/python-anti-patterns/SKILL.md) | Skill |
| [python-background-jobs](python/python-background-jobs/SKILL.md) | Skill |
| [python-code-style](python/python-code-style/SKILL.md) | Skill |
| [python-configuration](python/python-configuration/SKILL.md) | Skill |
| [python-design-patterns](python/python-design-patterns/SKILL.md) | Skill |
| [python-development-fastapi-pro](python/python-development-fastapi-pro/SKILL.md) | Perfil de agente |
| [python-error-handling](python/python-error-handling/SKILL.md) | Skill |
| [python-observability](python/python-observability/SKILL.md) | Skill |
| [python-packaging](python/python-packaging/SKILL.md) | Skill |
| [python-performance-optimization](python/python-performance-optimization/SKILL.md) | Skill |
| [python-pro](python/python-pro/SKILL.md) | Perfil de agente |
| [python-project-structure](python/python-project-structure/SKILL.md) | Skill |
| [python-resilience](python/python-resilience/SKILL.md) | Skill |
| [python-resource-management](python/python-resource-management/SKILL.md) | Skill |
| [python-scaffold](python/python-scaffold/SKILL.md) | Procedimiento |
| [python-testing-patterns](python/python-testing-patterns/SKILL.md) | Skill |
| [python-type-safety](python/python-type-safety/SKILL.md) | Skill |
| [uv-package-manager](python/uv-package-manager/SKILL.md) | Skill |

### security

| Especialidad | Tipo recibido |
| --- | --- |
| [attack-tree-construction](security/attack-tree-construction/SKILL.md) | Skill |
| [auth-implementation-patterns](security/auth-implementation-patterns/SKILL.md) | Skill |
| [backend-api-security-backend-security-coder](security/backend-api-security-backend-security-coder/SKILL.md) | Perfil de agente |
| [backend-development-security-auditor](security/backend-development-security-auditor/SKILL.md) | Perfil de agente |
| [compliance-check](security/compliance-check/SKILL.md) | Procedimiento |
| [sast-configuration](security/sast-configuration/SKILL.md) | Skill |
| [security-compliance-security-auditor](security/security-compliance-security-auditor/SKILL.md) | Perfil de agente |
| [security-dependencies](security/security-dependencies/SKILL.md) | Procedimiento |
| [security-hardening](security/security-hardening/SKILL.md) | Procedimiento |
| [security-requirement-extraction](security/security-requirement-extraction/SKILL.md) | Skill |
| [security-sast](security/security-sast/SKILL.md) | Procedimiento |
| [security-scanning-security-auditor](security/security-scanning-security-auditor/SKILL.md) | Perfil de agente |
| [stride-analysis-patterns](security/stride-analysis-patterns/SKILL.md) | Skill |
| [threat-mitigation-mapping](security/threat-mitigation-mapping/SKILL.md) | Skill |
| [threat-modeling-expert](security/threat-modeling-expert/SKILL.md) | Perfil de agente |

### engineering

| Especialidad | Tipo recibido |
| --- | --- |
| [code-refactoring-code-reviewer](engineering/code-refactoring-code-reviewer/SKILL.md) | Perfil de agente |
| [code-refactoring-legacy-modernizer](engineering/code-refactoring-legacy-modernizer/SKILL.md) | Perfil de agente |
| [code-review-excellence](engineering/code-review-excellence/SKILL.md) | Skill |
| [debugging-strategies](engineering/debugging-strategies/SKILL.md) | Skill |
| [error-handling-patterns](engineering/error-handling-patterns/SKILL.md) | Skill |
| [git-advanced-workflows](engineering/git-advanced-workflows/SKILL.md) | Skill |
| [monorepo-architect](engineering/monorepo-architect/SKILL.md) | Perfil de agente |
| [monorepo-management](engineering/monorepo-management/SKILL.md) | Skill |
| [refactor-clean](engineering/refactor-clean/SKILL.md) | Procedimiento |
| [tech-debt](engineering/tech-debt/SKILL.md) | Procedimiento |

### backend

| Especialidad | Tipo recibido |
| --- | --- |
| [architecture-patterns](backend/architecture-patterns/SKILL.md) | Skill |
| [backend-api-security-backend-architect](backend/backend-api-security-backend-architect/SKILL.md) | Perfil de agente |
| [backend-development-backend-architect](backend/backend-development-backend-architect/SKILL.md) | Perfil de agente |
| [backend-development-performance-engineer](backend/backend-development-performance-engineer/SKILL.md) | Perfil de agente |
| [cqrs-implementation](backend/cqrs-implementation/SKILL.md) | Skill |
| [event-sourcing-architect](backend/event-sourcing-architect/SKILL.md) | Perfil de agente |
| [event-store-design](backend/event-store-design/SKILL.md) | Skill |
| [feature-development](backend/feature-development/SKILL.md) | Procedimiento |
| [microservices-patterns](backend/microservices-patterns/SKILL.md) | Skill |
| [projection-patterns](backend/projection-patterns/SKILL.md) | Skill |
| [saga-orchestration](backend/saga-orchestration/SKILL.md) | Skill |

### database

| Especialidad | Tipo recibido |
| --- | --- |
| [database-admin](database/database-admin/SKILL.md) | Perfil de agente |
| [database-design-database-architect](database/database-design-database-architect/SKILL.md) | Perfil de agente |
| [database-migrations-database-optimizer](database/database-migrations-database-optimizer/SKILL.md) | Perfil de agente |
| [migration-observability](database/migration-observability/SKILL.md) | Procedimiento |
| [postgresql-table-design](database/postgresql-table-design/SKILL.md) | Skill |
| [sql-migrations](database/sql-migrations/SKILL.md) | Procedimiento |
| [sql-optimization-patterns](database/sql-optimization-patterns/SKILL.md) | Skill |
| [sql-pro](database/sql-pro/SKILL.md) | Perfil de agente |

### frontend

| Especialidad | Tipo recibido |
| --- | --- |
| [accessibility-audit](frontend/accessibility/accessibility-audit/SKILL.md) | Procedimiento |
| [accessibility-compliance](frontend/accessibility/accessibility-compliance/SKILL.md) | Skill |
| [accessibility-expert](frontend/accessibility/accessibility-expert/SKILL.md) | Perfil de agente |
| [create-component](frontend/components/create-component/SKILL.md) | Procedimiento |
| [design-review](frontend/review/design-review/SKILL.md) | Procedimiento |
| [design-system-architect](frontend/design-systems/design-system-architect/SKILL.md) | Perfil de agente |
| [design-system-patterns](frontend/design-systems/design-system-patterns/SKILL.md) | Skill |
| [design-system-setup](frontend/design-systems/design-system-setup/SKILL.md) | Procedimiento |
| [interaction-design](frontend/components/interaction-design/SKILL.md) | Skill |
| [responsive-design](frontend/responsive/responsive-design/SKILL.md) | Skill |
| [ui-designer](frontend/components/ui-designer/SKILL.md) | Perfil de agente |
| [visual-design-foundations](frontend/design-systems/visual-design-foundations/SKILL.md) | Skill |
| [web-component-design](frontend/components/web-component-design/SKILL.md) | Skill |

### agents

| Especialidad | Tipo recibido |
| --- | --- |
| [agent-orchestration-context-manager](agents/agent-orchestration-context-manager/SKILL.md) | Perfil de agente |
| [context-restore](agents/context-restore/SKILL.md) | Procedimiento |
| [improve-agent](agents/improve-agent/SKILL.md) | Procedimiento |
| [multi-agent-optimize](agents/multi-agent-optimize/SKILL.md) | Procedimiento |

### api

| Especialidad | Tipo recibido |
| --- | --- |
| [api-design-principles](api/api-design-principles/SKILL.md) | Skill |

### testing

| Especialidad | Tipo recibido |
| --- | --- |
| [backend-development-tdd-orchestrator](testing/backend-development-tdd-orchestrator/SKILL.md) | Perfil de agente |
| [backend-development-test-automator](testing/backend-development-test-automator/SKILL.md) | Perfil de agente |
| [e2e-testing-patterns](testing/e2e-testing-patterns/SKILL.md) | Skill |

## Importación frontend: guideline de landing pages y especialistas completos

La séptima carpeta agrega `landing-page-design`: guideline reutilizable de AI Toque
para oferta, audiencia, CTA, estrategia, copy y construcción de landing pages,
independientemente del producto actual. El
[adaptador frontend](frontend/ai-toque-frontend-context.md) separa la guía general
del stack/tokens de Al Toque. Una landing no aplica su sistema visual al dashboard.

Se extrajeron **31 skills de siete repos**, con sus cuerpos completos, **127 recursos**
y snapshots exactos. Los instaladores, marketplaces, paquetes CLI, demos y tooling
de mantenimiento de origen quedan fuera de la biblioteca activa. La skill dedicada
a mockups de apps nativas iOS/Android se excluye según el alcance ya acordado;
web responsive y referencias móviles se conservan.

Elegir un procedimiento principal y una dirección visual compatible. `design-guide`
aporta planificación/revisión; `avoid-ai-design` y `antislop` son métodos alternativos
de diagnóstico. `no-slop-ui`, estilos expresivos y variantes históricas de taste no
se encadenan ni gobiernan las mismas decisiones simultáneamente. `web-ui-design`
sigue orientando los cambios del producto. No se instala otro stack ni se convierten
las preferencias de los originales en instrucciones globales.

El nombre previo `responsive-design` se mantiene; la nueva variante usa
`responsive-design-web-patterns`. `azerbaijan-website-build` queda disponible como
`multilingual-business-design`, con proceso/reglas/región originales íntegros y
locales seleccionados por ejecución. Los paths compartidos se resuelven en el
adaptador, sin duplicar las diez referencias de marca.

| Skill | Carpeta recibida | Cuándo usarla |
| --- | --- | --- |
| [antislop](frontend/review/antislop/SKILL.md) | `anti-slop-main` | Evaluar decisiones visuales y patrones genéricos con el filtro completo antislop; elegir auditoría o implementación según el encargo. |
| [antislop-code](engineering/antislop-code/SKILL.md) | `anti-slop-main` | Revisar higiene de comentarios sin cambiar comportamiento; complementar las convenciones del lenguaje, sin redefinirlas. |
| [antislop-copywriting](frontend/content/antislop-copywriting/SKILL.md) | `anti-slop-main` | Redactar o revisar copy, headlines y CTAs del producto; conservar significado, hechos verificables y localización. |
| [antislop-human](frontend/accessibility/antislop-human/SKILL.md) | `anti-slop-main` | Revisar contraste, teclado, foco y estados de UI; incluye cálculo local de contraste y referencias completas. |
| [antislop-layoutmobile](frontend/responsive/antislop-layoutmobile/SKILL.md) | `anti-slop-main` | Revisar reflow, overflow y objetivos táctiles de la web móvil; no implica desarrollo nativo. |
| [antislop-ui](frontend/review/antislop-ui/SKILL.md) | `anti-slop-main` | Aplicar el filtro visual antislop a composición, color, componentes y movimiento cuando ese filtro sea el elegido. |
| [avoid-ai-design](frontend/review/avoid-ai-design/SKILL.md) | `avoid-ai-design-main` | Auditar patrones visuales genéricos o corregirlos si se pide; incluye detector estático y catálogo de señales. |
| [design-guide](frontend/review/design-guide/SKILL.md) | `design-guide-main` | Planificar o revisar pantallas, estados y flujos de producto; elegir profundidad y evidencias proporcionales al cambio. |
| [no-slop-ui](frontend/review/no-slop-ui/SKILL.md) | `no-slop-ui-master` | Diseñar interfaces operativas sobrias o revisar su composición; dirección alternativa, no estándar simultáneo con estilos expresivos. |
| [brandkit](frontend/visual-assets/brandkit/SKILL.md) | `taste-skill-main` | Crear boards de identidad o aplicaciones de una marca mediante imágenes cuando se solicite ese resultado. |
| [industrial-brutalist-ui](frontend/styles/industrial-brutalist-ui/SKILL.md) | `taste-skill-main` | Explorar una dirección industrial/brutalista si el brief la pide; no aplicarla por defecto al dashboard. |
| [gpt-taste](frontend/styles/gpt-taste/SKILL.md) | `taste-skill-main` | Explorar composición y movimiento para una landing expresiva; no imponer AIDA o GSAP a pantallas de operación. |
| [image-to-code](frontend/visual-assets/image-to-code/SKILL.md) | `taste-skill-main` | Implementar una web desde imágenes suministradas o conceptos visuales pedidos; no generar imágenes nuevas si ya existe la referencia objetivo. |
| [imagegen-frontend-web](frontend/visual-assets/imagegen-frontend-web/SKILL.md) | `taste-skill-main` | Preparar imágenes de referencia para secciones web cuando se soliciten conceptos visuales; la imagen no sustituye implementación. |
| [minimalist-ui](frontend/styles/minimalist-ui/SKILL.md) | `taste-skill-main` | Explorar una dirección editorial minimalista solicitada; conservar tokens existentes en ajustes de producto. |
| [full-output-enforcement](engineering/full-output-enforcement/SKILL.md) | `taste-skill-main` | Completar implementaciones y archivos sin placeholders cuando se requiere código completo; no obliga a imprimir todos los archivos en la respuesta. |
| [redesign-existing-projects](frontend/review/redesign-existing-projects/SKILL.md) | `taste-skill-main` | Mejorar una interfaz existente con auditoría previa, conservando flujos, rutas y contratos. |
| [high-end-visual-design](frontend/styles/high-end-visual-design/SKILL.md) | `taste-skill-main` | Explorar una dirección visual suave y cuidada si se pide; no impone fuentes, sombras o animaciones ajenas al producto. |
| [stitch-design-taste](frontend/design-systems/stitch-design-taste/SKILL.md) | `taste-skill-main` | Preparar un DESIGN.md semántico para herramientas de diseño; Google Stitch es opcional y requiere una herramienta disponible. |
| [design-taste-frontend](frontend/landing-pages/design-taste-frontend/SKILL.md) | `taste-skill-main` | Diseñar o revisar páginas públicas y pantallas de aplicaciones con dirección artística y reglas específicas. |
| [design-taste-frontend-v1](frontend/landing-pages/design-taste-frontend-v1/SKILL.md) | `taste-skill-main` | Usar el procedimiento v1 de taste solo cuando se solicite compatibilidad con esa variante; alternativa al actual. |
| [design-taste-frontend-v2-experimental](frontend/landing-pages/design-taste-frontend-v2-experimental/SKILL.md) | `taste-skill-main` | Usar la variante histórica experimental de taste solo si se pide ese procedimiento; alternativa al actual y al v1. |
| [frontend-design](frontend/landing-pages/frontend-design/SKILL.md) | `web-design-skills-main` | Diseñar una superficie web con una dirección visual intencional, conservando el stack y los contratos del producto. |
| [responsive-design-web-patterns](frontend/responsive/responsive-design-web-patterns/SKILL.md) | `web-design-skills-main` | Consultar los patrones responsive de web-design-skills como alternativa a responsive-design; no duplicar ambos procedimientos. |
| [web-design-guidelines](frontend/review/web-design-guidelines/SKILL.md) | `web-design-skills-main` | Revisar archivos de UI contra Web Interface Guidelines, con versión/fuente y límites de evidencia explícitos. |
| [theme-factory](frontend/design-systems/theme-factory/SKILL.md) | `web-design-skills-main` | Proponer temas para artefactos o nuevas superficies; no reemplazar la paleta actual sin un encargo de rediseño. |
| [website-rebuild](frontend/landing-pages/website-rebuild/SKILL.md) | `web-design-skills-main` | Planificar o realizar una reconstrucción web amplia si está solicitada; adaptar el proceso completo a Next/React y al alcance autorizado. |
| [local-business-rebuild](frontend/landing-pages/local-business-rebuild/SKILL.md) | `web-design-skills-main` | Preparar una web de negocio local desde contenido del brief; scraping, contacto y publicación requieren alcance propio. |
| [multilingual-business-design](frontend/landing-pages/multilingual-business-design/SKILL.md) | `web-design-skills-main` | Diseñar una web comercial multilingüe con entrevista, identidad e i18n; locales y referencias regionales se eligen por producto. |
| [design-system-generator](frontend/design-systems/design-system-generator/SKILL.md) | `web-design-skills-main` | Producir un DESIGN.md estructurado para un sistema de diseño; partir de tokens y componentes existentes cuando los haya. |
| [landing-page-design](frontend/landing-pages/landing-page-design/SKILL.md) | `ai-design-skills-main` | Diseñar una landing con una oferta, audiencia y CTA: estrategia, copy, SEO, composición y estados; no impone su sistema visual al dashboard. |

El [manifest frontend](frontend/frontend-preservation.json) permite comprobar
bytes de snapshots, cuerpos completos, recursos y procedencia. Ver
[integración frontend](../docs/biblioteca-frontend-ai-toque.md) para selección,
helpers, validación y licencias. Los originales mantienen su idioma; la adaptación
de alcance está en español y es la misma para Claude y Codex.

## Preservación y mantenimiento

La consolidación previa había acotado las guías a Al Toque. Se recuperaron
**174 archivos completos seleccionados** de ambas importaciones, verificando sus hashes contra
los manifests de los respaldos. Los 84 cuerpos permanecen íntegros y 61 recursos
(referencias, templates y documentos de testing) también están activos. Metadata
de plugins y README de origen se conservan como evidencia, sin instalarlos.

El [manifest de la biblioteca](library-preservation.json) registra cada archivo,
snapshot, cuerpo activo y recurso. Los snapshots conservan los bytes recibidos;
los cuerpos activos conservan todo el texto normalizando finales de línea y
agregando únicamente metadata portable y contexto AI Toque. El idioma de los
originales se mantiene. Los 11 perfiles de seguridad conservan su propio
[manifest](security/security-profiles-preservation.json) y snapshots.

No se impone un máximo de líneas ni se recortan las capacidades de las
especialidades seleccionadas. Las exclusiones de plataforma están registradas
en el manifest y no se incluyen como snapshots en esta biblioteca. Un original
con ejemplos de otro framework se adapta al ejecutar la tarea. La preservación
no instala scanners, proveedores o conectores ni prueba sus resultados.

Después de cambiar fuentes, ejecutar desde la raíz:

```text
python tools/skills.py --sync
python tools/skills.py --check
python -m unittest discover -s tools/tests
```

No editar entradas generadas a mano. El sincronizador no elimina ni sobrescribe
entradas ajenas. Los tests comprueban snapshots, cuerpos completos, recursos y
descubrimiento para ambos hosts. `.gitattributes` conserva los bytes originales
en Windows y Linux. Cambiar deliberadamente un cuerpo exige revisar su trazabilidad.

La [guía de la biblioteca completa](../docs/biblioteca-skills-ai-toque.md)
explica restauración, portabilidad y criterios de ejecución. La
[guía de seguridad](../docs/biblioteca-seguridad-ai-toque.md) documenta los
11 perfiles especializados y las piezas de un futuro auditor.

## Procedencia

Las dos primeras importaciones proceden de las guías locales atribuidas a Seth
Hobson y al repositorio `wshobson/agents`. Conservar atribución y [licencia MIT](LICENSE).
Los snapshots registran el material realmente recibido; no se reemplazó por una
versión más nueva descargada de Internet. Los 11 perfiles de seguridad aportados
por el usuario no reciben por inferencia la licencia de las otras guías.

## Fuentes oficiales incorporadas

Se actualizaron `design-taste-frontend` (Taste Skill), `web-design-guidelines`
(Vercel) e `image-to-code` desde sus repositorios oficiales, conservando los
perfiles completos y los snapshots anteriores. Image to Code se trasladó de
`Leonxlnx/taste-skill` a `Leonxlnx/imagegen-skills`; OpenDesign todavía enlaza
su ubicación histórica. No se crean nombres de invocación duplicados.

Se incorporó [awesome-design-md](frontend/design-references/awesome-design-md/SKILL.md)
como biblioteca consultable de referencias DESIGN.md completas de VoltAgent.
Las revisiones y hashes de las cuatro fuentes están en
[upstream-imports.json](frontend/upstream-imports.json).
Las cuatro entradas están disponibles tanto para Codex como para Claude.


### Rendimiento frontend

[optimizing-performance](frontend/performance/optimizing-performance/SKILL.md)
añade medición de Web Vitals, React y bundles. Se recuperó de una revisión
anterior de `thkt/dotclaude` porque Skillfish ya no encuentra ese nombre en la
rama actual. Conserva los originales completos y actualiza FID a INP en la
entrada activa. Procedencia, licencia, revisión y hashes están en
[upstream.json](frontend/performance/optimizing-performance/upstream.json).
