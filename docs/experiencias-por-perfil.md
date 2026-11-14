# Experiencias separadas por perfil

Las rutas conservan Vue y Vite, pero cada experiencia tiene sus propias vistas y consultas. Una página de restaurante no contiene métricas de plataforma ocultas por un `v-if` de rol.

| Experiencia | Vistas principales | Datos y acciones |
| --- | --- | --- |
| Dueño de la app | `views/owner/OwnerHomeView.vue` | `/analytics/platform`: membresías, MRR, altas y actividad de restaurantes |
| Dueño o administrador del restaurante | `views/admins/AdminHomeView.vue` | `/analytics/restaurant`: ventas, pedidos, cobros, productos, clientes y resultados del local |
| Comprador con cuenta | `views/buyers/ShopperDashboardView.vue`, `ShopperCatalogView.vue`, `ShopperDirectoryView.vue` | Historial, favoritos, cupones, preferencias, perfil y compras asociadas a la cuenta |
| Comprador invitado | `views/buyers/GuestCatalogView.vue`, `GuestDirectoryView.vue` | Carta pública y pedido identificado por la sesión de compra, sin registro |

`ShopperAccessView.vue` contiene ingreso, registro y recuperación del comprador. `ShopperAccountView.vue`, `CatalogLandingView.vue` y `RestaurantDirectoryView.vue` son entradas pequeñas que resuelven la sesión y eligen la vista correspondiente. No concentran formularios o políticas de los dos perfiles. Las URLs `/mi-cuenta`, `/catalogo/:tenantId` y `/restaurantes` siguen funcionando.

La navegación de plataforma, restaurante y colaboradores está definida por separado en `navigation/platform.js`, `navigation/restaurant.js` y `navigation/collaborator.js`. `DashboardShell` se ocupa de presentar esos menús, abrir grupos y gestionar foco, sesión y tema.

## Componentes compartidos

- `OverviewAreaChart` y `OverviewDateRange`: presentación del gráfico y selección de fechas, sin conocer roles. El área, degradado, ejes y períodos de 7 días, 15 días y un mes se adaptaron del dashboard de Al Toque. Cada vista decide qué consultar.
- `RestaurantMenu` y `RestaurantDirectory`: búsqueda y presentación del catálogo público. Las acciones de cuenta se aportan mediante slots desde la vista correspondiente.
- `OrderCheckout`: motor común del carrito, cotización, creación, cookies, pago y seguimiento SSE. No importa el estado del comprador ni comprueba roles. `GuestCheckout` aporta el acceso invitado; `ShopperCheckout` aporta cupón, contacto, repetición y vínculo visual con la cuenta.
- Formularios, skeletons, selectores de producto, iconos, colores y tipografía siguen siendo reutilizables.

El calendario del encabezado filtra todas las métricas, gastos y tablas del resumen; los botones de 7 días, 15 días y un mes del gráfico se sincronizan con ese filtro general. Las fechas también se muestran en Resultado del período. Un único calendario permite elegir inicio y fin; el rango usa días de Argentina y admite hasta 366 días. No se presentan los campos Desde/Hasta separados. El tooltip se puede consultar con puntero o teclado, y el detalle diario sigue disponible como tabla.

Esta separación organiza el frontend; la autorización sigue en la API. El backend valida sesión, rol, marca, local y recurso. Un administrador de restaurante no puede consultar `/api/analytics/platform`, aunque cambie el cliente o fuerce la URL. El rol `admin` administra carta, locales, equipo, clientes, finanzas y reportes. No hereda permisos de caja ni puede consultar o modificar pedidos, imprimir tickets, generar links de cobro de pedidos, abrir/cerrar caja, usar cocina o suscribirse al SSE del personal. Los colaboradores conservan su operación según rol y local asignado; los dueños de la app conservan su acceso global. Estas restricciones se aplican en rutas Vue y handlers Go, sin cambios de esquema.

`views/admins/AdminPaymentsView.vue` contiene la experiencia financiera administrativa (movimientos, cuentas, revisiones y su propia membresía). `PaymentsEntryView` elige esa página o la experiencia de cobros operativos; el admin no carga pedidos ni el formulario para cobrarlos.

## Verificación

`tools/test-dashboard-overview-browser.mjs` verifica consultas independientes, ausencia de membresías en el resumen del restaurante, períodos, calendario, tooltip, teclado, reintentos y menú agrupado. Las suites existentes de compradores, cookies, cuenta, repetición, SSE y geometría verifican que la separación conserve el recorrido de compra y la estabilidad de carga.
