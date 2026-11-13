# Membresías comerciales de LeguiBurger

Implementado en Go/Vue y Supabase. Requiere las migraciones `202611130003_membership_commercial_terms` y `202611130004_membership_access`. Los importes y las credenciales de Mercado Pago de la plataforma todavía no están configurados en el entorno local: no se publicaron precios ni se generaron cobros reales.

## Alcances de dinero y permisos

- Comprador → restaurante: pedidos de comida y cobros con la cuenta de Mercado Pago del restaurante.
- Administrador de restaurante → LeguiBurger: membresía mensual y adicionales por tienda, con la cuenta de Mercado Pago de la plataforma.
- El dueño de la app configura tarifas y consulta los ingresos de membresías; no compra su propia membresía.
- La membresía pertenece a la marca y abarca sus tiendas. Los colaboradores no tienen acceso a facturación.

Los comprobantes comerciales de membresía siguen en `legui_private.billing_*`; no son ventas de hamburguesas ni comprobantes fiscales autorizados.

## Planes definidos

| Plan | Administradores por marca | Tiendas | Terminales operativas por tienda | ERP existente |
| --- | --- | --- | --- | --- |
| Prueba Free | 1 | 1 | 2 | No |
| Clásico mensual | 1 | 1 | 2 | No |
| Premium mensual | 1 | 1 incluida + adicionales contratados | 5 | Sí |

**Colaboradores significa terminales simultáneas de caja/pedidos**, según la aclaración del producto. El límite no es la cantidad de personas registradas o invitadas: los distintos turnos pueden tener sus cuentas. Una sesión operativa de navegador ocupa una terminal, incluso si la misma persona abre otra sesión en otro navegador. Las pestañas que comparten la cookie de sesión cuentan como una sola. Los roles operativos actuales (`cashier`, `employee`, `kitchen`) comparten este cupo; el administrador conserva su función administrativa.

Cerrar sesión, revocarla o desconectarla desde el administrador libera el cupo. También queda libre cuando la sesión vence por 30 minutos sin actividad, conforme a la política de sesión existente. Cerrar una pestaña no implica un cierre fiable del navegador: el cupo puede permanecer reservado hasta cerrar sesión o alcanzar ese tiempo de inactividad. El conteo usa el reloj de PostgreSQL y se serializa por marca para impedir altas concurrentes por encima del cupo.

## Prueba y vencimiento

La prueba se inicia una única vez cuando aparece el primer administrador activo de la marca. Dura 168 horas; su inicio y vencimiento son inmutables. Volver a iniciar sesión, reactivar una cuenta o transferir el responsable no la reinicia.

Las marcas que ya tenían un administrador activo al aplicar la migración reciben siete días de transición desde la migración. No se utilizó su fecha histórica de alta ni se borraron recursos existentes. Los recursos por encima del cupo se conservan; no se pueden agregar tiendas fuera de la capacidad vigente. Las marcas sin administrador permanecen pendientes de activación.

Al vencer la prueba o el período pagado se bloquean las operaciones administrativas y de colaboradores. El administrador conserva cuenta, seguridad, soporte y renovación. El colaborador puede consultar el estado de acceso y su cuenta, sin acceder a facturas. Las experiencias de administrador, colaborador y configuración de la plataforma tienen archivos de vista separados.

La API resuelve la vigencia desde la DB en cada solicitud y la revalida en las mutaciones protegidas; el JWT no almacena un permiso Premium que pueda quedar desactualizado. El SSE de personal vuelve a comprobar la vigencia. Los dueños de la app mantienen sus permisos globales.

Una cancelación de renovación no quita el período que ya se pagó. La vigencia no depende de ejecutar un cron. Los contratos anteriores válidos conservan sus condiciones mediante el tipo `legacy`; no se ofrecen como nuevas membresías desde la API de contratación del restaurante.

Las nuevas compras públicas se detienen si el negocio no tiene acceso operativo. La carta sigue visible y se conservan seguimiento, pagos y conciliación de pedidos existentes.

## Funciones Premium

Se restringen inventario, insumos, recetas, producción, gastos y analítica operativa. La analítica básica conserva ventas, pedidos y clientes; se eliminan los costos y resultados Premium de las respuestas, incluidos detalles anidados. La carta básica y la gestión de personal siguen disponibles.

Los CRUD anteriores de productos por tienda y extras también validan los campos de stock para evitar eludir `/inventory`. Los planes básicos usan disponibilidad manual: los saldos de inventario se conservan para una futura mejora de plan, pero no impiden vender ni se descuentan durante el plan básico.

La administración de equipo existente es alta, permisos y asignaciones. Proveedores, compras, nómina, asistencia y otros componentes de un ERP completo requieren sus propios desarrollos; no se presentan como módulos ya terminados por agregar un plan Premium.

