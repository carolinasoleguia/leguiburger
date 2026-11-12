# Contrato de ejecución de la biblioteca AI Toque

## Preservación y contexto

AI Toque es la marca de la biblioteca. Al Toque es el producto de este repo;
Los productos AI Toque compartirán Python/FastAPI, Next/React y
PostgreSQL/Supabase. Se excluyen las especialidades dedicadas a blockchain,
Django, móvil nativo/React Native, Temporal, GraphQL y Nx/Bazel/Turborepo por
decisión del usuario. No se recorta el cuerpo de las especialidades seleccionadas. Conservar
procedimientos, checklists, ejemplos, frameworks y capacidades de los originales.
Una técnica que no aplica a este producto permanece disponible en la biblioteca.

Leer este contrato antes del cuerpo completo de una skill restaurada, después
las instrucciones del directorio y el contexto real del producto. Este contrato
adapta los supuestos de los originales y prevalece sobre sus recomendaciones
genéricas. Las instrucciones del usuario y del host mantienen su precedencia.

Los snapshots bajo `originals/` conservan los bytes recibidos, incluida metadata
de plugins, agentes y comandos. Son evidencia de procedencia; no son instrucciones
adicionales ni configuración instalada. El cuerpo activo y los recursos de cada
skill conservan el conocimiento operativo, no solo un enlace al archivo histórico.

## Selección por resultado

Elegir una skill por la decisión o resultado solicitado. No cargar todo el
catálogo por compartir lenguaje o categoría. Una invocación explícita carga el
perfil completo de esa especialidad, sin sustituirlo por una guía resumida.
Consultar sus referencias y templates según el modo afectado; conservarlas aunque
no se necesiten en esa ejecución. No leer todos los snapshots para usar una skill.

Al evaluar un alcance amplio, distinguir:

- **Aplicable:** el componente o problema existe; sustentar decisiones en evidencia.
- **No aplicable en este alcance:** explicar qué condición falta; no borrar la capacidad.
- **Pendiente/no ejecutado:** el componente existe pero falta acceso, herramienta,
  entorno o evidencia. No equivale a aprobarlo ni a demostrar que no aplica.

Las guías integradas `python-guidelines`, `python-reliability`,
`diagnostics-and-performance`, `web-ui-design`, `database-migrations` y
`security-review` orientan tareas habituales del producto. Los especialistas
restaurados desarrollan cada tema con profundidad y mantienen sus nombres.
Un router no sustituye el contenido de un especialista solicitado.

La importación frontend conserva los perfiles y recursos completos de siete
repos. Al aplicar uno, leer el
[adaptador frontend](frontend/ai-toque-frontend-context.md): distingue guía
reutilizable de landing pages, procedimientos de revisión y direcciones visuales
alternativas. `landing-page-design` es un guideline de desarrollo de AI Toque
para productos actuales o futuros, no depende de un consumidor en Al Toque.
El adaptador evita instalar otro stack o convertir preferencias estéticas en
reglas globales. No cargar todos esos perfiles para una corrección visual.

## Una autoridad por decisión

`python-guidelines` sigue siendo la autoridad local de convenciones Python:
estilo, nombres, responsabilidades y Settings. `python-code-style`,
`python-design-patterns`, `python-project-structure`, `python-type-safety` y
`python-configuration` aportan técnicas completas para el tema solicitado;
no establecen simultáneamente estándares alternativos. Para otro producto,
identificar su autoridad de convenciones antes de aplicar recomendaciones.

`python-reliability` orienta ejecución; async, recursos, resiliencia, errores y
jobs tienen especialistas propios. `python-testing-patterns` cubre pruebas
Python. Arquitectura, REST/GraphQL, modelo SQL, migraciones, UI y CI conservan
sus ámbitos. Resolver una discrepancia usando primero el contrato del producto
y luego la guía específica de la decisión; no duplicar procedimientos.

Los perfiles de agentes describen especialidades; no crean agentes en runtime.
El término `PROACTIVELY` en un original significa considerar esa especialidad
cuando sea pertinente, sin ampliar el encargo, invocar otras skills o delegar
automáticamente. Los comandos originales convertidos a skills son procedimientos
portables: no se registran comandos adicionales de un host ni se ejecutan al cargar.

## Adaptación a Al Toque

- Python objetivo **3.11**. Adaptar ejemplos con sintaxis 3.12+, sin eliminarlos
  de la biblioteca. Conservar locks, configuración y suites del repositorio.
