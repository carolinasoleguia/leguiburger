# Espacio operativo

Adaptación de los patrones de Al Toque a Vue/Go: navegación por rol, formularios,
errores con reintento, estados vacíos, paginación, temas y disposición móvil.
Conserva la identidad de hamburguesería y usa las operaciones reales de Supabase.

## Pantallas

- `/admin`: ventas, pedidos, ticket promedio, cobros, gastos, resultado estimado,
  comparación anterior, ventas diarias y productos más pedidos. Fecha/local/moneda
  seleccionables. Los costos desconocidos se muestran sin estimar ganancias.
- `/owner`: ingresos de membresías, MRR contractual, negocios nuevos/activos,
  suscripciones, deuda y actividad. No mezcla ventas de restaurantes con ingresos
  de la plataforma ni incorpora negocios de prueba/internos/sin clasificar.
- `/inventory`: existencias y movimientos de productos, extras e insumos; compras,
  producción, desperdicios, devoluciones y ajustes. Mínimos y costos para gestores;
  costos de productos derivados de recetas. Las escrituras exigen versión y clave
  de idempotencia. Buscar, bajo mínimo y tipo filtran todo el inventario del local
  en la base, antes de paginar. Los filtros permanecen operables durante la carga.
- `/expenses`: alta y anulación con motivo e historial; importes decimales exactos.
- `/orders`: pedidos de mostrador, cliente opcional, carta paginada, aceptación,
  cambios de estado, cobros/devoluciones manuales y acceso a Mercado Pago.
  Registrar un cobro manual supone dinero ya recibido; no envía una transacción
  bancaria. `/cash` agrega turnos de caja y arqueos de efectivo, con enlace desde
  esta pantalla. La búsqueda de la carta del mostrador se realiza en el servidor.
- `/kitchen`: hasta 100 pedidos activos de producción, ordenados por antigüedad;
  aceptado → preparando → listo, refresco cada 30 segundos. Colaboradores no ven
  clientes, direcciones, importes ni pagos en esta API.
- `/customers`: búsqueda y edición, pedidos y última compra por cliente/local.
- `/notifications`: bandeja personal, lectura individual/todas y paginación.
  El worker genera alertas de stock, pedidos, pagos, membresía y equipo, además de
  nuevos negocios para plataforma. Se deduplican por fuente/versión y se filtran
  por asignación actual de marca/local, además de la identidad destinataria.
- `/onboarding`: verifica identidad, local activo, carta disponible y equipo.
  La conexión de Mercado Pago es opcional. Completar exige comprobaciones del
  servidor; no marca pasos arbitrarios ni habilita cobros sin proveedor configurado.
- `/platform/health`: señales del worker, cola de correo/webhooks y clasificación
  de negocios para los informes. Sólo plataforma puede consultar o clasificar.

## Seguimiento privado

`POST /api/orders/{id}/link` crea un enlace de lectura válido por siete días.
`DELETE /api/orders/{id}/link` revoca todos los enlaces del pedido. Ambas operaciones
requieren rol de caja/gestión, local autorizado e `Idempotency-Key`.

El secreto aleatorio viaja en el fragmento del enlace y se elimina de la barra al
abrir. `/api/public/order` lo recibe exclusivamente como bearer de capacidad y sólo
devuelve estado, productos/cantidades, total y cobrado. No contiene identidad del
cliente, domicilio, referencias del proveedor ni permite mutar el pedido o iniciar
pagos. Al recargar se debe volver a abrir el enlace original. No es un JWT de sesión.

## Cuenta y seguridad

El ingreso actual requiere usuario y contraseña, sin segundo factor.
`AUTH_REQUIRE_PLATFORM_MFA=true` permite reactivar TOTP para plataforma.
JWT de acceso de 15 minutos, refresh HttpOnly rotativo, revocación, vencimiento por
inactividad y autorización por rol/recurso siguen activos en ambos modos.

La misma política de contraseña (12–72 bytes, sin espacios extremos) se aplica al
alta/edición de usuarios y colaboradores, invitaciones, recuperación y cambio.
Las claves existentes siguen siendo válidas para iniciar sesión.

`POST /api/auth/email/change` requiere contraseña actual y SMTP configurado. Envía
capacidades distintas a ambas casillas; `POST /api/auth/email/confirm` sólo completa
el cambio después de ambas aprobaciones. Conserva el correo anterior hasta entonces,
vence a los 30 minutos y revoca todas las sesiones al completarse. Un cambio de
credenciales, una solicitud posterior o un correo ya ocupado invalidan la operación.

## Worker y observabilidad

Modo local predeterminado: `WORKER_MODE=embedded`, tareas dentro del proceso API.
Despliegue separado: API con `WORKER_MODE=external` y otro proceso:

```powershell
go run ./cmd/worker
```

Ambos procesos necesitan la misma DB y claves de cifrado/JWT/MP. Usan las mismas
colas, repositorios y arrendamientos persistidos. Los correos se reclaman mediante
lease antes de SMTP, sin mantener una transacción de DB durante la conexión de red.
SMTP es entrega al menos una vez: una caída después de enviar y antes de guardar
la confirmación puede repetir un correo; los enlaces siguen siendo de un solo uso.

Los ciclos de notificaciones publican heartbeat cada 15 segundos. El panel marca
una señal obsoleta después de dos minutos; ese heartbeat no prueba entrega SMTP ni
disponibilidad de MP. El panel muestra colas; los logs de cada worker registran fallos
sin secretos. SMTP y Mercado Pago deben configurarse para usar sus servicios reales.

`GET /health/ready` verifica la conexión DB. Cada respuesta tiene un request ID
generado por servidor; logs HTTP con método, patrón de ruta, estado y duración,
sin body, query, tokens ni datos personales. API/worker aceptan apagado con señales.

## Migraciones y verificación

`202611110002_workspace_operations` y `202611110003_workspace_scope_guards` agregan
tablas privadas con RLS/grants revocados, capacidades con FK de local/pedido,
bandejas por destinatario/alcance, preparación, cambios de correo y leases de correo.
Se aplicaron con identidad del proyecto y comprobación de conservación de datos.

CI toma Go desde `go.mod` (1.26.11), ejecuta toda la suite, vet, verificación de
módulos, govulncheck, detección de credenciales en fuente, npm audit, build y pruebas
de navegador. Las acciones están fijadas por SHA y no reciben secretos de producción.
Los tests de DB son optativos, apuntan al proyecto seleccionado y revierten fixtures.

```powershell
go test ./...
go vet ./...
go test ./internal/operations -run TestDatabaseIntegration -count=1 -v -args -db-integration
go test ./internal/operations -run TestSecurityIntegration -count=1 -v -args -security-integration
go test ./internal/mercadopago -count=1 -v -args -mp-db-integration
npm --prefix frontend run build
python tools/check-source.py
```

`tools/test-workspace-browser.mjs` usa frontend construido, servidor aislado y API
simulada; verifica flujos, permisos de navegación, decimales, reintentos con la misma
clave y anchos 1440/390. Requiere `PUPPETEER_MODULE` y, opcionalmente, `CHROME_PATH`.
Las pruebas de seguridad y pagos existentes se ejecutan con el mismo mecanismo.
