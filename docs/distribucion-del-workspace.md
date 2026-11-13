# Distribución del espacio de trabajo

El escritorio prioriza las acciones, cifras y primeras filas de cada pantalla en una notebook de 1366 × 768. La densidad se aplica únicamente dentro de `.app-shell`: la landing, la carta pública y el checkout conservan sus proporciones.

- Encabezado reducido, selector de local junto al título y sin el subtítulo decorativo repetido. El mensaje de falta de locales sigue visible.
- Resumen: seis métricas en una fila y tres paneles contiguos para gráfico, productos/suscripciones y actividad/pedidos. El detalle diario puede expandirse.
- Caja y pedidos: pendientes primero, pedido de mostrador e historial en columnas. Asignar cliente es una sección opcional desplegable que conserva la selección y el borrador.
- Turnos de caja: cifras arriba, operación e historial en columnas. Los controles «Ingreso o retiro» y «Arqueo y cierre» conservan ambos formularios; «Turnos» y «Movimientos» cambian la consulta. Elegir un arqueo abre sus movimientos.
- Clientes, gastos, cupones y opciones por producto: formularios y consultas relacionadas en columnas. Atención del local reúne recepción, capacidad y horarios.
- Cocina: carga compacta y tablero visible debajo. Cobros agrupa sus avisos y reduce el espacio anterior a los movimientos.

Los listados conservan todos sus registros, paginación y desplazamiento interno cuando hace falta. En móvil se apilan las columnas y se conserva el desplazamiento natural y los controles táctiles de 44 px; no se recorta el contenido para forzar una altura de pantalla. Las tablas estrechas mantienen su desplazamiento horizontal localizado.

Skeletons y contenido comparten los mismos tamaños reservados. El panel de caja reserva el espacio necesario para alternar entre apertura y turno activo sin mover el historial durante la carga.

## Validación

`tools/test-loading-layout-browser.mjs` verifica geometría, columnas, CLS y aparición sutil en estados vacíos/cargados. También comprueba que las tablas principales, el tablero de cocina y los paneles del resumen permanezcan dentro de presupuestos de posición en escritorio. Admite `LAYOUT_WIDTH`, `LAYOUT_HEIGHT`, `LAYOUT_THEME`, `LAYOUT_POPULATED` y `LAYOUT_SCREENSHOTS` para revisar notebooks y guardar capturas en `.cache/loading-layout/`.

El CI incluye una pasada a 1366 × 768, además de los tamaños habituales de escritorio y móvil. Las suites de workspace, caja, cobros, opciones y atención del local verifican las acciones existentes, incluido SSE, después del cambio de distribución.
