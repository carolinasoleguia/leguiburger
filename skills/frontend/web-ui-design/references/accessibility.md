# Accesibilidad web

Revisar la interacción concreta y sus estados. Preferir HTML nativo con
semántica correcta antes de reconstruir controles con ARIA.

- Formularios: label asociado, ayuda/error vinculados, required/invalid
  coherentes y errores que permitan recuperar sin perder datos. No usar
  placeholder como única etiqueta ni color como único indicador.
- Teclado: acceso y orden lógico a acciones, foco visible y activación nativa
  de botones/enlaces. Los controles custom necesitan el comportamiento completo,
  no solo `role` o `tabIndex`.
- Dialogs: nombre accesible, foco inicial razonable, gestión de foco/modalidad,
  cierre previsto por el flujo y devolución al disparador. Conservar salida
  accesible ante error y no atrapar teclado fuera de una modalidad real.
- Feedback: estados/errores anunciados sin ruido ni saltos innecesarios de foco.
  Mantener información de carga/pending para lectores de pantalla.
- Imágenes/iconos: describir contenido útil, ocultar decoración y dar nombre
  a botones de solo icono. Mapas y gráficas requieren alternativa comprensible
  para los datos/acciones importantes del usuario.
- Comprobar zoom/reflow, contraste y modo oscuro; respetar reducción de movimiento.
  No bloquear zoom ni ocultar acciones esenciales en viewports pequeños.

Verificar teclado y estados primero; complementar con árbol de accesibilidad,
lector de pantalla o herramienta instalada si aporta evidencia. Informar
qué se comprobó y qué queda pendiente. No afirmar cumplimiento de una norma,
dispositivos o tecnologías asistivas que no se hayan evaluado.
