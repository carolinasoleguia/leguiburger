# Mercado Pago: cobros, membresías y conciliación

La implementación adapta los circuitos de Al Toque a pedidos y membresías de
LeguiBurger. Utiliza Checkout Pro con Preferences, como el proyecto de referencia,
y Preapproval para suscripciones. No guarda tarjetas ni envía sus datos al backend.
El adaptador HTTP está separado del libro de pagos para permitir otra API de
checkout en el futuro sin cambiar el contrato contable.

## Dos cuentas y dos libros

- Pedidos: cuenta del negocio conectada mediante OAuth. Puede configurarse por
  marca o por local; la configuración específica del local tiene prioridad.
  La acreditación se guarda en `public.order_payments` y actualiza el saldo del pedido.
- Membresías: cuenta propia de LeguiBurger, identificada por token y collector ID.
  La acreditación se guarda en `legui_private.billing_payments` y se vincula a una
  factura de membresía. Estas facturas son comprobantes comerciales internos;
  no son facturas fiscales autorizadas por ARCA.

Los roles y recursos se validan antes de cada comando. Caja puede crear links de
pedidos de su local. La administración de una marca puede conectar cuentas y ver
los movimientos/revisiones de su negocio. La plataforma administra todos los
negocios; únicamente ella puede devolver dinero de su propia cuenta de membresías.

## Tablas de la migración 202611110001

Todas las tablas están en `legui_private`, tienen RLS y no conceden acceso a anon,
authenticated ni service_role. Solo el backend accede mediante su conexión DB.

| Tabla | Responsabilidad |
|---|---|
| `mp_connections` | Cuenta, negocio/local, ambiente y versiones de conexión/credenciales |
| `mp_oauth_attempts` | Autorización de un solo uso, state/binding con hash y PKCE cifrado |
| `mp_checkouts` | Importe, moneda, recurso y referencia de compra inmutables; diario de envío |
| `mp_provider_payments` | Estado más reciente verificado, bruto, devolución, comisión y neto informado |
| `mp_payment_observations` | Evidencia normalizada e inmutable, sin tarjetas, documentos ni tokens |
| `mp_payment_allocations` | Vínculo único de cada cobro/devolución con un asiento en el libro correspondiente |
| `mp_webhook_inbox` | Recepción duradera, deduplicación, lease y reintentos |
| `mp_reviews` | Inconsistencias y resoluciones con actor y motivo |
| `mp_operations` | Diarios de devolución y cancelación; nunca repetir a ciegas un comando incierto |
| `mp_subscription_mandates` | Mandato remoto, estado observado y cancelación de renovación |
| `mp_scan_cursors` | Ventanas y paginación duraderas para recuperar notificaciones perdidas |

Importes: PostgreSQL numeric y decimales exactos en Go. La API recibe los importes
opcionales como strings, por ejemplo `"amount":"1500.50"`. La comisión, neto y
fecha de disponibilidad quedan en NULL si el proveedor no los informa. No se
inventan retenciones ni se calcula disponibilidad a partir de la fecha de venta.
La UI muestra el neto y la disponibilidad como datos informados por el proveedor.

## Procesamiento

1. Verificar sesión, rol y recurso; calcular importe desde DB. Para pedidos puede
   pedirse un cobro parcial que no supere el saldo.
2. Confirmar el intento y sus términos antes de llamar a Mercado Pago.
3. Marcar el envío en DB y realizar una sola solicitud HTTP fuera de la transacción.
4. Si la respuesta se pierde o es incierta, guardar REVIEW. La conciliación busca
   por la referencia `LEG:UI:<uuid>` y consulta el recurso; no repite el POST.
5. Validar firma HMAC del webhook, coherencia de IDs/tipo y tolerancia temporal de
   cinco minutos; guardar la recepción antes de responder 202.
6. El worker consulta el recurso con el token de la cuenta correspondiente y verifica
   collector, live_mode, moneda, importe, referencia, versiones y fechas.
