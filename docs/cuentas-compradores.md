# Cuentas de compradores, compras entre marcas y beneficios

La compra invitada conserva su cookie privada de 24 horas y no requiere cuenta. La nueva cuenta de comprador es global y separada de usuarios administradores y colaboradores: una persona puede comprar en varias marcas sin tener un rol en sus tiendas. No se agregó un rol privilegiado al JWT existente ni se reutilizó la tabla CRM `customers`, que sigue siendo local de cada sucursal.

## Pantallas

- `/mi-cuenta`: ingreso/registro, historial de todas las marcas, recibos y seguimiento en vivo, cupones, preferencias por marca y perfil/cambio de contraseña. Diseñada para escritorio y móvil, claro/oscuro.
- `/restaurantes`: directorio público de sucursales activas con carta publicada. Permite seguir comprando sin cuenta.
- El enlace **Mi cuenta** en la carta conserva la sucursal para poder vincular un pedido invitado con su cookie, mediante una acción explícita.
- `/marketing`: administradores y plataforma crean cupones, pausan/reactivan su disponibilidad y consultan compradores de su marca. Cocina, cajeros y colaboradores no reciben ese permiso.

## Autenticación y privacidad

La migración `202611120003_shopper_accounts` crea tablas privadas para identidades, sesiones, acciones de correo, vínculos de pedidos, preferencias con historial y cupones/usos. Todas tienen RLS y acceso directo revocado a PUBLIC, anon, authenticated y service_role. No se cargan compradores, cupones ni suscripciones comerciales de ejemplo.

Las contraseñas usan bcrypt y la política existente. La cookie host-only `legui_shopper` es HttpOnly, SameSite=Lax, Secure en producción, limitada a `/api` y con vencimiento fijo de siete días. El secreto de 256 bits se guarda solo como SHA-256 en sesiones revocables. Cada petición verifica cuenta activa, sesión vigente y origen. El navegador no guarda contraseñas, tokens o perfil en localStorage/sessionStorage. La autenticación de compradores no habilita endpoints de administración.

El registro permite comprar antes de verificar el correo. Confirmación y recuperación usan tokens aleatorios de un uso, almacenados como hash, con vencimiento de una hora. Los correos transaccionales usan la outbox cifrada existente y requieren SMTP, origen público y clave de cifrado configurados. Sin SMTP, la cuenta sigue siendo útil para comprar, pero el correo queda sin confirmar; la UI no simula una confirmación o envío. Cambiar/restablecer contraseña revoca sesiones. La recuperación informa una respuesta genérica para evitar enumerar cuentas.

Las compras nuevas realizadas con la cuenta abierta se vinculan al comprador dentro de la transacción del pedido. Los pedidos invitados anteriores solo se vinculan al demostrar posesión de la cookie/capacidad y si no pertenecen ya a otra cuenta: no se buscan por email o teléfono. Los vínculos son inmutables. Iniciar sesión en otro dispositivo recupera el historial mediante la cuenta, sin necesitar las cookies invitadas originales.

`X-Shopper-Order` selecciona un pedido para recibo, SSE y pago. El backend siempre comprueba `shopper_orders.shopper_id`; conocer el UUID no permite consultarlo. Una cabecera de pedido de cuenta inválida nunca recurre a un token invitado. El checkout reutiliza los controles de aceptación, importe del servidor, diario financiero y conciliación de Mercado Pago. Los pedidos de la cuenta no vencen con la capacidad de compra invitada; el acceso requiere una sesión de cuenta vigente. La página muestra estado actual y progreso por etapas, sin inventar una hora estimada de entrega.

Cerrar sesión desde el área de compradores revoca esa sesión y borra las cookies invitadas enviadas al endpoint público de logout, para evitar dejar el recibo a otra persona en ese navegador compartido. No cierra la sesión administrativa.

## Cupones

El administrador crea un código de 3–32 caracteres (letras/números/guion), un porcentaje entero de 1–50%, compra mínima, vigencia de 1–90 días y límite total de usos. Puede limitarlo a la sucursal seleccionada o a todas las de su marca, y a un comprador concreto que ya le compró o a todas las cuentas. No puede seleccionar personas de la base global sin una relación comercial con su marca. La lista de compradores solo incluye nombre, compras en esa marca y consentimiento confirmado; no expone emails ni compras de otras marcas.

Cada cupón admite un uso por cuenta. Se valida cuenta destinataria, marca/sucursal, activación, vencimiento, mínimo y cupo desde PostgreSQL. El descuento se calcula por unidad sobre el precio base, redondeando a centavos igual que PostgreSQL; no descuenta extras o envío. La cotización incluye descuento y digest. Crear el pedido vuelve a validar los datos y bloquea el cupón FOR UPDATE, registrando uso y snapshots en la misma transacción. Reintentar con igual clave no consume otro uso.

`order_details.discount_unit`, `final_price_unit` y `orders.total_discounts` conservan el descuento. La aceptación genera ventas cuyo ingreso neto y descuento coinciden con el cobro, como se verifica con importes enteros y fracciones de centavo en integración. Cancelar un pedido conserva el uso del cupón en esta versión; no hay restitución automática ni descuentos acumulables, envío gratis o importes fijos.

## Newsletters

La preferencia es independiente por marca, no se activa al registrar la cuenta y tiene registro inmutable de cambios. La solicitud puede guardarse antes de confirmar el correo, pero solo se presenta como suscripción habilitada si el correo está confirmado. Se puede desactivar desde la cuenta. Esta entrega no incorpora editor ni envío masivo de campañas: prepara las preferencias y la audiencia confirmada para esa integración posterior. No se enviaron newsletters de prueba ni mensajes a compradores reales.

## Endpoints y comprobación

- Cuenta: POST `/api/shopper/register`, `/login`, `/logout`, `/verify`, `/resend`, `/forgot`, `/reset`, `/password`; GET `/api/shopper/me`. La UI usa POST `/api/public/shopper/logout` para borrar también las cookies invitadas por su Path.
- Área: GET `/api/shopper/orders` (paginado), `/coupons`, `/preferences`; PATCH `/api/shopper/preferences`; POST `/api/shopper/claim`.
- Restaurante: GET `/api/marketing/customers`, GET/POST `/api/marketing/coupons`, PATCH `/api/marketing/coupons/{id}`. Las escrituras conservan Idempotency-Key y ámbito de marca/sucursal.
- Público: GET `/api/public/restaurants`; cotización/creación de pedido aceptan `coupon_code` opcional y siempre revalidan su elegibilidad.

`TestShopperAccountsIntegration -security-integration` usa DB real con rollback y comprueba registro, cookies, ingreso, recuperación, verificación/replay, cambio de contraseña, compras entre marcas, acceso ajeno, destinatarios, cupones/reuso/redondeo, ventas netas, preferencias y logout. Los correos de fixture quedan dentro de la transacción revertida. `tools/test-shopper-account-browser.mjs` cubre las pantallas en 1440/390 y ambos temas; las suites de compra invitada, cookies, SSE, pagos, jerarquía y seguridad continúan siendo parte de CI.
