# Comprar desde la carta del restaurante

La experiencia pública vive en `/catalogo/:tenantId`, sobre la carta y el QR existentes. No requiere crear una cuenta ni iniciar sesión como colaborador. El administrador publica la carta y, por separado, elige **Habilitar pedidos online** en Catálogo. La migración conserva todos los locales con recepción de pedidos desactivada; publicar la carta no implica recibir compras.

El comprador elige productos disponibles, cantidad, extras y preferencias para la cocina. El carrito pertenece a ese local y permanece en memoria; cambiar de restaurante no traslada sus productos. Antes de confirmar, el servidor cotiza los productos y extras de esa sucursal, verifica cantidades y stock, y calcula el envío desde un método activo de ese mismo local. El retiro siempre está disponible; el envío aparece si el restaurante configuró métodos activos. Los horarios se muestran como información: todavía no existe un calendario automático de apertura, zonas geográficas ni estimación de entrega.

La confirmación solicita nombre y teléfono, correo opcional y dirección para el envío. No guarda esos datos en almacenamiento del navegador. Contacto, dirección y medio elegido llegan a **Caja y pedidos**; Cocina recibe productos, extras y preferencias, sin datos de contacto. El pedido nace pendiente, sin descontar stock ni registrar una venta. Aceptarlo realiza la operación existente de venta e inventario en una transacción. El local puede rechazarlo si no puede atenderlo.

## Pago y seguimiento

Se puede elegir pagar al retirar o recibir. El restaurante registra el cobro efectivamente recibido con los controles habituales de caja. Mercado Pago se ofrece únicamente con integración habilitada para producción, cifrado configurado y cuenta conectada de esa marca/local. El comprador puede iniciar el pago después de que el local acepte el pedido. Los checkouts, cargos, comisiones, reintegros y conciliación usan el diario financiero existente; regresar desde Mercado Pago no acredita un cobro. Un intento ambiguo se mantiene en verificación y no se reemplaza con otra creación automática.

El recibo se recupera al recargar o volver de Mercado Pago en la misma pestaña. Se consulta cada 30 segundos mientras la página es visible, o con **Actualizar estado**. Muestra importe debido y efectivamente recibido. No se envía un correo/SMS de compra: se conserva en la pestaña una capacidad de 24 horas y se ofrece el enlace separado de seguimiento de 7 días. El enlace de seguimiento permite consultar el estado, pero no iniciar pagos. Cerrando la pestaña se pierde la capacidad de compra; conviene guardar ese enlace. Los pedidos pendientes no reservan stock.

## Contrato y seguridad

| Endpoint público | Función |
| --- | --- |
| `POST /api/public/menu/{id}/quote` | Cotización actual de un carrito del local |
| `POST /api/public/menu/{id}/orders` | Crear pedido con cotización vigente y confirmación idempotente |
| `GET /api/public/buyer-order` | Recibo del pedido de esta capacidad |
| `POST /api/public/buyer-order/mercadopago` | Pago del propio pedido aceptado, sin importe ni ID elegibles |

La cotización recibe `items` (`product_id`, `quantity`, `extras` con `extra_id` y `quantity`, `note`), `fulfillment` (`pickup`/`delivery`) y `shipping_method_id` solo para delivery. La creación agrega `quote_digest`, `name`, `phone`, `email`, `address` y `payment_method` (`at_local`/`mercadopago`). El digest obliga a revisar de nuevo si cambió la cotización; no autoriza precios elegidos por el cliente. Se rechazan campos desconocidos, recursos ajenos y cantidades fuera de límite. No se aceptan descuentos, roles, entorno, estado ni importes desde este flujo.

El navegador genera un secreto aleatorio de 256 bits y una clave UUID de confirmación, conserva solo la capacidad y claves en `sessionStorage` por sucursal y envía el secreto como `Authorization: Bearer`. La base guarda únicamente su SHA-256 en `legui_private.buyer_orders`, con acceso directo revocado y RLS. No es una sesión JWT. La capacidad no abre endpoints privados y el servidor fija el pedido y local del checkout desde la capacidad, ignorando IDs de ámbito enviados por el cliente. El recibo es una proyección explícita sin contacto, dirección, credenciales ni datos de otras compras. Se niega cuando la marca está suspendida/cerrada o la sucursal inactiva; pausar nuevos pedidos no impide seguir los ya recibidos.

