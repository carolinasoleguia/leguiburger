# Operación en vivo y comunicaciones

## Pantallas

- **Caja y pedidos** (`/orders`): bandeja de pendientes con contador, revisión/aceptación/rechazo y sonido voluntario para nuevos pedidos. Los eventos no borran la selección ni los formularios abiertos.
- **Cocina** (`/kitchen`): preparación en vivo, sin datos personales ni importes del comprador.
- **Atención del local** (`/store-operations`): horarios semanales, turnos que cruzan medianoche, zona horaria IANA, pausa temporal o indefinida y tiempos de confirmación/preparación.
- **Campañas** (`/campaigns`): borradores, vista previa, destinatarios elegibles, envío explícito y contadores de entrega. Acceso exclusivo de responsables de la marca, con selección de un local autorizado.
- **Baja** (`/baja#token=…`): confirmación explícita; el token se retira de la barra de direcciones. Solo retira newsletter de esa marca; conserva personalización y suscripciones a otras marcas.

## Pedidos y SSE

`GET /api/orders/events` usa JWT por encabezado y `X-Tenant-ID`. Valida sesión y local al conectar y cada 45 segundos; reconecta como máximo cada dos minutos. El canal transmite invalidaciones vacías y no contiene contactos o información de otros locales. Caja vuelve a pedir sus DTO; cocina utiliza su proyección limitada. Un único LISTEN por instancia de API distribuye los cambios confirmados mediante el hub existente. Los heartbeats cada 15 segundos no consultan la DB. Hay consulta de respaldo cada 30 segundos sin SSE, o cada dos minutos con SSE; pestañas ocultas desconectan.

`GET /api/orders/inbox` devuelve contador completo de pendientes de producción y hasta los 100 más antiguos. Es exclusivo de caja y responsables; un colaborador de cocina no puede consultar esta proyección.

`POST /api/public/buyer-order/cancel` acepta la cookie de compra o la sesión global de un comprador y su `X-Shopper-Order`. Los enlaces de seguimiento son de solo lectura y no autorizan cancelaciones. Solo se cancela un pedido web pendiente, de producción, sin pago ni checkout incierto/activo. El bloqueo del pedido serializa aceptación/cancelación/pagos. Repetir la cancelación es inocuo.

Los pedidos web nuevos guardan `pending_deadline`, el intervalo de preparación y, al aceptarlos, una hora estimada. El plazo predeterminado es 15 minutos, configurable entre 5 y 60; preparación predeterminada 20–35 minutos. El worker revisa vencimientos cada 15 segundos, hasta 50 por ciclo, con SKIP LOCKED. Aceptar después del plazo se rechaza aunque el worker aún no haya procesado el vencimiento. Los cobros recibidos y checkouts submitting/review siguen protegidos incluso cuando vence su URL. No se realizan reintegros automáticos.

Los pedidos anteriores a la migración no reciben un vencimiento retroactivo. Los plazos nuevos tampoco cambian al editar la configuración del local.

`GET/PATCH /api/ordering/settings` exige responsable y ámbito del local. La API comprueba la disponibilidad efectiva al cotizar y crear; ocultar botones en el navegador no es la autorización. Horarios desactivados conservan el comportamiento anterior. Una pausa no oculta la carta. Publicación y habilitación de venta siguen siendo necesarias.

## Correo

Configurar `PUBLIC_SITE_URL`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM` y, si corresponde, `SMTP_USERNAME`/`SMTP_PASSWORD` en el entorno del backend/worker. En producción la URL pública debe usar HTTPS. SMTP exige TLS. Las credenciales nunca se envían al frontend ni se incluyen en los reportes. Reiniciar el worker tras configurar SMTP.

Sin SMTP se pueden guardar borradores, pero el envío de campañas devuelve un conflicto claro. Los avisos transaccionales quedan en el outbox cifrado. No se deben usar campañas reales como pruebas.

Los eventos de pedidos se registran en la misma transacción de creación/cambio de estado/pago. El worker genera avisos para compradores con cuenta, email confirmado y pedido vinculado por `shopper_orders`; no usa direcciones arbitrarias introducidas por invitados. Los eventos obsoletos de estado/importe se omiten. La notificación enlaza a Mi cuenta y no transmite una credencial del pedido.

Las campañas incluyen únicamente perfiles activos con email confirmado y consentimiento newsletter de la marca. Las audiencias frecuente (3+ compras pagadas), recuperar (última compra pagada hace más de 30 días) y nuevos (hasta una compra pagada) requieren además personalización. Las estadísticas corresponden a esa marca y producción. Máximo 5000 destinatarios por campaña.

El envío captura una audiencia; el worker vuelve a comprobar cuenta, dirección, marca y consentimiento antes de encolar, y el sender vuelve a comprobarlos antes de SMTP. Cada destinatario recibe un enlace de baja de su marca, almacenado como digest y válido un año. Cancelar una campaña o retirar el consentimiento inhabilita correos pendientes.

Hay una entrada única de outbox por evento, y una por campaña/comprador. El contenido está cifrado y se elimina al entregar o expirar. Se priorizan correos transaccionales. El sender usa leases, hasta ocho intentos, y reintento a los cinco minutos; los contadores no confunden encolado con entregado. La aceptación por SMTP no confirma lectura ni llegada a bandeja de entrada. SMTP no garantiza exactamente una entrega si el proceso cae después de la aceptación remota y antes de guardar el resultado; un correo ya en tránsito no puede retirarse.

## Migración y verificación

Aplicada `202611120005_live_storefront_operations` en el proyecto fijado `ljdebnvzdrlqdxnligvh`, con ensayo previo y comprobación de que las filas preexistentes conservaron sus valores. Tablas nuevas privadas con RLS y permisos de acceso directo revocados.

`TestStorefrontOperationsIntegration` prueba horarios nocturnos, permisos cruzados, cancelación autenticada/invitada, vencimientos, pagos protegidos, tiempos estimados, campañas, consentimiento, deduplicación y baja. Todos sus datos se revierten y nunca invoca SMTP. `tools/test-storefront-operations-browser.mjs` verifica escritorio/móvil y ambos temas, SSE, sonido, conservación del formulario, horarios, revisión de campañas, cancelación y baja. Se agrega al CI.
