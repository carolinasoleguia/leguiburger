# Recursos y limpieza

Leer el [contexto del repo](../../../../AGENTS.md). Identificar quién crea, usa y
cierra cada respuesta HTTP, archivo temporal, imagen o sesión de navegador.

- Usar context managers/finally según el módulo. Cerrar recursos también
  cuando fallen validación, parseo, timeout o cancelación.
- Limitar lectura antes de materializar el body; comprobar tamaño, tipo y
  expansión de formatos comprimidos, imágenes o planillas.
- Para streaming, mantener autorización, límites y respuesta parcial como
  parte del contrato; no exponer rutas de objetos recibidas sin verificar.
- Liberar gates/semaphores y renovar/conservar leases mientras siga activo
  el trabajo. Cancelar una coroutine no garantiza cierre de un hilo bloqueado.
- En ARCA portal, mantener el carácter opcional del navegador y la recuperación
  ante CAPTCHA/2FA; no repetir automáticamente un alta de resultado incierto.
- Limpiar únicamente temporales creados por el flujo. Borrado de objetos
  persistidos sigue autorización y jobs de cleanup del dominio.

Probar recurso cerrado después de éxito y excepción, respuesta sobredimensionada
y cleanup repetido en fixtures. No borrar archivos reales para validar limpieza.
