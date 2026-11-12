# Errores en Python

Leer el [contexto del repo](../../../../AGENTS.md). Revisar excepciones del
adaptador y su traducción en rutas/casos de uso antes de añadir un tipo nuevo.

- Validar entradas en la frontera y conservar invariantes del dominio antes
  de hacer I/O. Reutilizar validadores de documentos, fechas y teléfonos.
- Capturar errores que se pueden resolver en ese nivel; preservar la causa
  con `raise ... from error`. No devolver una colección vacía por cualquier fallo.
- Distinguir autenticación/validación, fallo transitorio y efecto incierto;
  la incertidumbre requiere conciliación y no un retry indiscriminado.
- En lotes, identificar resultado de cada elemento sin perder éxitos previos.
  Mantener transacción/idempotencia por la unidad definida en el caso de uso.
- Cerrar recursos incluso cuando falle parseo o cancelación. Sanear el mensaje
  que se persiste en outbox o devuelve al cliente.

Probar fallo antes del efecto, durante la respuesta y después de persistir,
además de mezcla de éxitos/fallos en lotes. Usar unittest y mocks del repo;
no sustituir excepciones de dominio por un esquema global sin necesidad.
