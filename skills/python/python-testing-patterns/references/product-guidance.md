---
name: python-testing-patterns
description: "Crear regresiones Python de Al Toque con unittest, mocks y fixtures de API, worker o herramientas."
---

# Pruebas Python

## Alcance y selección

Usar para escribir/mantener regresiones Python de API, worker o herramientas.
Cubre unittest y fixtures; las convenciones Python están en `python-guidelines`.
Una edición de documentos no exige crear pruebas de comportamiento de producto.

Leer el [contexto del repo](../../../../AGENTS.md). Las suites principales usan
unittest; copiar el patrón de test del módulo y el comando de su proceso.

- Cubrir decisiones y fallos observables del caso de uso, no una réplica del
  algoritmo. Escoger escenarios que habrían detectado la regresión.
- Sustituir la dependencia donde la importa el consumidor. En async usar
  AsyncMock/IsolatedAsyncioTestCase conforme a los tests existentes.
- Usar valores ficticios y mocks para Supabase, Meta, MP, ARCA, OpenAI,
  Storage, reloj y esperas; una suite local no debe producir efectos reales.
- Para jobs/proveedores, cubrir reentrega, claim perdido, fallo después del
  efecto, incertidumbre y agotamiento además de la ruta feliz.
- Restaurar entorno, módulos/globales modificados, temporales y tareas para
  que los casos puedan ejecutarse de forma independiente.
- Comprobar aislamiento entre actores y errores de autorización cuando el
  cambio altere acceso, no solo respuestas válidas.
- Si cambia un prompt o una salida del bot, usar casos sintéticos para comprobar
  decisiones, validación y fallback con el modelo simulado. Un mock prueba el
  contrato determinista, no la calidad del modelo: una evaluación real requiere
  muestras y presupuesto autorizados, con costo/latencia medidos aparte.

Ejecutar `python -m unittest discover -s tests` desde API/worker y
`python -m unittest discover -s tools/tests` desde la raíz para herramientas.
No migrar la suite a pytest para aplicar esta skill. Distinguir tests con
mocks, SQL parseado y evidencia del entorno real.
