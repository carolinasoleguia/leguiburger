# Analítica de LeguiBurger: datos disponibles y propuesta

Este documento conserva el diagnóstico previo a las migraciones. Para ver el esquema instalado y las operaciones implementadas, consultar [Migraciones](../database/README.md) y [API de operación y analítica](api-operaciones.md).

Auditoría: 10 de noviembre de 2026. Se revisaron la base real configurada en `.env`, el backend Go y el dashboard y consultas SQL del proyecto local Al Toque. Las consultas remotas se ejecutaron en una transacción de solo lectura. No se aplicaron migraciones ni se modificaron registros.

## Estado comprobado

La base tiene 21 tablas de aplicación, todas en `public`. En los esquemas accesibles no se encontraron otras tablas de negocio, suscripciones o cobros de membresías.

| Entidad | Registros actuales |
| --- | ---: |
| Marcas | 4 |
| Locales | 5 |
| Productos generales | 6 |
| Asignaciones de productos a locales | 2 |
| Usuarios de administración | 2 |
| Colaboradores | 2 |
| Reportes de producción | 2 |
| Pedidos, líneas de pedido, ventas y clientes | 0 en cada tabla |
| Insumos, recetas, extras y promociones | 0 en cada tabla |
| Historiales de precios y costos | 0 en cada tabla |
| Catálogos de estados de pedido y de pago | 0 en cada tabla |

Son cantidades de registros: no prueban que haya 4 negocios reales, 5 restaurantes operando o clientes pagando. Todavía no hay datos para determinar esas últimas condiciones. No se encontraron triggers de aplicación que generen ventas, historiales o movimientos automáticamente.

El backend tiene módulos de marcas, locales, usuarios, empleados, clientes, productos, catálogo por local, insumos, recetas, extras, envíos y producción. No tiene módulos ni rutas de pedidos, ventas, pagos, suscripciones o analítica. La existencia de una tabla no implica que el sistema ya la esté alimentando.

## Audiencias y permisos

| Audiencia del producto | Representación actual | Alcance propuesto |
| --- | --- | --- |
| Administración de LeguiBurger | `users.role = owner`; auth también contempla `super_admin` | Negocios de la plataforma, membresías y uso agregado |
| Dueño del restaurante/marca | `users.role = admin`, asociado a `brand_id` | Ventas y operación de sus locales; comparación entre sucursales |
| Encargado de local | No hay un permiso diferenciado completo | Operación y ventas de los locales asignados |
| Colaborador | `employees.role = employee`, asociado a `tenant_id` | Trabajo y producción del local asignado |

Conviene conservar la distinción entre los nombres técnicos actuales y las etiquetas visibles. No basta con renombrar los dashboards: cada consulta debe validar el alcance en el backend. Los colaboradores no deberían obtener automáticamente acceso a ingresos, márgenes ni facturación de membresías. Hace falta definir permisos para ver ventas, costos, exportaciones y comparativas.

La autenticación contempla `super_admin`, pero algunos controles y la creación de usuarios solo admiten `owner`/`admin`; ese rol no debe considerarse completamente habilitado sin unificar los permisos.

## Dashboard del restaurante

El dueño debería poder elegir período y sucursal, con comparativa contra el período equivalente anterior. Las semanas y días deben calcularse con la zona horaria del negocio, inicialmente Buenos Aires.

| Indicador o gráfico | Datos existentes | Qué falta para mostrarlo correctamente |
| --- | --- | --- |
| Ventas de hoy, semana y mes | `orders.created_at`, importes del pedido; `sales.sale_date` e importes por producto | Implementar pedidos/ventas y definir cuándo se reconoce una venta; no hay registros todavía |
| Pedidos totales y ticket promedio | `orders` y líneas con cantidad y precios | Flujo de pedidos, estados definidos y total final coherente; excluir anulados según la definición del indicador |
| Evolución diaria/semanal/mensual de ventas | Fechas e importes existentes | Consultas agrupadas, calendario con períodos sin ventas y filtros por sucursal |
| Cobros efectivos y saldo por cobrar | `orders.total_paid` y `payment_status_id` | Libro de pagos con fecha efectiva, importe, método, moneda, referencia y devoluciones; un saldo acumulado no explica cuándo se cobró |
| Ventas por medio de pago | No existe registro de medio de pago de cada cobro | Tabla de pagos y relación con pedidos; un estado de pago no es un medio de pago |
| Estado actual de pedidos | `orders.order_status_id` y tabla lookup | Poblar catálogo de estados e implementar transiciones de pedidos |
| Pedidos aceptados/cancelados | Estado actual previsto | Definir estados y denominador de la tasa; no presentar el indicador como conversión de visitantes |
| Tiempos de preparación y entrega | Solo fecha de creación; `shipping_methods.estimated_time` es texto de configuración | Timestamps o eventos de aceptación, preparación, despacho, entrega y cancelación; tiempo estimado no mide tiempo real |
| Productos más vendidos | `order_details.product_id`, `quantity`; `sales.product_id`, `quantity` | Alimentar ventas y sumar unidades; ofrecer ranking por unidades y por importe por separado |
| Ventas por sucursal | `tenant_id` y relación local/marca | Consultas con autorización; aclarar si se comparan ventas o cobros |
| Clientes nuevos y recurrentes | `customers.created_at`, `orders.customer_id` | Pedidos identificados y definición de primera compra; pedidos anónimos no permiten medir recurrencia |
| Producción por jornada | `production_reports` con fecha, medallones y panes | Ya se puede consultar y graficar; actualmente solo hay dos reportes |
| Stock actual y productos no disponibles | `tenant_products`, `extras`, `supplies` | Se puede mostrar el estado actual cuando se carguen datos; definir mínimos configurables para alertas |
| Consumo, reposición y mermas | Solo saldos de stock actuales | Movimientos fechados con tipo, cantidad, motivo, unidad y vínculo con venta/producción; actualizar un saldo no conserva historial |
| Costo y margen bruto de productos vendidos | Recetas e insumos; `sales.frozen_cost_unit`, `quantity`, `total_net_revenue` | Poblar recetas/insumos y congelar costos al vender; definir costos de extras y tratamiento de envío/descuentos |
| Ganancia neta | No hay gastos operativos | Registrar compras/gastos, comisiones, costos laborales y otros egresos; el margen de producto no representa ganancia neta |

