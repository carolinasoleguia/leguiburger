# Pruebas de navegador

Leer el [contexto del repo](../../../../AGENTS.md) y `apps/web/AGENTS.md`. Partir de
los scripts `apps/web/scripts/test-fiscal-settings-browser.mjs`,
`test-public-ticket-browser.mjs` y `test-budget-editor-browser.mjs`.

- Usar Puppeteer y el patrón de fixture/servidor existente. No introducir
  otro runner para un flujo que ya cubren estos scripts.
- Cubrir resultados visibles: validación, éxito, fallo recuperable, doble
  clic, recarga, autorización y transporte JWT, según el cambio.
- Esperar condiciones observables; evitar pausas fijas y selectores que
  dependan del orden accidental del DOM. Aislar datos entre escenarios.
- Simular proveedores y APIs fuera del alcance del caso. No emitir, cobrar
  ni enviar mensajes reales desde la fixture.
- Cerrar browser y servidores en `finally`; conservar evidencia sanitizada
  en `apps/web/scripts/output/` o `artifacts/`, ambos ignorados por Git.

Ejecutar el script afectado desde `apps/web`; para el cierre completo usar
`node tools/verify.mjs --browser` desde la raíz. Ver requisitos de Chrome y
`PUPPETEER_EXECUTABLE_PATH` en `docs/operacion/despliegue.md`. Declarar que las
fixtures no prueban proveedores, RLS ni límites del hosting real.