7. Guardar observación y aplicar una sola vez. Los snapshots atrasados no hacen
   retroceder un pago; los conflictos de versión y los cambios de términos van a revisión.

El GET del proveedor es la prueba de pago. Volver a `/payments/return` no acredita
dinero. Las búsquedas descubren IDs; cada pago se vuelve a verificar individualmente.
El registro global `(environment,payment_id)` impide aplicar un pago a dos cuentas
o a ambos libros. Cada checkout de pedido/factura admite una acreditación; una
suscripción admite distintos cobros periódicos, con facturas separadas.

El worker interno corre cada 15 segundos. Usa `SKIP LOCKED`, claims, vencimiento de
leases y hasta 8 intentos del inbox con espera creciente. Además consulta checkouts
de los últimos siete días y busca pagos por ventanas de actualización, con cursor
y páginas de 50. Los cursores no avanzan si una lectura/verificación falla. Se
puede solicitar conciliación desde el panel; está limitada por actor.

## Suscripciones y devoluciones

Seleccionar un plan crea/reutiliza una suscripción local `incomplete`; contratar el
mandato remoto no la activa. El cobro aprobado genera la factura, el asiento y el
período pagado en una transacción. Cobros superpuestos, pagos sin mandato o cambios
de contrato requieren revisión. El scheduler pasa a `past_due` al vencer el período
sin otro cobro. Cancelar renovación confirma el estado remoto y conserva el período
ya abonado; al vencer se cancela la suscripción local.

Las devoluciones salen de la cuenta receptora original. Primero se guarda el
comando, luego se consulta el pago y sus devoluciones verificadas para crear los
asientos. Se bloquean los reintegros manuales que pretendan descontar un pago MP
sin evidencia del proveedor. Un timeout de devolución se conserva en revisión;
un refund remoto conocido puede conciliarse por ID. Si se perdió su ID, hace falta
revisión operativa y nunca se repite el cargo/devolución para averiguar el resultado.

Los contracargos y estados inconsistentes se conservan para revisión; no se borra
el historial ni se descuenta un saldo de forma arbitraria. Resolver una revisión
registra quién y por qué, y puede volver a encolar la verificación. No fuerza un
`succeeded` ni salta las comprobaciones de importe, cuenta o contrato.

## Configuración

El entorno actual no tiene credenciales MP configuradas. La integración queda
deshabilitada hasta configurar las credenciales propias de LeguiBurger. No se
copian las de Al Toque. El panel muestra el estado de configuración y permite
consultar datos existentes sin habilitar llamadas al proveedor.

Variables en `.env.example`:

- `MP_ENABLED=true`, `MP_ENVIRONMENT=sandbox` para pruebas con proveedor.
- `PUBLIC_SITE_URL` y `MP_PUBLIC_ORIGIN`: el mismo origen HTTPS público, sin path.
  La aplicación actual usa frontend/backend bajo un mismo origen; el callback
  OAuth necesita recibir la cookie que se creó al iniciar la autorización.
- `MP_CLIENT_ID`, `MP_CLIENT_SECRET`: aplicación propia con OAuth/PKCE habilitado.
- `MP_ENCRYPTION_KEY`: 32 bytes aleatorios codificados en base64; `MP_ENCRYPTION_KEY_ID=v1`.
  `MP_ENCRYPTION_KEYS` permite conservar un anillo de claves para leer credenciales
  anteriores al rotar. No retirar una clave mientras haya registros que la usen.
- `MP_PLATFORM_ACCESS_TOKEN`, `MP_PLATFORM_COLLECTOR_ID`: cuenta de LeguiBurger.
- `MP_WEBHOOK_SECRET`: firma de la aplicación MP.
- Para producción también es necesario `MP_LIVE_ENABLED=true` y
  `MP_ENVIRONMENT=production`, con las credenciales correspondientes. El código
  verifica live_mode para impedir mezclar dinero de prueba y real.

