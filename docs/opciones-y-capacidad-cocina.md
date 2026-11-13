# Opciones por producto y capacidad de cocina

## Revisión de la DB y reutilización

La inspección remota confirmó que ya existían `public.extras` (precio, stock, costo y estado) y `public.order_detail_extras` (extra elegido, nombre/precio/costo congelados y cantidad por unidad). Al revisar había cero extras cargados. No existía relación entre extra y producto ni grupos de elección.

Se conserva ese inventario y su historial. Las tablas privadas `product_option_groups` y `product_option_items` agregan la relación local/producto/extra, mínimos, máximos, orden y un precio opcional para ese producto. Las claves compuestas impiden vincular extras de otra tienda.

## Configurar variantes o combos

En **Combos y opciones** (`/product-options`), elegí un producto asignado al local. La búsqueda y paginación recorren la carta completa. Se pueden crear extras usando el CRUD existente, y administrar stock/costo desde las pantallas habituales.

Activá las opciones específicas del producto para definir hasta seis grupos. Cada grupo admite hasta treinta extras y entre cero y cinco elecciones por unidad:
- Tamaño: mínimo uno, máximo uno; Simple incluido y Doble con un adicional.
- Bebida o acompañamiento de un combo: mínimo uno, máximo uno.
- Adicionales opcionales: mínimo cero, máximo hasta cinco, con cantidades por extra.

Un precio vacío hereda el precio habitual del extra. Cero significa incluido en el precio base. Los componentes se modelan como extras del local: no se agregan precios, inventarios ni descuentos paralelos. Para un combo se usa un producto base del catálogo con sus grupos de componentes. Un extra puede aparecer una sola vez dentro de cada producto, y reutilizarse en otros productos.

Los productos mantienen por defecto los extras generales existentes. Activar opciones específicas sin grupos significa que no acepta extras. Un grupo obligatorio sin stock suficiente en sus opciones deja el producto no disponible en la carta; las opciones opcionales agotadas no bloquean el producto base.

El comprador y caja usan el mismo selector con opciones obligatorias, incluidas, cantidades y precio adicional. Se puede editar una selección desde el carrito; al hacerlo se consultan las reglas actuales del producto y se avisa si alguna opción dejó de estar ofrecida. Variantes del mismo producto se conservan como líneas distintas en mostrador.

## Validación y datos de venta

GET/PUT `/api/menu/products/{id}/options` requiere responsable y ámbito del local. La API verifica producto/marca/local y pertenencia de cada extra. Los clientes envían únicamente IDs y cantidades. Cotización y creación de pedidos públicos y privados validan grupos, límites, stock y precios en servidor.

El precio especial se congela en el campo de precio ya existente del extra del pedido. Se agrega `option_group_name` como snapshot inmutable para conservar etiquetas al modificar la configuración. Cocina, comprobantes y seguimiento reutilizan esas elecciones. Los extras incluidos también consumen stock y tienen costo, aunque su precio cobrado sea cero. Los cupones siguen aplicándose al precio base, sin descuento sobre extras o envío.

Pedir lo mismo usa las reglas y precios actuales, avisa si una elección dejó de estar admitida y permite completarla mediante Cambiar opciones. No reutiliza precios anteriores ni crea un pedido sin confirmación.

## Capacidad de cocina

En **Atención del local** se configuran:
- Pedidos simultáneos: cero significa sin límite.
- Minutos adicionales por pedido en cola: cero conserva la estimación base.
- Tope de demora adicional, para limitar el incremento.

La carga incluye pedidos de producción pendientes, aceptados y en preparación de todos los canales. Listos, despachados, entregados, rechazados y cancelados no ocupan lugar. Al llegar al límite se detienen nuevas compras y pedidos de mostrador; la recepción vuelve a habilitarse al liberar capacidad. Bajar el límite por debajo de la carga existente no cancela esos pedidos.

La creación toma un bloqueo transaccional por local y vuelve a leer el cupo después de adquirirlo. El trigger de inserción también impide excederlo; no depende de botones ocultos en el navegador. El bloqueo se comparte entre instancias. Las pruebas verifican que otra conexión no pueda obtener el mismo bloqueo durante la reserva.

La demora se calcula como carga anterior por minutos adicionales, limitada por el tope. Los nuevos pedidos web guardan ese intervalo; editar después la configuración no altera el tiempo ya informado. La pausa manual y los horarios siguen aplicándose.

GET `/api/kitchen/load` ofrece a personal autorizado del local carga, límite y estimación, sin datos personales ni importes. La cocina lo refresca con su SSE existente. La carta pública muestra saturación y estimación, sin publicar el número de pedidos ni datos privados.

Los nuevos límites están desactivados por defecto y los PATCH de clientes anteriores conservan los campos nuevos que no envían. No se crean extras, grupos o límites reales de ejemplo durante la migración. Actualizar el backend antes de habilitar configuraciones por producto.

## Migraciones y pruebas

Aplicadas `202611120008_product_options_kitchen_capacity` y `202611120009_kitchen_queue_index` en el proyecto fijado, con ensayo previo y comprobación de conservación de filas preexistentes. Las nuevas tablas usan RLS y permisos directos revocados. El índice de cola usa local/estado de producción para evitar recorrer el historial completo.

`TestProductOptionsCapacityIntegration` cubre compatibilidad con extras generales, precios incluidos/especiales, elecciones obligatorias, snapshots, ventas/stock, permisos por rol y tienda, reserva de cupo, bloqueo entre conexiones, reactivación y compatibilidad de configuración. Todos sus datos se revierten. `tools/test-options-capacity-browser.mjs` prueba configuración, selección/edición, mostrador y saturación en escritorio/móvil, temas claro/oscuro; se incorpora al CI.
