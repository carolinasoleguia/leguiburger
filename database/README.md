# Base de datos y migraciones

Proyecto remoto de esta sesión: `ljdebnvzdrlqdxnligvh` (LeguiBurger).

Las migraciones incrementales viven en `database/migrations`. El historial aplicado y el checksum SHA-256 de cada archivo quedan en `legui_private.schema_migrations`. Un archivo aplicado no se modifica: una corrección se agrega en otra migración.

El runner usa `DATABASE_URL` del entorno o del `.env` local. No necesita `supabase login`. Verifica el host/usuario del pooler contra el proyecto esperado y, después de instalarlo, el contrato de identidad de la propia base.

Ensayo completo, sin conservar cambios:

```powershell
go run ./cmd/migrate --expect-project ljdebnvzdrlqdxnligvh
```

Aplicación:

```powershell
go run ./cmd/migrate --expect-project ljdebnvzdrlqdxnligvh --apply
```

Cada ejecución toma un bloqueo de migración y comprueba las filas preexistentes de `public` y `legui_private`: cuenta y digest de sus columnas originales. El reporte local contiene hashes y metadatos, no las filas ni las credenciales. Las filas de prueba se crean en un savepoint y siempre se revierten. Un ensayo revierte también el esquema y el historial de migraciones que estaba probando.

`resources/migration.slq` es un archivo histórico que contiene `DROP TABLE`. No es el mecanismo de actualización del proyecto.

## Bloques instalados

| Migración | Contenido |
| --- | --- |
| `202611100001` | Estados de pedidos/pagos, separación de entornos, relaciones por local, pagos y reintegros, estados históricos y gastos |
| `202611100002` | Versiones y movimientos de stock; saldo anterior/posterior; historiales automáticos de precio y costo |
| `202611100003` | Cuentas, planes versionados, suscripciones, cargos, cobros y créditos de membresía; actividad y auditoría de plataforma |
| `202611100004` | Recibos de operación para reintentos, integridad de balances y protección de snapshots de ventas |
| `202611100005` | Integración con el backend, costos desconocidos explícitos, versión del pedido y actividad desde los CRUD existentes |
| `202611100006` | Identidad de la base e índices de consultas; control de versión aplicado a cambios efectivos de stock |

Los estados se poblaron porque los catálogos estaban vacíos. Los precios y stocks existentes se conservaron; sus snapshots iniciales arrancan en la fecha de migración, sin fabricar movimientos pasados.

Las marcas existentes conservan su clasificación `unclassified`. El administrador debe distinguir `production`, `test` e `internal` antes de incluirlas en métricas de uso y membresías reales. No se crearon planes, precios comerciales, suscripciones ni cobros ficticios.

## Acceso

Las migraciones `202611110009_menu_and_cash_shifts` y
`202611110010_collaborator_role_compatibility` agregan publicación explícita de
la carta, categorías y el registro privado de turnos/movimientos de efectivo;
también retiran un check legado que impedía el rol de cocina, conservando el
control estricto de roles de colaboradores. La guía de [menú y caja](../docs/menu-y-caja.md)
explica la compatibilidad de cobros anteriores y las nuevas reglas de arqueo.

Los datos operativos sensibles nuevos tienen RLS habilitado y no se conceden permisos de acceso directo a `anon`, `authenticated` o `service_role`. La facturación y las auditorías de la plataforma viven en `legui_private`, fuera de los endpoints REST públicos de Supabase.

La aplicación usa el backend Go y su autenticación propia. El backend es quien valida el usuario activo, el rol, la marca y el local antes de consultar con su conexión confiable a PostgreSQL. No se deben entregar al navegador las credenciales de esa conexión.

La migración `202611120001_buyer_checkout` agrega recepción optativa de pedidos,
contacto del comprador, entrega, preferencias por producto y capacidades privadas
de compra de 24 horas. Extiende el actor de checkout de Mercado Pago únicamente
para el propio pedido del comprador. Ver [experiencia de compradores](../docs/experiencia-compradores.md).

## Reglas operativas

