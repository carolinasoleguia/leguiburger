# API de operación y analítica

Base local: `http://localhost:8080/api`. En desarrollo, Vite reenvía `/api` al backend.

Todas las rutas de este documento requieren `Authorization: Bearer <token>`. Las mutaciones requieren también `Idempotency-Key` de 1–160 caracteres (`A-Z`, `a-z`, números, `.`, `_`, `:`, `-`). Repetir la misma operación y los mismos datos devuelve la respuesta original y `Idempotent-Replayed: true`; usar la clave con otros datos devuelve `409`.

El backend verifica la cuenta activa y los permisos actuales en PostgreSQL. Desactivar una cuenta o cambiar su rol/asignación invalida el uso de los tokens anteriores, aunque su firma y vencimiento todavía sean válidos.

## Permisos

| Rol técnico | Alcance |
| --- | --- |
| `users.owner` | Administración de LeguiBurger y selección explícita de negocio/local |
| `users.admin` | Marca asignada y sus locales; analítica comercial, pedidos, pagos, gastos e inventario |
| `employees.cashier` | Pedidos y registro de cobros de su local, sin analítica financiera global |
| `employees.employee` | Analítica operativa e inventario de su local, sin importes de ventas ni datos de membresías |

El alias global `super_admin` se reconoce en los controles globales, pero la creación ordinaria de usuarios sigue ofreciendo `owner`/`admin`. Un administrador de restaurante no obtiene acceso a `/api/platform/*`.

Para operación por local se usa `X-Tenant-ID`. Un empleado puede omitirlo para usar su local asignado. No se acepta elegir otra marca/local ajeno mediante headers o query strings.

## Analítica

| Método y ruta | Respuesta |
| --- | --- |
| `GET /analytics/restaurant` | Pedidos, ventas, cobros efectivos, reintegros, deuda actual, ticket promedio, costos, márgenes, productos, sucursales, clientes, estados y período anterior |
| `GET /analytics/operations` | Producción, stock bajo/no disponible y pedidos en preparación, sin datos financieros |
| `GET /analytics/platform` | Altas de negocios/locales, uso, suscripciones, cobros de membresías, deuda, MRR/ARR y cancelaciones |

Filtros: `from=AAAA-MM-DD&to=AAAA-MM-DD`, ambos días inclusivos, hasta un año. Sin filtros se usa el mes actual hasta hoy. La respuesta especifica `from`, `until_exclusive` y `timezone`. Se usa la zona del local o Buenos Aires para vistas de marca/plataforma.

El dueño de restaurante puede omitir `X-Tenant-ID` para ver toda su marca. El administrador de plataforma debe proporcionar `brand_id` o un local cuando consulta analítica de restaurante.

Los importes están separados por moneda. Las ventas se agrupan por fecha de la venta; los cobros por la fecha efectiva del pago. `previous_period` tiene la misma duración que el intervalo solicitado. El ticket promedio usa pedidos vendidos únicos, no cantidad de líneas.

`gross_margin` y `estimated_operating_result` son `null` si faltan costos de alguna línea vendida. El resultado operativo resta los gastos registrados, no es una declaración de ganancia fiscal definitiva. `has_financial_data=false` distingue un negocio sin actividad financiera de un saldo real de cero.

En plataforma, las cifras monetarias y de uso real incluyen negocios clasificados como `production`, con entorno `production`. Los registros totales y los no clasificados se informan por separado. `active_mrr` normaliza contratos activos y vigentes: importe anual / 12. No equivale al efectivo cobrado. La tasa de bajas usa las suscripciones activas al inicio del período y devuelve `null` si no hay una base histórica para calcularla.

## Pedidos y pagos del restaurante

| Método y ruta | Operación |
| --- | --- |
| `GET /orders` | Listado paginado del local |
| `POST /orders` | Crear pedido con precios y extras consultados en el catálogo del servidor |
| `GET /orders/{id}` | Pedido, versión, estado y líneas |
| `PATCH /orders/{id}/status` | Cambiar estado con versión esperada |
| `GET /orders/{id}/payments` | Cobros/reintegros del pedido |
| `POST /orders/{id}/payments` | Registrar un cobro o reintegro manual |