Los reintentos de creación deben mantener secreto, `Idempotency-Key` y cuerpo idénticos. Un mismo identificador con otro secreto no revela el pedido; modificar el cuerpo genera conflicto. Si se perdió la respuesta, el cliente conserva el mismo intento y puede consultar el recibo antes de reintentar. Nunca interpreta un error de red como una confirmación de compra o de pago. Los pedidos/capacidades son históricos; no se exportan secretos de acceso en respaldos comerciales.

Hay límites persistentes por local y dirección TCP: 240 cotizaciones, 30 confirmaciones y 60 inicios de pago por ventana de 15 minutos. No se confía en `X-Forwarded-For` arbitrario. Detrás de un proxy sin configuración de IP confiable, los clientes comparten el límite del proxy; antes de una publicación masiva se necesita configurar esa frontera y controles de tráfico del despliegue. No se incorporó CAPTCHA ni limitación por identidad telefónica.

## Verificación

- `npm --prefix frontend run build`, `go test ./...`, `go vet ./...`.
- `python tools/check-source.py` y `python tools/check-skills.py`.
- `go test ./internal/operations -run TestBuyerCheckoutIntegration -count=1 -v -args -security-integration`: aislamiento, precios/extras/envío, reintentos, datos privados, aceptación y stock. Solo fixtures revertidas en el proyecto fijado.
- `go test ./internal/mercadopago -run TestDatabaseIntegration -count=1 -v -args -mp-db-integration`: proveedor HTTP simulado; comprador, aceptación, importe del servidor, capacidad de seguimiento sin acceso a pago, retorno sin secreto y liquidación verificada. No hace cobros reales.
- `node tools/test-buyer-browser.mjs`: escritorio/móvil, claro/oscuro, personalización, entrega, cotización obsoleta, respuesta perdida y reintento idéntico, recibo al recargar, pago y separación entre locales. Usa el mismo harness Puppeteer que las otras suites.

## Recuperar fotos anteriores y publicar una carta

Las referencias antiguas `/uploads/products/*.jpg` pueden recuperarse desde los archivos originales con `cmd/repair-catalog-images`. El comando exige proyecto y sucursal, limita la lectura a imágenes con nombre UUID dentro del directorio indicado, reutiliza la validación/limpieza y el diario de subidas actuales, verifica la imagen pública y reemplaza la referencia con control de concurrencia. Conserva los archivos originales y registra el vínculo anterior/nuevo en actividad. Solo procesa productos de la marca de esa sucursal; no copia productos ni cambia sus asignaciones, precios o activación.

```powershell
go run ./cmd/repair-catalog-images --expect-project ljdebnvzdrlqdxnligvh --tenant <UUID>
```

Sin `--apply` solo informa el alcance. Agregar `--apply` recupera las fotos. `--publish` publica esa sucursal después de verificar las fotos de sus productos activos asignados; `--enable-orders` habilita además los pedidos y requiere `--publish`. Se comprueban los proyectos de DB y Storage antes de cualquier escritura. El comando necesita las credenciales de Storage existentes en `.env` y nunca las imprime. Un fallo deja completadas las recuperaciones ya verificadas; el reintento solo toma las referencias antiguas pendientes.

Se recuperaron las cinco imágenes locales antiguas de Poggs y se publicó su sucursal City Bell (`ddf1e412-d10a-46f1-8b00-fc15e78059f0`), con los dos productos que ya tenía asignados: Ankara Messi y La Ganadora. La marca Leguiburger es otro restaurante en la DB y no recibió esos productos. El acceso público de prueba es `/catalogo/ddf1e412-d10a-46f1-8b00-fc15e78059f0`.