- Un cobro confirmado es inmutable. Se corrige con un reintegro, no editando o borrando el registro.
- Un pago de prueba no puede pertenecer a un pedido/cargo de producción. Los identificadores externos son únicos por proveedor y entorno.
- Los totales cobrados se concilian desde sus registros de pago. No se escriben importes cobrados arbitrarios en pedidos o cargos.
- Una devolución de membresía y un crédito comercial son acciones diferentes. Emitir un crédito permite devolver dinero sin inventar una deuda nueva.
- Los cambios de suscripción quedan versionados y auditados. Los precios se cambian creando una versión nueva.
- Los movimientos de inventario conservan cantidades anteriores/posteriores, versión y origen. El backend nuevo exige versión esperada y clave de operación para ajustes manuales.
- Los cambios directos de stock que hacen los CRUD antiguos también generan historial, pero los clientes que requieran protección de reintentos/concurrencia deben usar el endpoint de movimientos.
- Aceptar un pedido registra una venta por línea y consume stock en la misma transacción. Cancelarlo después requiere devolver primero el dinero cobrado; el stock se repone desde los movimientos originales, no desde una receta que pudo haber cambiado.
- Un costo desconocido se guarda como `NULL`. Los informes no presentan un margen como conocido cuando faltan costos.

## Validación

```powershell
go test ./...
go test ./internal/operations -run TestDatabaseIntegration -count=1 -v -args -db-integration
```

La segunda prueba es explícita y apunta únicamente al proyecto LeguiBurger indicado en el test. Hace llamadas HTTP al handler dentro de una transacción externa; las transacciones de los endpoints usan savepoints y al finalizar se revierte todo. No genera pedidos, ventas ni cobros reales persistentes.

La integración con un proveedor de pago requiere validar firmas/notificaciones y consultar el pago al proveedor. La tabla de webhooks está preparada para deduplicar ingresos; una notificación por sí sola no acredita dinero ni habilita una membresía.

La migración `202611120003_shopper_accounts` agrega cuentas privadas de compradores globales, sesiones/acciones de correo, vínculos de compras, preferencias auditadas y cupones por marca/sucursal con usos. No agrega un rol a usuarios/colaboradores ni importa datos comerciales. Ver [cuentas de compradores](../docs/cuentas-compradores.md).

`202611120004_shopper_personalization` agrega contacto opcional del comprador y personalización por marca desactivada por defecto, con historial de consentimiento. La identidad de cada compra sigue en `shopper_orders`; no agrega DNI ni duplica el perfil en ventas.

La operación en vivo, horarios y campañas está documentada en [operacion-en-vivo.md](../docs/operacion-en-vivo.md); requiere la migración `202611120005_live_storefront_operations`.

Comandas, zonas de delivery, favoritos, recompra y disponibilidad por recetas: [guía de uso](../docs/comandas-delivery-y-recompra.md). Este flujo requiere hasta `202611120007_untracked_menu_availability`.

Opciones por producto y capacidad de cocina: [guía de uso](../docs/opciones-y-capacidad-cocina.md), migraciones hasta `202611120009_kitchen_queue_index`. Se reutilizan los extras y snapshots existentes.

La migración `202611130001_public_menu_slugs` agrega direcciones públicas estables por marca y sucursal. Ver [cartas públicas y protección de pedidos](../docs/cartas-publicas-y-abuso.md).

La migración `202611130002_checkout_security` agrega contadores de riesgo, incidentes y permisos de verificación de compra privados. Ver [seguridad de compras y activación de Turnstile](../docs/seguridad-de-compras.md).

Las migraciones `202611130003_membership_commercial_terms` y `202611130004_membership_access` agregan tarifas mensuales por marca, tiendas adicionales, prueba de siete días y terminales simultáneas por local. Ver [membresías comerciales](../docs/membresias-comerciales.md).

`202611130005_membership_changes` agrega cotizaciones privadas e inmutables para ampliar membresías activas, sin modificar el contrato ni ejecutar cobros. La política de aplicación del cambio sigue pendiente.