Creación:

```json
{
  "channel": "counter",
  "environment": "production",
  "items": [
    {"product_id": "UUID", "quantity": 2, "extras": [{"extra_id": "UUID", "quantity": 1}]}
  ]
}
```

`customer_id`, `shipping_method_id` y `shipping_address` son opcionales. El precio y el costo de envío se toman de la base; el cliente HTTP no puede imponer totales. La moneda del catálogo actual es ARS. Las cantidades de extras son por unidad de producto. De 1 a 50 líneas, hasta 500 unidades por línea.

Cambio de estado:

```json
{"status":"accepted","expected_version":2}
```

Flujo: `pending → accepted → preparing → ready → dispatched → delivered`. Desde `ready` se puede pasar directamente a `delivered`. Un pendiente puede rechazarse; los pedidos no entregados pueden cancelarse con un motivo. Los estados terminales no vuelven a abrirse.

Al aceptar se congelan costos y se genera una venta por línea. Si el producto controla stock terminado, se descuenta ese stock; de lo contrario, se consumen los insumos de su receta local. Los extras controlados se descuentan según sus cantidades. Se bloquean los insumos/recetas mientras se calcula el consumo y el costo. Los costos no cargados permanecen desconocidos.

Los pedidos `sandbox` registran su propio flujo de prueba sin alterar el stock físico ni entrar en las cifras de producción.

Pago manual:

```json
{"kind":"charge","amount":"9800.00","method":"cash"}
```

Reintegro:

```json
{"kind":"refund","original_payment_id":"UUID","amount":"9800.00","method":"cash"}
```

Métodos manuales: `cash`, `transfer`, `card`, `other`. `settled_at` es opcional; por defecto se registra la hora de la operación. Este endpoint no afirma haber verificado un cobro online con un proveedor. No acepta datos de proveedor como si fueran evidencia de pago.

## Inventario y gastos

| Método y ruta | Operación |
| --- | --- |
| `GET /inventory` | Existencias, mínimos y versiones por producto/extra/insumo |
| `GET /inventory/movements` | Historial del local |
| `POST /inventory/movements` | Movimiento atómico, exclusivo de dueños/administradores |
| `PATCH /inventory/{kind}/{id}/settings` | Mínimo y costo del insumo/extra; el costo de producto proviene de su receta |
| `GET /expenses` | Gastos operativos del local |
| `POST /expenses` | Registrar gasto |
| `POST /expenses/{id}/void` | Anular gasto con motivo, conservando el registro |

Movimiento:

```json
{"item_kind":"product","item_id":"UUID","quantity_delta":"5","expected_version":3,"reason":"purchase"}
```

Tipos: `product`, `extra`, `supply`. Productos/extras requieren cantidades enteras; insumos admiten hasta cuatro decimales. Motivos: `purchase`, `production`, `sale`, `return`, `waste`, `adjustment`. Compras/devoluciones son positivas; ventas/mermas negativas. Stock insuficiente o versión desactualizada devuelven `409`.

Configuración de extra:

```json
{"minimum_stock":"5","unit_cost":"350.00"}
```

Gasto:

```json
{"category":"rent","amount":"100000.00","currency":"ARS","incurred_at":"2026-11-10T12:00:00-03:00","description":"Alquiler"}
```

Categorías: `payroll`, `rent`, `utilities`, `processing_fees`, `delivery`, `marketing`, `maintenance`, `other`. No se suman nuevamente compras de ingredientes como gasto operativo si su costo ya está incluido en el costo de venta.

## Administración de LeguiBurger

