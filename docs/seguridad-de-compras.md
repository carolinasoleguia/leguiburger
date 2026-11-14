# Seguridad de compras

La pantalla `/platform/security` pertenece exclusivamente a `owner` y `super_admin`. Sus endpoints son `GET /api/platform/security/incidents` y `POST /api/platform/security/incidents/{id}/actions`. Un administrador de restaurante o colaborador recibe 403 incluso si fuerza un ID.

## Detección y respuesta

Los contadores compartidos en PostgreSQL usan ventanas de 15 minutos por origen. Requieren verificación adicional a partir de 12 sesiones de compra nuevas, 8 solicitudes inválidas o 8 pedidos nuevos. Los reintentos de un pedido ya persistido no cuentan como compras nuevas. Un cambio de precios o disponibilidad (409) no se considera un intento inválido. Las señales de verificación rechazada también se registran.

Las ráfagas que superan los límites existentes y 30 solicitudes inválidas activan una pausa de compras de 15 minutos. Las decisiones y su vencimiento utilizan el reloj de la DB. Las cartas públicas y el seguimiento siguen accesibles. Los controles no intentan detectar DevTools.

Los incidentes agrupan motivo, contadores, fechas y local. La identidad del origen usa HMAC con `ABUSE_HASH_KEY` o, si falta, `JWT_SECRET`; el panel muestra una red parcialmente oculta y una referencia, no la IP completa, contactos o contenido del pedido. Usar una clave de HMAC propia y estable evita que una rotación de JWT cambie las referencias y bloqueos existentes. No cambiarla sin planificar esa transición.

El dueño de la app puede resolver la revisión, bloquear durante 15 minutos, una hora o 24 horas, y desbloquear falsos positivos. Todas las acciones requieren motivo, usan idempotencia y se auditan con actor en la misma transacción. Resolver un incidente no elimina un bloqueo activo. Desbloquear reinicia las señales de riesgo y permite superar una pausa local previa en otras réplicas; los límites generales de velocidad siguen vigentes. Una nueva actividad abusiva puede producir otro bloqueo.

El worker limpia verificaciones y tokens vencidos, y conserva incidentes hasta 30 días desde su última señal cuando no tienen un bloqueo activo. La auditoría administrativa se conserva por separado. Las tareas de retención no detienen el procesamiento de pedidos si fallan.

## Activar Turnstile en Render

1. Crear un widget **Managed** en [Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/get-started/), con el hostname de producción de LeguiBurger permitido. No requiere cambiar el stack Vue/Go.
2. Configurar en Render `TURNSTILE_SITE_KEY` y `TURNSTILE_SECRET_KEY`. La clave secreta permanece únicamente en el backend. `PUBLIC_SITE_URL` debe contener el origen público real; su hostname se valida contra la respuesta del proveedor.
3. Reiniciar o desplegar el servicio y comprobar que el panel indique **Configurada**. Las compras normales no cargan el widget. Solo aparece cuando el backend exige verificación.

**Sin claves, la verificación no está activa.** Una compra que requiere esa prueba queda en pausa: puede reintentarse cuando venza la ventana o después de que un owner revise y libere el origen. No se habilita un bypass ni se instalan claves de prueba en producción. La pantalla conserva carrito, contacto y confirmación pendiente.

La validación servidor comprueba éxito, hostname, acción `checkout`, contexto ligado a IP/sesión/local y antigüedad de hasta cinco minutos. Un token verificado no puede reutilizarse. El permiso dura cinco minutos y se consume para un único pedido dentro de la transacción de creación. Si falla la operación de negocio, esa transacción revierte el consumo. Repetir la misma compra confirmada devuelve su recibo sin crear otra venta. Un error de red o caída del proveedor no aumenta el riesgo del comprador.

La CSP permite únicamente el origen oficial `https://challenges.cloudflare.com` para el widget. Las credenciales de pruebas y respuestas del proveedor se simulan en las suites; no se contacta Turnstile real desde las pruebas de navegador.

## Validación

- `go test ./...`: verificador, contexto, caducidad, configuración y contratos existentes.
- `go test ./internal/operations -run TestCheckoutSecurityIntegration -args -security-integration`: handlers reales y DB remota fijada, con transacción exterior revertida; proveedor simulado.
- `node tools/test-checkout-security-browser.mjs`: panel, permisos, filtros, motivos, bloqueos, reintentos y responsive.
- `node tools/test-buyer-browser.mjs`: verificación configurada y sin configurar, conservación del borrador y reintentos idempotentes.

Estas defensas reducen abuso y falsos positivos, pero una IP puede ser compartida y una verificación humana no demuestra que una compra sea legítima. No sustituyen la autorización por recurso, controles de stock, revisión del restaurante ni protección de infraestructura.

Referencia: [validación servidor de Turnstile](https://developers.cloudflare.com/turnstile/get-started/server-side-validation/).