Ventas, cobros y facturación fiscal son conceptos diferentes. Para comprobantes fiscales reales también hacen falta entidades de comprobantes, numeración, autorización y notas de crédito; no existe ese módulo actualmente.

`sales` tiene una fila por producto, no necesariamente por pedido. Para contar pedidos hay que usar pedidos únicos. Tampoco corresponde sumar simultáneamente totales del pedido y de sus ventas, ni repetir el costo de envío por cada producto. La forma de generar esas filas debe establecer esas reglas.

Para clientes, contar una sola compra dentro de una ventana no prueba que el cliente sea nuevo: hay que revisar su primera compra en todo el historial. Actualmente los clientes pertenecen a cada local; una comparación de recurrencia de toda la marca necesita definir cómo identificar a la misma persona entre sucursales.

## Dashboard de administración de LeguiBurger

| Indicador | Situación de la base | Datos o implementación necesarios |
| --- | --- | --- |
| Negocios y locales registrados | Disponible con `brands`, `tenants`, `created_at` | Definir negocio/marca versus sucursal; excluir pruebas internas cuando se identifiquen |
| Nuevos negocios/locales por semana o mes | Disponible por fecha de alta | Consulta agregada; distinguir registro de activación y de primer pago |
| Locales habilitados/deshabilitados | Disponible con `tenants.active` | No usar el flag como prueba de uso reciente o de suscripción paga |
| Restaurantes que usan la app, DAU/WAU/MAU | No hay historial de actividad | Eventos de acciones útiles, con negocio, local, usuario y fecha; definir actividad, no solo sesión abierta |
| Conversión de alta a primer uso/primer pedido | No hay hitos de onboarding ni pedidos operativos | Fechas/eventos de activación, primer producto, primer pedido y primer cobro |
| Restaurantes de pago y en prueba | No hay planes ni suscripciones | Cuenta contratante, plan, vigencia, estado de prueba y estado de suscripción |
| Cobrado por membresías por semana/mes | No hay cobros de plataforma | Facturas/cargos y pagos de membresía, con fecha efectiva y devoluciones |
| MRR/ARR | No hay contratos ni períodos de facturación | Precio y moneda contratados, intervalo, vigencia y cambios de plan; normalizar anual a mensual, separar pagos únicos y pruebas |
| Deuda por membresías y pagos fallidos | No hay cargos ni intentos de cobro | Importes exigibles, vencimiento, pagos aplicados, estados e intentos |
| Cancelaciones, churn y retención | No hay historial de suscripciones ni actividad | Eventos de suscripción y fecha efectiva de baja; cohorts por alta/activación; distinguir deshabilitación administrativa de abandono |
| Ingreso medio por negocio de pago | No existe base de membresías | MRR dividido por negocios pagadores, con la misma moneda y período |
| Locales/productos/pedidos por negocio | Inventario organizativo parcialmente disponible | Agregados autorizados; pedidos y ventas deben empezar a registrarse |
| Popularidad de funciones de la app | No hay eventos de uso | Eventos de negocio para catálogo, producción, pedidos y reportes |
| CAC y rentabilidad del servicio | No hay adquisición ni costos de plataforma | Origen de adquisición y gastos comerciales/infraestructura; no deducirlo de facturación de membresías |

El dinero que vende un restaurante no es ingreso de LeguiBurger. Debe haber registros separados para ventas gastronómicas y cobros de la plataforma. Un pago anual recibido hoy tampoco equivale a MRR de ese importe.

## Información a agregar o completar

### Comercio gastronómico

