---
name: api-design-principles
description: "Disenar o revisar contratos HTTP REST de Al Toque: rutas, DTO, paginacion, errores e idempotencia entre consumidores."
---

# Contratos REST

## Alcance y selección

Usar para decidir rutas, DTO, status, paginación y compatibilidad HTTP.
Cubre contrato externo, no estilo/arquitectura de implementación Python.

Leer el [contexto del repo](../../../../AGENTS.md). Esta guía cubre el contrato HTTP;
el estilo, organización e implementación Python siguen las skills existentes.
No renombrar rutas ni reorganizar módulos para aplicar una convención genérica.

1. Localizar ruta en `apps/api/app/api/routes`, caso de uso, adaptador Next y
   otros consumidores, incluidos callbacks y handlers internos del worker.
2. Definir entrada, identidad/ámbito, resultado, errores y semántica de
   reintento. Mantener `/v1` y el contrato vigente salvo cambio explícito.
3. Revisar compatibilidad: campos opcionales/defaults, status que entiende
   el consumidor, cantidades, fechas, nulos y límites de tamaño. Un DTO público
   no debe exponer la fila completa ni depender del nombre de cada columna.
4. Para listados, definir orden estable y paginación; para totales/reportes,
   usar agregados exactos, no una muestra de la primera página.
5. Separar recepción/persistencia de un job de su efecto final. Un ACK del
   webhook no confirma entrega de mensaje, cobro ni emisión. Conservar
   idempotencia y recuperación cuando el cliente repita o pierda la respuesta.
6. Mantener CORS, transporte JWT y secreto interno conforme al llamador;
   reservar diagnósticos privados para rutas autenticadas. Evitar caché
   compartida para recursos privados y enlaces que actúan como capacidades.

Mantener errores de validación/acceso separados de indisponibilidad y efecto
incierto. No descartar eventos con un 200 tras un fallo transitorio ni mostrar
trazas/secretos al browser. La web traduce el contrato a feedback; el worker
decide complete/retry según claim y estado, no solo por una excepción.

Revisar OpenAPI y pruebas de productor/consumidor cuando cambie un DTO/status.
Los tests de API y web deben cubrir el contrato observable. Para implementación
Python o acceso, consultar solo la guía específica que ese cambio necesite.
No añadir GraphQL, otra estrategia de versionado o dependencias de cliente.
