# Skills y pulido de frontend

Se importó la biblioteca completa de `skills` de Al Toque y se agregó `unslop`
desde su entrada nativa. Son 135 skills de la biblioteca más `unslop`.
Los 788 archivos recibidos se conservan con hashes en el manifiesto de importación;
el adaptador de LeguiBurger es un documento separado. No se ejecutan instaladores
ni scripts importados al copiar o verificar la biblioteca.

## Aplicación

Se aplicaron `web-ui-design` y `interaction-design` a componentes compartidos,
feedback y carga. Finalmente se aplicó `unslop` de Damir Sokolovsky
([@heydamir](https://x.com/heydamir)), CC BY 4.0, en modo pulido.
Las guías de React/FastAPI y las direcciones visuales alternativas no se
aplicaron a este frontend Vue.

Dirección conservada:

- Producto: gestión de marcas, sucursales y operación de hamburgueserías.
- Carácter: cálido, directo y práctico.
- Mundo: cocina de barrio, carbón, mostaza y papel de pedidos.
- Protagonistas: ilustración de hamburguesa en la landing y actividad real en el dashboard.
- Detalle propio: “Tu negocio, a punto”.

El análisis visual encontró ruido puntual, no una necesidad de rediseño:
íconos genéricos repetidos, rótulos pequeños en mayúsculas con mucho espaciado,
menú excesivamente largo y brillos en controles de la landing.

Las pasadas de `unslop` se acotaron a esos hallazgos:

1. Espaciado: filas de navegación de 44 px con separación de 2 px y padding
   uniforme; métricas con 24 px. Se preservan áreas táctiles y geometría de carga.
2. Color: bordes neutros en el dashboard, selección sobria, sin sombras de color
   en botones ni fondos degradados en selección y plan destacado.
3. Íconos: cobros, gastos, actividad, seguridad, soporte e inventario tienen
   símbolos distintos del mismo conjunto SVG existente.
4. Texto: encabezados compartidos y navegación en estilo oración; se conserva
   el wording y orden de secciones elegidos para la landing.
5. Tipografía: rótulos y columnas legibles a 12 px, encabezados de peso 600,
   sin tracking en texto pequeño y cifras tabulares.
6. Layout: se mantiene la estructura de Al Toque adaptada y el ancho del sistema.
7. Detalles: foco visible, color consistente en selección y grilla tenue en gráficos.
8. Visuales: se mantiene la hamburguesa elegida por el usuario; no se generan
   imágenes ni se incorporan recursos externos.
9. Movimiento: hover discreto y curvas consistentes. Por pedido del usuario,
   las animaciones y demostraciones continúan activas, sin botón de pausa ni
   barra superior global. Se mantienen skeletons, spinners locales y fundido suave.

La preferencia anterior `legui-motion` se descarta al iniciar. La prueba de
movimiento comprueba también un navegador que conserva la antigua pausa.

## Evidencia

Build de producción y verificación de fuentes e integridad de la copia.
La prueba `tools/test-motion-browser.mjs` valida carga, errores, borradores,
modales y animaciones activas. `tools/test-loading-layout-browser.mjs` cubre
84 combinaciones de pantallas, roles, datos vacíos/poblados y viewports de
1440/390 px: sin deriva de anclas, sin overflow, CLS 0 y fundido observado.
Las pruebas usan API simulada; no representan una auditoría completa de
accesibilidad ni una validación en dispositivos físicos.
