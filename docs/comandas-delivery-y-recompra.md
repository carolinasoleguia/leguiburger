# Comandas, delivery por zonas y recompra

## Comandas y comprobantes

Caja ofrece Imprimir comprobante e Imprimir comanda desde el detalle del pedido. Cocina ofrece Imprimir comanda por cada pedido. Se abre una vista específica; Imprimir abre el diálogo del navegador. El contenido tiene 72 mm de ancho útil y está preparado para papel de 80 mm: hay que seleccionar el formato de papel de la impresora. No hay impresión silenciosa ni integración con hardware propietario.

GET `/api/orders/{id}/ticket?kind=receipt|kitchen` exige JWT y un local autorizado mediante X-Tenant-ID. El comprobante completo requiere caja o responsable. Cocina y colaboradores solo pueden consultar la comanda, que no contiene nombre/contacto/dirección ni importes del comprador. Los productos, cantidades, extras y notas provienen de los snapshots del pedido, conservando lo comprado aunque luego se edite el catálogo.

## Cobertura de delivery

Atención del local enlaza a Zonas de delivery (`/delivery-zones`). Solo responsables pueden crear/editar zonas o habilitar su uso. Cada zona reutiliza un método de envío del local y agrega códigos postales cubiertos, descripción y compra mínima. Se admiten hasta 50 zonas por local, 100 códigos por zona; no se permiten códigos repetidos entre zonas activas mediante este flujo. Pausar una zona retira su método de las opciones activas.

La validación por zonas está **desactivada por defecto** para conservar los envíos existentes. Para usarla hay que cargar al menos una zona activa y habilitarla. No se inventan zonas, tarifas ni áreas reales durante la migración. Al estar activada, se rechaza el uso directo de métodos legacy en el checkout público: se exige zona y código postal, ambos verificados en servidor.

Se aceptan códigos de cuatro dígitos o CPA completos argentinos (por ejemplo B1896ABC); se normalizan al código base de cuatro dígitos. La cobertura se valida por código postal declarado. No hay geocodificación ni verificación de que la dirección escrita corresponda al código; tampoco polígonos o distancia por GPS.

La compra mínima se calcula sobre productos y extras **después del descuento**, sin incluir envío. Las tarifas siempre se consultan en DB. Cambiar una tarifa exige revisar de nuevo la cotización. Pedidos confirmados guardan zona, nombre y código postal, más el costo de envío ya existente: editar la configuración no altera esas compras.

Endpoints privados:
- GET/PATCH `/api/delivery/settings`.
- GET/POST `/api/delivery/zones`.
- PATCH `/api/delivery/zones/{id}`.

Todos los cambios usan la infraestructura de idempotencia y auditoría. La configuración toma el bloqueo del local para serializarse con cotización/creación; no puede referenciar un método o zona de otro local.

## Favoritos y repetir pedido

Las cuentas globales de compradores pueden guardar favoritos en distintas marcas/locales. Cada favorito identifica comprador, local y producto; el permiso se obtiene exclusivamente de la cookie HttpOnly del comprador. No se acepta un shopper_id enviado por el cliente. Hay un límite de 200 favoritos por cuenta; guardar/quitar es idempotente. Mi cuenta agrega la pestaña Mis favoritos. Las cartas retiradas y productos inactivos no se exponen por esta lista; todavía es posible eliminar sus favoritos.

GET `/api/shopper/favorites` admite paginación y filtro por tenant_id. PUT/DELETE `/api/shopper/favorites/{tenant}/{product}` opera únicamente sobre la cuenta actual. Un comprador puede marcar productos agotados para volver más adelante.

Pedir lo mismo aparece en pedidos entregados y pagados, cancelados o rechazados. POST `/api/shopper/orders/{id}/repeat` verifica propiedad y disponibilidad del local; devuelve un **borrador**, nunca crea un pedido ni cobra. Recalcula precios, conserva notas/cantidades de extras, informa cambios de precio y quita productos o extras no disponibles. Los descuentos y el envío anteriores no se reutilizan.

La carta importa el borrador mediante un ID de pedido sin credenciales en la URL. El comprador revisa cambios, entrega, cotización actual y confirma con el flujo habitual. Un pedido en curso del navegador sigue teniendo prioridad y bloquea iniciar otro; un pedido terminado permite una nueva sesión de compra. Un fallo de red conserva el borrador.

## Disponibilidad e ingredientes

El catálogo público, favoritos y mostrador calculan la disponibilidad con la misma fuente que el descuento de stock al aceptar:
- Producto con track_stock: unidades terminadas del local.
- Producto sin track_stock y con receta local: capacidad según ingredientes y cantidades de receta.
- Sin control de stock ni receta local: conserva la disponibilidad manual existente, sin inventar capacidad.

Ingredientes inactivos o insuficientes dejan el producto agotado. La cotización suma la demanda de ingredientes compartidos por productos diferentes y líneas repetidas; no basta con que cada hamburguesa sea posible por separado. La aceptación vuelve a verificar antes de consumir stock. Los pedidos pendientes no reservan unidades: puede ser necesario rechazar uno si otro pedido se acepta primero y consume lo disponible.

Caja y responsables tienen Disponibilidad de carta (`/menu-availability`) para pausar/reactivar productos del local. Reactivar no repone inventario: puede seguir agotado si faltan ingredientes. GET `/api/menu/availability` y PATCH `/api/menu/products/{id}/availability` están limitados por rol, marca y local. Cocina no adquiere permisos de caja mediante esta pantalla.

## Base y validación

Migraciones `202611120006_restaurant_commerce_experience` y `202611120007_untracked_menu_availability` aplicadas al proyecto fijado, con ensayo y comprobación de conservación de datos previos. Nuevas tablas privadas con RLS y permisos directos revocados. No se cambian las migraciones anteriores.

`TestCommerceExperienceIntegration` verifica precios, demanda compartida, falta de ingredientes, pausa, favoritos por cuenta, cobertura/mínimos, historial de envío, propiedad al repetir y proyecciones de impresión. Todas las fixtures se revierten. `tools/test-commerce-experience-browser.mjs` prueba escritorio/móvil y temas claro/oscuro, errores/reintentos, favoritos, cambios en la recompra, cobertura y vista de impresión; no acredita una prueba en impresora física.
