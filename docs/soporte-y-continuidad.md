# Soporte, continuidad y consistencia visual

La pantalla `/care` reúne soporte para todos los perfiles, respaldos para responsables de restaurantes y plataforma, y cierre explícito de restaurante o cuenta. Se conserva el acceso con usuario y contraseña.

## Permisos y reintentos

Todos los endpoints `/api/care/*` pasan por la validación de sesión vigente en la base. Las mutaciones requieren `Idempotency-Key`; el recibo incluye actor, método, recurso, restaurante y contenido. Una repetición con los mismos datos devuelve el resultado original. Una clave reutilizada con otro contenido se rechaza. Los identificadores de restaurante del cliente no reemplazan el alcance del administrador autenticado.

Las conversaciones son privadas para su autor y los administradores de la plataforma. Cada mensaje conserva autor, fecha y origen. Marcar una consulta como resuelta no elimina mensajes; responder la vuelve a abrir. Los textos se representan como texto, sin HTML. Se limita el tamaño, la frecuencia y la cantidad de registros devueltos. Las respuestas de la plataforma se notifican por la misma bandeja SMTP cifrada de autenticación; `SUPPORT_EMAIL` permite notificar nuevas consultas. El historial funciona aunque SMTP no esté configurado.

## Respaldos

Un responsable puede generar hasta tres respaldos por hora. La exportación opera sobre una transacción con snapshot consistente y una lista explícita de tablas operativas del restaurante. Incluye catálogo, locales, inventario, producción, clientes, pedidos, ventas, pagos de pedidos e historial. Las tablas lookup necesarias también se capturan. No incluye hashes de contraseñas, sesiones, factores, correos pendientes, credenciales OAuth ni respuestas crudas de proveedores. No es una copia completa de la base ni un reemplazo del backup/PITR de Supabase.

La prueba de reconstrucción inserta los registros en tablas temporales que conservan tipos, NOT NULL, CHECK y columnas generadas; recrea claves únicas y comprueba las relaciones entre tablas capturadas. Las referencias a identidades que se excluyen del archivo se informan por separado. No escribe filas en producción. La cuenta autenticada se vuelve a validar antes de una mutación y el acceso a la descarga se limita al restaurante correspondiente.

El snapshot se cifra con AES-GCM y datos asociados al nombre de objeto. `BACKUP_ENCRYPTION_KEY` es el secreto dedicado recomendado; si está vacío, se deriva una clave separada de `AUTH_ENCRYPTION_KEY` o `JWT_SECRET`. Debe mantenerse estable y respaldarse fuera de la base. Agregar o cambiar un secreto que altere esa selección impide abrir copias anteriores: antes de cambiarla hay que conservar el secreto previo como `BACKUP_ENCRYPTION_KEY` o implementar una rotación explícita. API y worker usan el mismo secreto. No se suben secretos a Git.

La migración crea `business-backups` como bucket privado. El job y su contenido cifrado se guardan antes de cualquier subida; el worker reclama con lease, usa una ruta inmutable, comprueba SHA-256 después de leer el archivo guardado y recién entonces retira el contenido temporal de la base. Si se pierde una respuesta, primero consulta la ruta existente y verifica su hash; nunca sobrescribe otra copia. Los fallos reintentan y pasan a revisión después de ocho intentos. Las descargas pasan por el backend autenticado, verifican hash y autenticación del cifrado y entregan JSON mediante attachment y `no-store`; no hay URL pública ni capacidad permanente de descarga. El límite actual es 20 MB por snapshot.

## Cierre del restaurante y baja personal

El bucket de respaldos también tiene una política restrictiva sobre `storage.objects`: los roles del navegador no pueden acceder aunque exista una política permisiva más amplia para otros buckets. La clave de servicio del backend conserva el acceso por su bypass de RLS.

Cerrar un restaurante exige al responsable principal o a la plataforma, contraseña actual, motivo y la confirmación `CERRAR RESTAURANTE`. Se archiva el negocio y se revocan accesos del equipo, invitaciones y seguimientos. Se conservan ventas, pagos e identidades históricas. El cierre es distinto de la suspensión administrativa: el toggle habitual no puede reabrir un restaurante cerrado.

Las membresías dejan de renovar localmente al terminar su período. Si existe un mandato de Mercado Pago, se prepara una cancelación durable y se procesa por el mismo journal, cliente, lease y conciliación de pagos. Una respuesta incierta queda en revisión; la pantalla distingue la solicitud de baja de la confirmación del proveedor. No se afirma que un débito remoto esté cancelado por haber archivado el negocio. La configuración `MP_ENABLED`, el entorno y `MP_LIVE_ENABLED` siguen controlando las llamadas reales. Se conserva el período abonado y no se generan devoluciones automáticas.

La baja personal exige contraseña y `CERRAR CUENTA`. Desactiva la identidad y revoca sus sesiones sin borrar historial. Un responsable de un restaurante activo debe transferirlo o cerrarlo primero; la plataforma conserva al menos otro administrador activo. Tras la baja, los recibos y la auditoría siguen en la base y la interfaz cierra la sesión. No es una eliminación física de datos contables.

## Imágenes y limpieza

