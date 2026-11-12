# Responsive y feedback

Revisar dashboard, páginas públicas y formularios según el flujo afectado,
incluidos nombres largos, importes, archivos y mensajes de validación.

- Ajustar layout al contenido con Grid/Flex y breakpoints existentes. Para
  contenedores reutilizados valorar container queries solo si resuelven el caso.
  Evitar anchos fijos que desbordan, zoom bloqueado y pérdida de acciones en móvil.
- En tablas/listados, elegir reflow, tarjetas o desplazamiento localizado sin
  separar datos de su acción ni ocultar campos necesarios para decidir.
- Comprobar sidebar/dialogs, viewport con teclado virtual y áreas táctiles.
  Conservar orden lógico, foco visible y orientación al cambiar tamaño.
- Mantener información y acciones ante zoom, texto largo, datos vacíos,
  permisos insuficientes y conexiones lentas, sin mostrar filas truncadas como
  si fueran el total del recurso.
- Distinguir idle, carga, pendiente persistido, éxito confirmado, error
  recuperable y resultado incierto. No borrar el borrador por un fallo de red.
- Deshabilitar solo acciones que realmente no se pueden ejecutar, explicar
  el estado y prevenir doble envío sin bloquear toda la navegación.
- El reintento visual debe seguir el contrato existente de idempotencia y
  recuperación. No crear otro cobro o emisión por cambiar el estado del botón.
- Preferir feedback simple; para movimiento respetar `prefers-reduced-motion`,
  permitir interrupción y evitar layout shifts/animaciones que retrasan la tarea.

Elegir tamaños y estados representativos y probarlos en el navegador. Usar
fixtures locales del repo para validar recorridos con proveedores simulados;
no afirmar prueba en dispositivo físico si solo se usó un viewport emulado.