Registrar en Mercado Pago:

- Redirect OAuth: `https://<origen>/api/integrations/mercadopago/callback`.
- Webhook de la plataforma: `https://<origen>/api/webhooks/mercadopago/platform`.
  Activar `payment`, `subscription_preapproval` y `subscription_authorized_payment`.
- Los checkouts de negocios incluyen su notification_url por conexión. Las
  notificaciones `merchant_order` firmadas se reconocen sin aplicar dinero.

El intercambio OAuth pide `test_token=true` en sandbox, vincula state al navegador
y a una sesión válida, y cifra access/refresh con AES-GCM y datos asociados a la
conexión. La renovación se serializa por versión; un resultado desconocido exige
reconectar y no puede resucitar una conexión desconectada.

## Endpoints y panel

`/payments` contiene Movimientos, Links de cobro, Cuentas conectadas, Mi membresía
y Revisiones. La plataforma puede crear planes y publicar nuevas versiones de precio.

| Método | Ruta |
|---|---|
| GET | `/api/payments/mercadopago/config` |
| GET/POST | `/api/integrations/mercadopago/connections` |
| POST | `/api/integrations/mercadopago/connections/{id}/authorize` |
| DELETE | `/api/integrations/mercadopago/connections/{id}` |
| GET | `/api/integrations/mercadopago/callback` (callback OAuth) |
| POST | `/api/orders/{id}/checkout` (X-Tenant-ID e Idempotency-Key) |
| GET | `/api/billing/plans`, `/api/billing/my-account` |
| POST | `/api/billing/subscribe` |
| POST | `/api/billing/invoices/{id}/checkout`, `/api/billing/subscriptions/{id}/checkout` |
| POST | `/api/billing/subscriptions/{id}/cancel-renewal` |
| GET | `/api/payments/mercadopago/checkouts`, `/transactions`, `/reviews` |
| POST | `/api/payments/mercadopago/transactions/{id}/refunds` |
| POST | `/api/payments/mercadopago/reviews/{id}/resolve` |
| POST | `/api/payments/mercadopago/reconcile` |
| POST | `/api/webhooks/mercadopago/{connectionId}` (firma MP, sin JWT) |

Los checkouts, devoluciones y cancelaciones requieren `Idempotency-Key`. Las listas
admiten limit/offset; el panel muestra hasta 50 registros por consulta inicial.
La consulta de movimientos distingue pago verificado y aplicado al saldo.

## Pruebas

```powershell
go test ./...
go test ./internal/mercadopago -count=1 -v -args -mp-db-integration
npm --prefix frontend run build
node tools/test-payments-browser.mjs
```

La integración DB verifica el proyecto antes de abrir una transacción que se
revierte; usa un servidor HTTP simulado para MP. No crea cuentas/cobros reales ni
deja tokens, facturas o correos de prueba. Incluye OAuth y separación de negocios,
deduplicación, cobro, devolución parcial, membresía, cancelación, resultado incierto
y conciliación por GET. Las pruebas unitarias cubren cifrado vinculado a la cuenta,
firmas falsas/vencidas, importes exactos, mezcla de ambientes y redirects del proveedor.

La prueba del navegador requiere `PUPPETEER_MODULE` y `CHROME_PATH` locales y usa el
build con respuestas simuladas. Verifica flujos, importes, costos desconocidos,
estado deshabilitado y vista de 1440/390 píxeles. La puesta en marcha con MP real
requiere probar la aplicación propia en sandbox y sus webhooks HTTPS antes de activar producción.

Referencias oficiales:
[OAuth](https://www.mercadopago.com.ar/developers/es/docs/security/oauth/creation),
[Checkout con preferencias](https://www.mercadopago.com.ar/developers/es/reference/online-payments/checkout-pro-preferences/create-preference/post),
[notificaciones](https://www.mercadopago.com.ar/developers/en/docs/checkout-pro-preferences/payment-notifications).