## Tarifas y contratación

El dueño de la app tiene `/platform/plans` para crear planes, guardar tarifas en borrador y publicarlas o retirarlas. Un borrador con importes cero no puede publicarse ni contratarse. Publicar una tarifa comercial requiere importe mensual positivo y, en Premium, adicional por tienda positivo.

Las tarifas son versionadas e inmutables: corregir un importe exige otra versión. Retirar una tarifa no modifica contratos anteriores. Publicar una nueva versión comercial retira la oferta anterior para nuevas contrataciones, sin afectar sus contratos. La selección pública muestra la última versión publicada de cada plan.

Total mensual:

`precio base + (tiendas contratadas - 1) × adicional por tienda`

La DB calcula el importe contratado con `numeric`; el contrato tiene `quantity=1` y `purchased_locations` separado. Checkout, mandato recurrente, cargo, conciliación y analítica de ingresos usan ese total, sin multiplicar dos veces los adicionales. Crear un local no aumenta silenciosamente un mandato ni dispara un cobro. El contrato necesita cubrir todos los locales activos.

El alta de checkout o la autorización de un mandato no habilitan un período pago. La conciliación existente verifica cuenta, referencia, entorno, importe y pago aprobado antes de habilitar la membresía. Un pago sandbox no habilita acceso comercial de producción.

Cambiar de tarifa o de cantidad de tiendas de un contrato existente requiere un acuerdo nuevo. El flujo actual conserva el contrato y devuelve conflicto ante una selección incompatible; no implementa prorrateos ni cambia mandatos existentes sin confirmación.

## Endpoints y pantallas

- `GET /api/membership`: resumen de acceso del propio negocio, también disponible al vencer.
- `GET /api/billing/terminals`: terminales activas de la marca del administrador.
- `DELETE /api/billing/terminals/{id}`: desconexión auditada, con clave de operación y validación de marca.
- `POST /api/platform/billing/plans/{id}/prices`: tarifa comercial y borrador; `membership_tier`, `extra_location_amount`, `published`.
- `PATCH /api/platform/billing/prices/{id}/publication`: publicar o retirar sin cambiar condiciones.
- `POST /api/billing/subscribe`: incorpora `purchased_locations`; devuelve la misma contratación ante reintentos compatibles.
- `/platform/plans`: configuración del dueño de la app.
- `/admin/membership`: estado y renovación del restaurante.
- `/employee/membership`: aviso al operador, sin facturación.
- `/payments`: experiencia del restaurante con selección de tiendas, total mensual y liberación de terminales.

## Validación

`database/tests/membership_commercial_terms.sql` comprueba condiciones inmutables, importes, tiendas contratadas, límites de terminales, liberación de cupo y prueba que no se reinicia. El runner verifica preservación de las filas anteriores y revierte todas las fixtures.

`TestMembershipIntegration` prueba endpoints, sesiones, límites, recuperación del administrador, separación entre marcas, stock y métricas Premium. La suite de Mercado Pago usa proveedor simulado para comprobar también el total con tiendas adicionales. Las suites operativas previas usan contratos históricos de prueba explícitos, exclusivamente dentro de transacciones con rollback.

`tools/test-membership-browser.mjs` cubre borrador/reintento/publicación, total mensual, renovación del administrador, bloqueo de colaboradores, rutas Premium y disposición móvil/escritorio. No representa una prueba en un teléfono físico ni un cobro real.

## Qué tomar de Al Toque

Referencias locales de la revisión:

- `database/v2/supabase/migrations/20261015130000_trials_and_referral_fulfillment.sql`: prueba persistida, siete días normales y protección contra reinicios por reintento o después de una activación pagada. La bonificación por referidos de ese producto no forma parte del alcance de LeguiBurger.
- `database/v2/supabase/migrations/20261022020100_subscription_no_renew_until_paid.sql`: conserva el acceso ya pagado al cancelar renovación; aplica el vencimiento y no concede acceso por una autorización de cobro sin pago. Expone el resumen de plan, vigencia y estado.
- `database/v2/supabase/migrations/20261025120300_bot_access_gate.sql`: una decisión de acceso comercial consumida por cada canal, incluida la prueba vencida.
- `apps/backend/app/normalized/presenters/access.py`: presentación del estado para la web y separación entre dueño e integrante.
- `apps/web/src/lib/subscription-session.ts`: operación de compra/cancelación identificada, recuperación por clave y bloqueo entre pestañas; una respuesta incierta no crea otra contratación.

La adaptación mantiene Go/Vue y los roles existentes. Los nombres Free/Individual/Equipo, créditos, módulos fiscales y reglas de referidos de Al Toque no se trasladan como condiciones comerciales de este producto.
