---
name: security-review
description: "Disenar, implementar o revisar seguridad de cambios Al Toque: entradas, permisos, datos, integraciones y herramientas IA. Elegir solo el modo afectado; escaneres en sast-configuration."
---

# Revisión de seguridad

## Alcance y selección

- Usar para diseñar/implementar controles o revisar un diff/flujo potencialmente
  vulnerable: entradas externas, acceso privilegiado, datos privados, archivos,
  proveedores, efectos comerciales o herramientas/contexto de IA.
- Para una revisión de desarrollo, empezar por aplicación y seguir solo las
  fronteras afectadas. Un cambio de labels o estilo no exige auditar todo el repo.
- Aplicar al código propio o generado con IA con el mismo criterio de evidencia;
  el autor del código no acredita una vulnerabilidad ni su ausencia.
- Cubre decisiones y controles. Preparar/ejecutar escáneres corresponde a
  `sast-configuration`; no instalarlos ni añadir gates por cargar esta guía.

Leer el [contexto del repo](../../../AGENTS.md) y elegir el modo solicitado:

Esta es la guía integrada del producto Al Toque. La
[biblioteca completa de AI Toque](../../README.md#especialistas-completos-de-seguridad-de-ai-toque)
conserva 12 especialistas extensos. Si se solicita `security-architect`, AppSec,
cloud u otra especialidad, usar su perfil completo; no sustituirlo por estas
referencias locales. Su aplicabilidad se decide al ejecutarlo, conservando
capacidades que otros productos puedan necesitar.

- [Seguridad de aplicación](references/application-security.md): revisión por
  diff, entradas/sinks, lógica comercial, archivos, HTTP y despliegue afectado.
- [Autorización](references/authorization.md): JWT, taller/objeto, RLS,
  cookies, firmas, OAuth, rutas internas y enlaces públicos.
- [IA y herramientas](references/ai-agents.md): prompt injection, datos del
  modelo, permisos del ejecutor, contexto almacenado y pruebas adversariales.
- [Credenciales y datos sensibles](references/credentials.md): exposición,
  bundles/logs, cifrado, propósito de claves y remediación verificable.
- [Amenazas y requisitos](references/threat-model.md): activos, fronteras,
  STRIDE/árboles de ataque y criterios de aceptación de un flujo concreto.
- [Remediación](references/remediation.md): relacionar hallazgos sustentados
  con cambios, pruebas y evidencia de cierre.

## Procedimiento y resultado

1. Fijar el alcance: diseño, diff frente a una base, flujo existente o corrección.
   Leer productores/consumidores y requisitos; no exigir revisar cada línea del
   repositorio para un cambio acotado. El [deepdive de seguridad](../../../docs/deepdive-seguridad.md)
   aporta riesgos conocidos, no un veredicto sobre un diff posterior.
2. Trazar entrada, identidad, control, recurso/efecto y salida. Elegir referencias
   según esa frontera, comprobar el caso legítimo y el caso sin autoridad.
3. Si se pidió implementación/corrección, hacer el cambio autorizado y validar
   la garantía. Si se pidió análisis, entregar hallazgos/propuestas sin mutaciones
   externas. Usar las suites existentes; mocks no prueban RLS/proveedores remotos.

Por hallazgo indicar archivo/línea, precondición, camino de datos/efecto,
consecuencia, prioridad razonada y certeza (observado, inferido o por verificar),
cambio mínimo y prueba de aceptación. Separar defecto, endurecimiento y duda;
no ocultar una incertidumbre ni inventar un exploit para poder reportarla.
CWE/OWASP ayudan a clasificar cuando se pueda justificar; no fijan la prioridad.

Después de corregir, volver al escenario original y comprobar qué se resolvió,
qué permanece y qué requiere entorno real. Informar controles/checks revisados
y pendientes; un rescan sin alertas no prueba corrección de lógica de negocio.
La revisión no autoriza explotación, SQL remoto, rotación, permisos o avisos
externos. No afirmar cumplimiento normativo, porcentaje de seguridad o estado
remoto a partir de instrucciones, búsquedas estáticas o fixtures.

## Especialidades originales completas

Esta guía integra el contexto de Al Toque; no tiene un original
independiente recortado que deba reemplazarse. Para el modo afectado,
leer el perfil completo correspondiente y el
[contrato común de AI Toque](../../ai-toque-execution-context.md).
Las referencias locales complementan esos perfiles; no los sustituyen.
No cargar todos los modos ni establecer convenciones competidoras.

| Decisión | Perfil completo |
| --- | --- |
| Autenticación e identidad | [auth-implementation-patterns](../auth-implementation-patterns/SKILL.md) |
| STRIDE | [stride-analysis-patterns](../stride-analysis-patterns/SKILL.md) |
| Árboles de ataque | [attack-tree-construction](../attack-tree-construction/SKILL.md) |
| Requisitos verificables | [security-requirement-extraction](../security-requirement-extraction/SKILL.md) |
| Mitigaciones | [threat-mitigation-mapping](../threat-mitigation-mapping/SKILL.md) |
| Modelado de amenazas profundo | [threat-modeling-expert](../threat-modeling-expert/SKILL.md) |
| Implementación segura de backend | [backend-api-security-backend-security-coder](../backend-api-security-backend-security-coder/SKILL.md) |
| Hardening detallado | [security-hardening](../security-hardening/SKILL.md) |
| Dependencias | [security-dependencies](../security-dependencies/SKILL.md) |
| Análisis estático detallado | [security-sast](../security-sast/SKILL.md) |