1. Definir y poblar los estados de pedidos y de pago. Hoy ambos catálogos están vacíos y sus IDs son obligatorios en pedidos.
2. Implementar el flujo de pedidos y ventas sobre las tablas que ya existen, con transacciones y protección contra duplicaciones.
3. Añadir pagos/cobros del restaurante y devoluciones: pedido, importe, moneda, medio, estado, fecha efectiva y referencia externa. Mantener coherente `total_paid`.
4. Añadir eventos de estado del pedido: estado anterior/nuevo, fecha y responsable. Si se quieren tiempos por puesto o colaborador, también registrar asignaciones/turnos, sin atribuir automáticamente todo el tiempo del pedido a una sola persona.
5. Registrar movimientos de inventario y mínimos configurables; conservar cantidades y unidades. Hacer que cambios de costo/precio alimenten los historiales existentes: hoy los servicios actualizan valores directamente y no se detectaron triggers que registren el historial.
6. Completar captura de costos: recetas, insumos y costo congelado por venta. Agregar gastos solo si se va a mostrar ganancia neta.
7. Registrar canal de venta si se quieren distinguir mostrador, delivery, web o aplicaciones externas; actualmente no existe ese campo.

### Plataforma LeguiBurger

Los siguientes nombres son una propuesta conceptual, no tablas ya creadas:

- `billing_accounts`: quién contrata y paga. Definir si una cuenta cubre una marca con varios locales o si se cobra por local.
- `plans` y versiones de precios: plan, precio, moneda, intervalo y límites; conservar el precio que contrató cada cuenta.
- `subscriptions` y `subscription_events`: pruebas, inicio, renovación, cambios, suspensión y cancelación efectiva.
- `billing_invoices`/cargos y `billing_payments`: período cobrado, vencimiento, importe, pago efectivo y devoluciones. Separados de los pagos de pedidos.
- `billing_webhook_events`: eventos externos identificados de forma única para evitar duplicar cobros o renovaciones al reintentar una notificación.
- `activity_events`: acciones relevantes con identificador de negocio, local, actor y fecha. Guardar lo necesario para medir uso; no copiar datos personales o contenido de clientes al registro de analítica.
- Hitos de onboarding y clasificación de cuentas de prueba/internas, para distinguir crecimiento real de datos de desarrollo.

No hacen falta tablas que almacenen cada número del dashboard al principio: las métricas se pueden calcular desde estos registros operativos. Los resúmenes periódicos se justifican después si el volumen lo requiere.

## Qué tomar del dashboard de Al Toque

Se revisaron `overview-view.tsx`, sus gráficos, tarjetas de equipo y el read model `v2_dashboard_overview` definido en la migración `20261106700000_dashboard_consultations_include_canceled.sql`.

- Presupuestos del mes → pedidos del período.
- Pendientes de respuesta → pedidos pendientes de aceptación/preparación, según su definición.
- Cobrado del mes → cobros reales del restaurante, con fecha de pago.
- Tasa de aceptación → pedidos aceptados sobre pedidos respondidos; no visitas convertidas en compra.
- Consultas diarias → pedidos creados por día; si se quieren visitas/consultas sin pedido, hacen falta eventos adicionales.
- Presupuestos pagados por mes → evolución de ventas o de cobros, eligiendo y explicando la fecha utilizada.
- Servicios más presupuestados → productos más vendidos, por unidades o importe.
- Estados y distribución → estado de los pedidos.
- Clientes nuevos/recurrentes → primera compra y repetición de clientes identificados.
- Presupuestos recientes → últimos pedidos, importes y estado.
- Inventario/mapa → stock actual y, si se desarrolla delivery, su operación. El esquema no registra ubicación ni seguimiento de repartidores.
- Créditos, trial y plan → pertenecen a la membresía de LeguiBurger. No existen todavía en esta base y no deben copiarse como si estuvieran implementados.

Hay dos detalles del modelo de Al Toque que no conviene trasladar por el nombre: su gráfico de importes agrupa presupuestos por fecha de creación, no por fecha efectiva de cobro; y la clasificación de clientes usa la cantidad de presupuestos en la ventana analizada, no necesariamente la primera compra histórica. Su componente de gráfico recibe `comparativo`, pero eso cambia el color de la barra: no incorpora una segunda serie del período anterior.

## Orden de implementación

1. Dashboard operativo real con las entidades y los reportes de producción existentes. Administración de plataforma: altas de marcas/locales y configuración, sin llamarlos pagadores o usuarios activos.
2. Captura de pedidos, ventas y pagos; recién entonces pedidos por período, ventas semanales, cobros, ticket promedio, ranking de productos y clientes recurrentes.
3. Cuenta contratante y sistema de membresías: cargos/pagos, suscripciones e historial. Habilitar ingresos de LeguiBurger, pagadores, deuda, MRR y cancelaciones.
4. Eventos de uso, estados e inventario: activación, retención, tiempos operativos y mermas.

Las consultas del dashboard deben separar alcance de marca/local, aplicar permisos y diferenciar ausencia de datos, funcionalidad todavía no implementada y un cero real.

## Hallazgo adicional

Durante la auditoría se detectaron credenciales incrustadas en `cmd/seed/main.go`. Se retiraron del código: ahora el script requiere las variables de entorno indicadas en `.env.example` y no escribe contraseñas en los logs. No se ejecutó el seed ni se cambiaron las credenciales remotas.