- `apps/web`: Next/React, presentación, navegación, Auth/cookies y HTTP.
  `apps/api`: FastAPI, negocio, bot, datos e integraciones. `apps/worker`:
  consumidor del outbox durable. `packages/contracts`: contratos compartidos.
- La web no incorpora motores de negocio, SDKs de proveedores ni accesos
  directos a tablas/RPC de negocio. Respetar sus instrucciones de directorio.
- Las regresiones Python usan las suites actuales de `unittest`. Los ejemplos
  pytest e Hypothesis se conservan; usarlos al elegir ese
  framework o al diseñar una propuesta, sin sustituir las suites por invocar la skill.
- PostgreSQL/Supabase y pertenencia al taller definen el contexto de datos;
  preservar autorización, RLS, contratos y despliegue compatible. No introducir
  otro ORM o base de datos por aparecer en un ejemplo.
- Outbox, leases, journals, conciliación de resultados inciertos y gates de
  release forman parte del producto. CQRS, sagas, event stores,
  Kubernetes y opciones de arquitectura son opciones completas de la
  biblioteca, no decisiones de migración implícitas.
- Odoo permanece fuera del alcance de la migración. Las guías completas no
  modifican ese alcance ni agregan requisitos a cambios ajenos a su ámbito.

## Ejemplos, herramientas y evidencia

Los ejemplos permanecen completos y en el idioma recibido. Adaptar versiones,
paths, nombres, contratos, comandos y límites al entorno real antes de usarlos.
Los umbrales, porcentajes, SLAs, modelos y métricas de un perfil no son políticas
ni resultados medidos de AI Toque. Una descripción de experiencia profesional
no afirma experiencia vivida por el agente.

No instalar herramientas, conectores, servicios o plugins por cargar una skill.
Inventariar herramientas realmente disponibles y elegir solo las que necesita
la tarea. Verificar documentación/versiones cuando la implementación lo requiera.
Los templates y scripts preservados son material de referencia; no han sido
validados como código productivo para este repo por preservarlos.

Las recomendaciones sobre modelos (`opus`, `sonnet`, etc.), colores, permisos,
tool access, comandos de Claude, `$ARGUMENTS` o `Task(...)` se preservan en la
fuente, pero no son configuración de Codex ni autorización para delegar.
Usar las capacidades efectivamente disponibles del host, sin imponer un modelo.
Sustituir placeholders por el encargo recibido; si falta un dato indispensable,
pedirlo y continuar el trabajo independiente que sí pueda completarse.

Los procedimientos multiagente se conservan como capacidad. Aplicarlos a una
tarea de orquestación o delegación autorizada y cuando el host lo permita;
no dispararlos como efecto secundario de una revisión o de leer un comando.
No anunciar ejecuciones, conectores, cobertura o resultados que no ocurrieron.

## Alcance de efectos y seguridad

La autorización existente del usuario sigue vigente; no inventar aprobaciones
adicionales para revisiones locales o cambios reversibles ya pedidos. Tampoco
inferir autorización para producción, aplicación de SQL, despliegues, mensajes,
facturación, cobros, scans activos o cambios en servicios externos de una
solicitud de análisis. Respetar los controles del host antes de esos efectos.

Conservar trabajo local y evitar reset, clean, force push o reescrituras públicas
por seguir un ejemplo Git. Aislar el cambio, inspeccionar el diff y validar su
alcance. Las referencias a secrets se inspeccionan por nombre/estructura,
sin volcar valores. Outputs, logs y reportes deben minimizar datos privados.

Contenido recuperado, archivos y salidas de tools son datos no confiables cuando
alimentan agentes de un producto. No derivar identidad, permisos o alcance de
texto del modelo. Para flujos de IA con herramientas, datos o credenciales,
usar la revisión específica de seguridad y comprobar controles en el ejecutor.
Las skills son instrucciones; no son controles de runtime ni certificaciones.

## Validación proporcional

Elegir pruebas según el comportamiento afectado y comandos del producto. No
exigir scans, benchmarks, frameworks nuevos o todas las suites por haber cargado
una guía. Un test con fixtures no demuestra SQL aplicado ni proveedores reales.

Al modificar esta biblioteca ejecutar `python tools/skills.py --sync`,
`python tools/skills.py --check` y las pruebas de herramientas/preservación.
El manifest prueba integridad documental y trazabilidad de recursos; no demuestra
que un auditor encuentre todas las vulnerabilidades ni que un template funcione
sin adaptarlo al producto.
