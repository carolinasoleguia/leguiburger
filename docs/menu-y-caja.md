# Carta pública, búsqueda y turnos de caja

El catálogo de marca y su asignación por sucursal ya existían. Se conservan
`products`, `tenant_products`, los precios locales, las imágenes y la pantalla
de gestión. La vista pública se completa sobre la ruta existente `/catalogo`;
no se crea un segundo catálogo ni se copian productos.

## Carta por sucursal

Desde **Catálogo → Catálogo por sucursal**, seleccionar el local y usar
**Publicar carta**. Se obtiene `/catalogo/{tenantId}`, un enlace para copiar y
un QR PNG descargable. El QR se genera en el navegador con
[node-qrcode](https://github.com/soldair/node-qrcode), sin enviar el enlace a
un servicio de imágenes. En local el enlace apunta al origen local; para
compartirlo fuera del equipo se necesita acceder a una instalación publicada.

La categoría se edita en el formulario de producto existente. Los productos
anteriores conservan la categoría inicial “Hamburguesas”.

`GET /api/public/menu/{id}` no requiere sesión. Devuelve únicamente nombre de
marca/local, dirección, contacto, horarios y productos activos con descripción,
categoría, imagen, precio efectivo y disponibilidad. El stock exacto, costos,
recetas, credenciales, clientes, pedidos y pagos no forman parte de esa proyección.
Un producto sin stock y con seguimiento aparece como no disponible.

La publicación comienza desactivada para todos los locales. Solo un gestor de la
marca o la plataforma puede cambiarla con GET/PATCH `/api/menu/settings`, usando
`X-Tenant-ID`; PATCH requiere `Idempotency-Key`. Retirarla, desactivar el local,
suspender o cerrar el negocio impide nuevas lecturas públicas. No hay preview
privilegiada que permita saltarse esa regla por enviar un JWT o un header.
Un contenido que alguien ya descargó no puede retirarse de su dispositivo.

La carta publicada es pública también para usuarios de otras marcas. Esa
publicación explícita no concede acceso a los endpoints internos del catálogo.

## Búsqueda completa

`GET /api/inventory` acepta `search` (hasta 100 caracteres), `low=true/false`
y `kind=product/extra/supply`. Los aplica dentro del local autorizado antes de
LIMIT/OFFSET y devuelve `total`. Bajo mínimo incluye solo existencias con
seguimiento de stock. Los costos se omiten para colaboradores en esta API.

`GET /api/pos/catalog` también acepta `search`. La búsqueda pública agrega
`search` y `category` a su paginación. Los valores se parametrizan: `%`, comillas
y sintaxis SQL son texto de búsqueda. Cambiar filtros reinicia la página;
no se vuelve a filtrar en el navegador una página parcial de inventario.

## Caja de efectivo

`/cash` está disponible para plataforma, administradores y colaboradores de caja.
Cocina y colaboradores generales no acceden al módulo. Cada operación valida el
local y el recurso; un ID de turno de otro local no concede acceso.

| Endpoint | Acción |
| --- | --- |
| GET/POST `/api/cash/shifts` | Historial y apertura con moneda/efectivo inicial |
| GET `/api/cash/shifts/{id}` | Arqueo calculado desde el registro de movimientos |
| GET/POST `/api/cash/shifts/{id}/entries` | Movimientos e ingreso/retiro con motivo |
| POST `/api/cash/shifts/{id}/close` | Efectivo contado y nota de cierre |

Las escrituras requieren `Idempotency-Key`. Hay un único turno abierto por local
y moneda, compartido por los cajeros autorizados. Aperturas, cierres y cobros en
efectivo se serializan con la misma clave de bloqueo. Un reintento con igual clave
y datos devuelve el resultado anterior, sin duplicar la operación.

Los cobros y reintegros confirmados en efectivo de producción se incorporan
automáticamente al turno abierto, incluso si corresponden a un pedido anterior.
Se usa el momento de registro en el turno, no una fecha histórica de liquidación.
No se incorporan pagos sandbox, tarjetas, transferencias ni Mercado Pago al efectivo.

Para conservar compatibilidad, los cobros anteriores a la primera apertura siguen
en su registro original y no se imputan retroactivamente. Desde la primera
apertura de una moneda, registrar efectivo requiere un turno abierto en ella.
El sistema no exige turnos para otros medios de pago.

Esperado = inicial + cobros − reintegros + ingresos − retiros.
Diferencia = contado − esperado. Un retiro no puede superar el efectivo esperado.
La nota es obligatoria si el cierre tiene diferencia. Los cálculos autoritativos
usan PostgreSQL numeric; el frontend conserva los importes de caja como texto y
acepta coma o punto decimal, sin calcular saldos financieros con float.

Los movimientos no se editan ni se eliminan. Un turno cerrado no admite nuevas
entradas ni cambios de arqueo. Se conservan actores, pagos vinculados y fechas.
Cerrar definitivamente un restaurante requiere cerrar antes sus turnos abiertos.
Los respaldos del negocio incluyen `cash_shifts` y `cash_entries`, con la misma
verificación de reconstrucción en tablas temporales que el resto del respaldo.

## Migraciones y pruebas

Se aplicaron `202611110009_menu_and_cash_shifts` y
`202611110010_collaborator_role_compatibility` al proyecto
`ljdebnvzdrlqdxnligvh`, tras ensayos con rollback y comprobación de valores y
cantidades de filas anteriores. La segunda retira una restricción antigua que
rechazaba `kitchen`; conserva el control que limita roles de colaboradores.

`TestTenantIsolationIntegration` usa handlers/repositorios reales, dos marcas y
tres locales para verificar aislamiento, publicación, búsquedas fuera de la
primera página, reintentos, pagos, devoluciones y cierre de caja. Se revierte toda
la transacción. `tools/test-menu-cash-browser.mjs` comprueba los flujos a 1440 y
390 px con API simulada; se ejecuta también en CI.

La prueba de geometría se amplió a 88 combinaciones vacías/pobladas, escritorio
y móvil, incluyendo caja y colaborador. Se comprobaron CLS 0, anclas estables,
ausencia de overflow y fundido en todas ellas. El selector de local del
colaborador y el historial de caja reservan su lugar durante la carga.
