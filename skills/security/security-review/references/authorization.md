# Autenticación y autorización

Leer el [contexto del repo](../../../../AGENTS.md). Ubicar el flujo en
`apps/web/src/lib`, `apps/api/app/api/dependencies.py` y los adaptadores
`apps/api/app/infrastructure/supabase_user.py` y `supabase_service.py`.

- La web conserva Supabase Auth/cookies y transporta JWT; la API verifica
  identidad y permisos antes de acceder a datos de negocio.
- Derivar actor y taller de identidad verificada y membresía persistida.
  Un ID recibido, un teléfono o `user_metadata` editable no otorgan permisos.
  Claims administrativos controlados por servidor tampoco sustituyen ownership,
  membresía vigente o permiso del recurso. No imponer un IdP como fuente única
  de los roles comerciales que este repo mantiene en SQL.
- Revisar acceso con JWT del usuario frente a uso de service role: este último
  exige autorización explícita en el caso de uso y puede omitir RLS.
- Mantener firmas de Meta/MP y sus cuerpos originales; el secreto interno
  autentica comunicación privada y no sustituye una firma de proveedor.
- En OAuth, seguir estado, callback y propietario del token cifrado. No
  registrar códigos, tokens ni URLs que concedan acceso por sí mismas.
- Revisar políticas/grants en `apps/web/supabase/migrations`, incluidos
  Storage y RPC con privilegios, sin confundir SQL local con políticas aplicadas.
  Comprobar rol, operación, USING/WITH CHECK, permisos y objeto efectivos.
  RLS habilitado no acredita aislamiento; `USING (true)` puede ser una lectura
  pública intencional o un permiso excesivo según recurso y rol. En SECURITY
  DEFINER revisar owner, EXECUTE, search_path y objetos calificados.
- Distinguir autenticación por JWT, cookie y secreto interno al revisar CORS,
  Origin/CSRF y cookies. Seguir proxies/orígenes efectivos; un header recibido
  o un token válido no acredita por sí solo autorización sobre el objeto.
- Revisar disponibilidad/revocación de enlaces públicos y pertenencia de
  objetos Storage, incluso después de una baja o borrado lógico.
  Un UUID compartido puede ser una capacidad pública por diseño; evaluar scope,
  vigencia y acciones, sin etiquetar toda ruta sin login como IDOR.

Probar usuario ausente, token inválido, otro taller, rol insuficiente,
firma inválida y ruta legítima. Usar las suites actuales y consultar
`docs/operacion/despliegue.md` para validación de RLS en staging.

## Procedimiento para probar acceso por objeto

1. Identificar actor, ámbito, rol, objeto y acción; distinguir la identidad técnica
   del worker/web de la identidad de negocio. Un secreto interno válido solo
   acredita el llamador según su alcance, no cada ID del body.
2. Localizar la comprobación y el acceso real: middleware/dependency, caso de uso,
   RPC, tabla/Storage. Revisar qué credencial ejecuta cada paso; documentar dónde
   una service role omite controles que sí tiene el JWT del usuario.
3. Anotar fuente de rol/membresía y su vigencia. Usuario/metadata editable no
   concede permiso; una membresía revocada puede exigir revalidación aunque el
   token aún sea válido. No asumir que todo permiso comercial viene del IdP.
4. Leer grants/policies y reemplazos posteriores del historial. Con RLS revisar
   SELECT/INSERT/UPDATE/DELETE, USING/WITH CHECK, rol y pertenencia; evaluar
   políticas combinadas y permisos de EXECUTE de RPC, no solo una sentencia.
5. Probar matriz por actor/acción y repetir para escritura REST directa si esa
   superficie existe. Los tests del handler no acreditan que la ruta directa
   tenga el mismo control. Validación SQL efectiva queda en staging autorizado.

| Actor/condición | Leer objeto privado | Mutar objeto | Efecto externo |
| --- | --- | --- | --- |
| Propietario/miembro habilitado con rol suficiente | Permitido según regla | Permitido con estado/versión correctos | Solo operación comercial autorizada. |
| Usuario de otro taller | Denegado | Denegado | Ningún envío/cobro/emisión por ese objeto. |
| Miembro del mismo taller sin permiso de acción | Según permiso de lectura | Denegado | Ningún efecto que requiera ese permiso. |
| Miembro revocado | Según política efectiva, no token histórico por sí solo | Denegado cuando perdió membresía | Ningún efecto nuevo por autoridad revocada. |
| Llamador técnico sin claim/actor permitido | Solo lectura técnica permitida | Denegado | Denegado, aunque el secreto sea válido. |
| Enlace público vigente | Proyección pública explícita | Solo acciones del scope y estado | No hereda capacidades administrativas. |

La respuesta puede ser 403 o 404 según política de no revelar existencia;
comprobar semántica y ausencia de efecto, sin exigir un status universal.

## Cookies, Bearer, OAuth y headers

- Seguir de dónde sale JWT/cookie y quién lo verifica; no confiar en decodificación
  local sin la validación requerida. Preservar Supabase Auth actual, sin inventar
  contraseñas/hashing propios a partir de un ejemplo importado.
- Clasificar mutación por canal: Bearer explícito, Server Action o cookie en Route
  Handler. Verificar origen/token CSRF según mecanismo; CORS no sustituye CSRF,
  autenticación ni permiso. Cookies sensibles requieren flags adecuados al flujo.
- Para OAuth, identificar state/nonce, sesión o capacidad de bot, TTL, replay,
  intercambio incierto y scopes. Comprobar continuidad/actor antes de persistir
  tokens; preservar mecanismos de claim/conciliación existentes.
- No usar headers de IP/actor como identidad por su nombre; seguir el proxy que
  los produce y el acceso directo al servicio. Rate limiting complementa acceso.

## Cómo leer un ejemplo RLS sin copiarlo ciegamente

Una policy `auth.uid() = user_id` solo sirve si ese campo y el modelo de ownership
son correctos para la tabla. Al Toque tiene ámbito taller, membresía, canales
públicos y RPC privilegiadas; copiar una policy individual a todo recurso puede
romper acceso legítimo o dejar una escritura sin control.

Comprobar con dos actores de talleres distintos y roles efectivos: lectura y
escritura permitidas, denegadas y rollback; revisar columnas sensibles y Storage.
Una policy pública legítima se justifica por dato/proyección/acción, no se
declara vulnerable solo porque usa `USING (true)`. Para `SECURITY DEFINER`, registrar
owner, search_path fijo/objetos calificados, roles que ejecutan y parámetros de
autoridad; esos controles no se deducen del frontend ni del prompt del bot.
