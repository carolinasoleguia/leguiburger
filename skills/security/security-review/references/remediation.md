# Amenazas y mitigaciones

Leer el [contexto del repo](../../../../AGENTS.md). Partir de amenazas/hallazgos
sustentados y del alcance pedido; no generar un plan genérico de infraestructura.

- Asociar amenaza a control existente, brecha, componente responsable y cambio
  mínimo: ruta, caso de uso, adaptador, política/RPC, worker o configuración.
- Valorar impacto en acceso entre talleres, datos privados, pagos, facturas,
  disponibilidad y posibilidad de duplicar un efecto.
- Separar prevención, detección y recuperación. Una alerta no reemplaza
  permisos y una deduplicación local no confirma entrega única al proveedor.
- Incorporar pruebas negativas, recuperación y compatibilidad entre deploys.
  Para SQL o efectos reales, seguir staging y corte de `docs/operacion/despliegue.md`.
- Si hay una credencial realmente expuesta, identificar proveedor y alcance
  de la rotación necesaria sin mostrar el valor ni afirmar que ya se rotó.
- Registrar riesgos residuales y dependencias de acceso/evidencia; no cerrar
  un hallazgo solo porque una fixture pasó.

Entregar una tabla amenaza/control/brecha/cambio/validación/prioridad. Ejecutar
remediación solo dentro del trabajo autorizado; no abrir tickets, cambiar
permisos o enviar avisos externos por el hecho de elaborar el mapa.