| Método y ruta | Operación |
| --- | --- |
| `GET /platform/businesses` | Negocios registrados, clasificación y cantidad de locales |
| `PATCH /platform/businesses/{id}` | Clasificar negocio y registrar activación/origen de adquisición |
| `GET/POST /platform/billing/plans` | Consultar/crear planes |
| `POST /platform/billing/plans/{id}/prices` | Crear versión inmutable de precio y condiciones |
| `GET/POST /platform/billing/accounts` | Cuenta contratante de marca completa o de un local |
| `GET/POST /platform/billing/subscriptions` | Consultar/crear suscripciones |
| `PATCH /platform/billing/subscriptions/{id}` | Cambiar estado/período con `expected_version` |
| `GET/POST /platform/billing/invoices` | Consultar/emitir cargos comerciales de membresía |
| `GET/POST /platform/billing/invoices/{id}/payments` | Cobros y reintegros de membresía |
| `POST /platform/billing/invoices/{id}/credits` | Crédito comercial de membresía |
| `GET /platform/billing/webhook-events` | Consultar notificaciones recibidas, cuando se integre un proveedor |

Clasificación:

```json
{"data_classification":"production","activated_at":"2026-11-10T12:00:00-03:00","acquisition_source":"referido"}
```

Precio de un plan, sin fijar precios comerciales predeterminados:

```json
{"currency":"ARS","amount":"15000.00","interval_months":1,"billing_unit":"brand","trial_days":14}
```

`interval_months` admite 1 o 12. `maximum_locations` y `maximum_collaborators` se conservan como condiciones opcionales del plan; todavía no bloquean el uso del sistema. `billing_unit` es `brand` o `tenant`. Las versiones anteriores no se sobrescriben. No se permite facturar simultáneamente una marca completa y sus locales por separado.

Cuenta:

```json
{"brand_id":"UUID","currency":"ARS","contact_email":"administracion@example.com"}
```

Para cobrar por local se agrega `tenant_id`; debe pertenecer a la marca indicada. Su identidad y moneda son inmutables.

Suscripción:

```json
{"account_id":"UUID","plan_price_id":"UUID","environment":"production","quantity":1,"status":"active","period_start":"2026-11-10T00:00:00-03:00","period_end":"2026-12-10T00:00:00-03:00"}
```

El importe y las condiciones provienen de la versión de precio elegida. Estados iniciales: `incomplete`, `trialing`, `active`. Una prueba necesita que el precio tenga días de prueba configurados. Crear manualmente una suscripción activa es una acción del administrador de plataforma; no es evidencia de un pago confirmado por proveedor.

Cargo:

```json
{"subscription_id":"UUID","period_start":"2026-11-10T00:00:00-03:00","period_end":"2026-12-10T00:00:00-03:00","due_at":"2026-11-17T00:00:00-03:00","description":"Membresía mensual"}
```

El importe se calcula desde el contrato, multiplicado por la cantidad. No se duplica un cargo de la misma suscripción y período. Son cargos comerciales, no comprobantes fiscales autorizados por ARCA. Los pagos usan el mismo DTO manual de pedidos, en tablas independientes. Para un crédito se envía `{"amount":"...","reason":"..."}`.

Las renovaciones automáticas, prorrateos y callbacks de Mercado Pago/otro proveedor necesitan una integración verificada. No se publicó un webhook que permita acreditar dinero a partir de datos sin verificar.

## Paginación, importes y errores

Los listados nuevos admiten `limit` (1–100, predeterminado 50) y `offset` (0–10000). Los listados financieros de plataforma/pedidos aceptan `environment=production|sandbox`, con producción como predeterminado.

Los DTOs monetarios aceptan números decimales o cadenas decimales, sin notación exponencial, NaN o Infinity; importes con hasta dos decimales. Los cálculos se hacen en PostgreSQL `numeric`, sin convertir dinero a `float64` en el backend. Se conservan los números decimales en las respuestas JSON; un consumidor que opere con cifras grandes debe utilizar aritmética decimal.

Errores: `400` entrada inválida, `401` sesión inválida/revocada, `403` sin permiso, `404` recurso inexistente, `409` conflicto/duplicación/concurrencia, `422` regla de integridad, `504` tiempo agotado. Los errores no devuelven SQL, credenciales ni información de otros negocios.
