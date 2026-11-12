# Depuración entre procesos

Leer el [contexto del repo](../../../../AGENTS.md). Reproducir con las fixtures
existentes y localizar dónde cambia el resultado esperado.

1. Seguir petición/identidad desde web a API y, si corresponde, evento outbox,
   claim del worker, handler interno y respuesta del proveedor.
2. Inspeccionar estado persistido, deduplicación, lease, journal y logs con
   identificadores sanitizados. No reenviar un pago, factura o mensaje real
   para comprobar si el fallo se repite.
3. Distinguir fallo de configuración, autenticación, disponibilidad, contrato
   y resultado externo incierto. Comprobar URLs públicas/internas y fingerprint
   antes de atribuirlo a código.
4. Probar una hipótesis con un caso mínimo y mocks del adaptador correcto;
   conservar el fallo inicial como regresión cuando aporte valor.

Usar `tools/local_smoke.py`, pruebas del módulo y las fixtures de navegador
cuando cubran el flujo. Recurrir a logs/CI externos solo con conexión disponible.
Reportar causa sustentada, corrección y validación; no presentar una hipótesis
como causa confirmada ni un smoke local como prueba de proveedores.