Las imágenes nuevas aceptan JPEG/PNG reales, hasta 5 MB, 4096 píxeles por lado y 8 megapíxeles. El backend decodifica y vuelve a codificar el archivo, eliminando metadatos y contenido agregado. No confía en la extensión ni en el Content-Type del cliente. Genera la ruta `products/{brand_uuid}/{asset_uuid}.png|jpg`, exige la clave de servicio y evita redirects, sobrescrituras y mensajes crudos del proveedor. El formulario anuncia los formatos aceptados; se eliminan temporales multipart al terminar.

Un journal independiente de la transacción de producto permite encontrar subidas que quedaron sin publicar. Después de un día, el worker elimina solamente objetos generados por el servidor que no están referenciados por ningún producto. La publicación bloquea el registro del archivo y rechaza assets de otro restaurante o en limpieza, evitando competir con un borrado en curso. Un archivo referenciado no se elimina por estar inactivo o por cerrar el restaurante.

## Monitoreo y arquitectura

Los errores se agrupan en `legui_private.runtime_errors` por origen y categoría fija. La plataforma puede revisarlos en `/care`. El middleware captura errores 5xx y panics con ID de solicitud; el frontend captura errores de render, navegación y promesas. No registra el contenido de excepciones, stack arbitrario, formularios, cabeceras, cookies, consultas, URLs, fragmentos, IP, correos ni tokens. El ingreso del navegador requiere autenticación, un payload de categoría permitida y límites persistentes. Hay un límite adicional en el navegador.

`SENTRY_DSN` habilita opcionalmente un relay exclusivamente de servidor con estos mismos datos mínimos. Sin configurar el DSN, el registro interno sigue activo y no se envía nada a Sentry. El relay usa leases y reintentos persistentes. Su formato sigue la [especificación oficial de envelopes](https://develop.sentry.dev/sdk/foundations/envelopes/). No reemplaza un SDK de tracing ni envía sesiones/replay del navegador.

Las pruebas de arquitectura protegen la dirección de dependencias y que las rutas de continuidad validen sesiones. API y worker externo reutilizan el mismo registro de procesos de segundo plano; la funcionalidad de continuidad no depende de handlers CRUD antiguos ni del puntero global de la base.

## Pantallas y validación

Se comparten encabezados, estados de carga, tablas, tipografía, campos, errores, confirmaciones y tokens de color en modos claro y oscuro. El catálogo y producción se alinean con las pantallas nuevas. Las confirmaciones y modales administran foco, Escape y navegación con Tab. Los campos inválidos muestran errores asociados; los formularios mantienen sus valores si falla el guardado, advierten antes de descartar cambios y no guardan contraseñas ni borradores en localStorage.

Los patrones visuales de Al Toque se trasladan a Vue: entrada de datos de 480 ms y 6 px, skeletons con pulso de 1,3 s y shimmer de 1,6 s, cargas que reservan la forma de cada pantalla, spinners por panel, indicador compartido de solicitudes y feedback en el botón que inició la operación. Las rutas, tarjetas, filas, gráficos y modales tienen transiciones suaves; `prefers-reduced-motion` desactiva el movimiento. No se agregan esperas artificiales y los formularios permanecen montados durante las recargas.

El control de animaciones junto al cambio de tema permite activarlas o pausarlas explícitamente. La preferencia se conserva en ese navegador; mientras no se elige, se respeta la configuración del sistema. La prueba `tools/test-motion-browser.mjs` verifica cargas concurrentes, errores, conservación del formulario, entrada de paneles, modales y preferencias de movimiento con API simulada.

Durante la carga se conservan los contenedores reales: encabezados, columnas, formularios y controles mantienen su geometría. Las filas de skeleton usan las columnas de cada tabla; los listados largos tienen un viewport con scroll interno. Los dashboards reservan el tamaño de métricas y gráficos, y la actualización de los datos usa un fundido de 220 ms sin desplazamiento vertical. El encabezado se alinea arriba, sin centrado vertical dependiente del contenido. Se conservan también los espacios de avisos y fechas que llegan con la respuesta.

`tools/test-loading-layout-browser.mjs` compara anclas y tamaños antes/después de responder a la API simulada, observa CLS y comprueba el fundido en 80 escenarios: 20 pantallas, escritorio/móvil y listas vacías/con 25 registros. Las secciones de soporte que dependen de la cuenta pueden ampliar el documento debajo del primer viewport; se comprueba que sus paneles visibles y el scroll no salten. No se insertan registros ni se usa información real del negocio.

```powershell
go test ./...
go vet ./...
go test ./internal/operations -run 'TestBusinessHierarchyIntegration|TestSecurityIntegration' -count=1 -v -args -security-integration
go test ./internal/businesscare -run TestPrivateBackupStorageIntegration -count=1 -v -args -care-storage-integration
npm --prefix frontend run build
node tools/test-care-browser.mjs
node tools/test-loading-layout-browser.mjs
```

La prueba de DB hace rollback de fixtures. La de storage crea un único objeto de prueba sin datos reales en el bucket privado, comprueba lectura y rechazo público y lo elimina. Los navegadores usan una API simulada y revisan reintentos, formularios, roles, cierre pendiente de conciliación, accesibilidad y layouts de 1440/390 píxeles.
