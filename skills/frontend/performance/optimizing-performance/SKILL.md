---
name: optimizing-performance
description: "Auditar rendimiento frontend de Al Toque: Web Vitals, React, bundles, imágenes y carga diferida, con medición antes/después."
---

## Alcance y adaptación a Al Toque

Usar para auditorías y optimizaciones de rendimiento frontend; las skills de
diseño y accesibilidad cubren otras decisiones. Leer el
[contrato de ejecución](../../../ai-toque-execution-context.md) y las
instrucciones del frontend antes de aplicarla.

- Medir una ruta o interacción concreta antes de cambiar código. Separar
  latencia de red/backend, espera del servidor, descarga de JS y renderizado.
  No atribuir una demora a SQL sin observar el backend.
- Usar la versión instalada de Next/React y su documentación local. Medir
  también la compilación optimizada; la compilación y HMR de desarrollo no
  representan el rendimiento del despliegue.
- Revisar React Profiler, cascadas de peticiones, precarga, caché e invalidación,
  límites servidor/cliente, chunks, imágenes y desplazamientos de layout.
  Mantener autenticación, aislamiento por cuenta/equipo y confirmación real de
  operaciones. No cachear sesiones o permisos globalmente para acelerar vistas.
- Los Core Web Vitals actuales son LCP, INP y CLS. Evaluar el percentil 75,
  separando móvil y escritorio; un ensayo local no acredita métricas de usuarios
  reales. [Fuente oficial](https://web.dev/articles/vitals).
- Los presupuestos de bundle y cantidades de filas del original son orientativos.
  Memoizar o virtualizar sólo si la medición lo justifica; no instalar librerías
  ni convertir todas las rutas de Next a React.lazy por seguir un ejemplo.
- Conservar el stack y los runners del repo. No instalar agentes, plugins,
  herramientas ni hooks por cargar esta skill. Los scripts heredados validan
  documentos, no auditan el rendimiento de la app, y no se ejecutan al cargarla.
- Informar hallazgos con evidencia, cambios y mediciones antes/después. Si no
  se pudo medir un caso, declararlo pendiente sin inventar puntuaciones.

## Procedencia y compatibilidad

Recuperada del repositorio `thkt/claude-config`, hoy `thkt/dotclaude`, en la
revisión `dd663c92a87f8ce0b60889c4f7dd70d83f532ece`, anterior a su retirada.
El comando Skillfish actual no encuentra este nombre en la rama principal.
El [snapshot original](../../../originals/thkt-optimizing-performance/SKILL.md)
preserva cuerpo, metadata, scripts y referencias íntegros. Las condiciones de
procedencia se registran en `LICENSE` o `source-notice.md`, según lo recibido.

La entrada activa sustituye la metadata específica de Claude (agente/fork y
herramientas) por instrucciones portables. No activa delegación automática.
Se actualizaron FID/getFID a INP/onINP y los demás callbacks vigentes, los links
relativos y el ejemplo de ordenamiento para no mutar props. El resto del
procedimiento y las tres referencias permanecen completos.

## Procedimiento original adaptado

# Performance Optimization

## Core Web Vitals

| Metric | Target | Measure                  |
| ------ | ------ | ------------------------ |
| LCP    | <2.5s  | Largest Contentful Paint |
| INP    | <=200ms | Interaction to Next Paint |
| CLS    | <0.1   | Cumulative Layout Shift  |

## Workflow

1. Measure (Lighthouse/DevTools)
2. Identify bottleneck
3. One change at a time
4. Re-measure

## References

| Topic  | File                                                    |
| ------ | ------------------------------------------------------- |
| Vitals | `references/web-vitals.md`          |
| React  | `references/react-optimization.md`  |
| Bundle | `references/bundle-optimization.md` |
