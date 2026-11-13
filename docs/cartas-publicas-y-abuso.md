# Cartas públicas y protección de pedidos

La ruta Vue `/:brand/:local/carta` resuelve el local mediante `GET /api/public/restaurants/{brand}/{local}`. Solo revela cartas publicadas de locales activos y marcas habilitadas. Después utiliza los endpoints de compra por UUID existentes: cookies, idempotencia, pagos y SSE mantienen su identidad estable. `/catalogo/:tenantId` sigue vigente.

La migración `202611130001_public_menu_slugs` asigna slugs a marcas y locales actuales y futuros. Normaliza acentos y espacios, agrega sufijos ante colisiones y exige unicidad global para marcas y por marca para locales. Los slugs se conservan al renombrar, desactivar o cerrar un negocio. Los índices y bloqueos transaccionales coordinan altas simultáneas. La publicación, QR, directorio y enlace desde configuración usan el nuevo path.

Render sirve el frontend mediante el fallback SPA que ya existe en Go (`spaHandler`). No hay reglas ni despliegues adicionales por sucursal. No se usa Vercel. El runner de migraciones compara datos originales dentro de una instantánea consistente para que los heartbeats concurrentes de los workers no alteren esa verificación.

## Abuso

No se detecta ni bloquea el uso de DevTools. Las cartas y el seguimiento continúan accesibles durante un bloqueo de creación de pedidos.

- Antes de consultar la DB: ventanas por IP y acción, compartidas entre sucursales. Por minuto: 20 intentos de pedido, 120 cotizaciones y 60 inicializaciones de sesión. Exceder la ventana pausa esa acción durante 5 minutos; reintentar no extiende ese plazo. Caché acotada a 10.000 entradas por proceso.
- Contadores persistentes por IP/sucursal: 120 intentos de pedido y 240 cotizaciones cada 15 minutos. Sesión/sucursal: 10 intentos de pedido cada 15 minutos. Son contadores en PostgreSQL compartidos entre réplicas; no se agregaron tablas ni se guarda la IP en claro.
- Respuesta HTTP 429, `Retry-After` y mensaje de pausa temporal en el checkout. Se conserva la confirmación pendiente para reintentar con la misma clave.
- En Render (`RENDER=true`) se utiliza `CF-Connecting-IP`, sobrescrito por su borde Cloudflare; fuera de Render se ignoran los headers de IP y se usa el peer TCP. No copiar `RENDER=true` a un servidor con ingreso directo. Verificar este header al cambiar de proxy/CDN. IPv4 mapeada a IPv6 se normaliza.

Estos controles reducen ráfagas y reintentos abusivos; no demuestran que una compra sea humana ni detienen por sí solos un ataque distribuido con muchas IP. Las redes Wi-Fi comparten el límite de IP, por eso el límite más restrictivo es por sesión. El filtro rápido en memoria es por réplica; PostgreSQL conserva el límite común. La protección DDoS de Render es complementaria.

Referencias: [Render](https://render.com/articles/how-render-handles-ddos-attacks), [headers de Cloudflare](https://developers.cloudflare.com/fundamentals/reference/http-request-headers/).
