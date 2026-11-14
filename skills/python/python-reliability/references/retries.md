# Resiliencia frente a proveedores

Leer el [contexto del repo](../../../../AGENTS.md) y el adaptador/conciliador del
dominio afectado. Clasificar lectura, escritura idempotente y efecto no repetible.

- Definir timeout en transporte y presupuesto total de operación. Tener en
  cuenta límites de concurrencia y lease durante la espera.
- Reintentar solo errores transitorios con intentos/tiempo acotados; usar
  backoff/jitter y Retry-After cuando el contrato del proveedor lo permita.
  No reintentar validación, credenciales inválidas ni bugs como indisponibilidad.
- Si hubo cobro/emisión/envío y se perdió la respuesta, consultar estado y
  journal. Un timeout no significa ausencia de efecto externo.
- Conservar las claves de idempotencia, números/orden fiscal y propiedad de
  claims. No reiniciar journals ni generar otra solicitud para salir del error.
- Coordinar retries del adaptador con los del outbox para evitar multiplicación
  de intentos y presión de varias réplicas.
- El fallback debe conservar significado: no marcar una operación realizada
  ni confirmar un dato fiscal cuando solo se obtuvo una estimación.

Probar 429, timeout, fallo permanente, agotamiento, éxito tras retry y efecto
incierto. Usar mocks del transporte y tiempo; no incorporar una librería de
retries ni cambiar proveedores para aplicar esta guía.
