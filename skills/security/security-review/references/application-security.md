# Seguridad del desarrollo de Al Toque

Usar para revisar o implementar un cambio potencialmente vulnerable, empezando
por el diff/flujo. Leer el [contexto del repo](../../../../AGENTS.md) y seguir
productores y consumidores fuera del diff si la garantía depende de ellos.
Las convenciones de código siguen en `python-guidelines`; no reorganizar módulos
o reemplazar frameworks por realizar una revisión de seguridad.

## Entrada, control y efecto

Ubicar quién controla la entrada, qué valida el servidor, qué identidad/objeto
usa el ejecutor y dónde termina el dato. Una búsqueda señala un candidato;
hace falta demostrar el camino alcanzable y el control faltante. Considerar
fallos de dependencias, datos válidos sin autoridad y operaciones concurrentes.

| Frontera modificada | Qué comprobar en este repo |
| --- | --- |
| Browser/Next a API | DTO/Zod/Pydantic efectivos, tamaños, identidad verificada y autorización por objeto; no confiar en rol/actor enviados. |
| API/bot a Supabase | JWT frente a service role, filtros/IDs tipados, proyección mínima, grants/RLS/RPC; RLS no protege automáticamente una operación privilegiada. |
| Webhook/job | Firma y bytes originales, receptor/topic, dedup, claim/actor del journal y ACK después de persistencia requerida. |
| HTTP saliente | URL/destino permitido, path/encoding, redirects, límites/timeout y credenciales; URLs externas no conceden acceso a recursos internos. |
| Archivo/parser | Tamaño antes de buffers, formato real, dimensiones/expansión/miembros, temporal/path y contenido servido; MIME declarado no prueba formato. |
| HTML/PDF/CSV | Encoding/escape según destino y enlaces controlados; preservar defensas del renderer, no aplicar una sanitización única para todo. |
| Mutación comercial | Objeto/estado/versión, importes recalculados y moneda, CAS/transacción, créditos/stock y resultado incierto; autorización no prueba corrección de negocio. |
| Logs/modelo/Storage | Campos entregados, contexto/error externo, links-capacidad, baja lógica, acceso y retención; reducir exposición antes de exportar. |

Para permisos leer [autorización](authorization.md); para herramientas de modelo,
[IA](ai-agents.md); para claves/cifrado, [credenciales](credentials.md). No cargar
las tres si el cambio no cruza sus decisiones.

## Casos que merecen seguimiento específico

- Revisar inyección en SQL/filtros, comandos, templates, XML y paths según el
  sink real. Mantener parámetros/validadores y presupuestos de consulta; un
  string interpolado no demuestra por sí solo un exploit alcanzable.
- Comparar campos recibidos con los que se permite mutar: ownership, rol, saldo,
  estado fiscal o plan no son editables por tener forma válida. Evitar mass
  assignment y contratos que entreguen filas completas con credenciales.
- Verificar CORS/origen/CSRF/cookies por canal. El uso directo de Bearer en este
  frontend no obliga a reescribir Auth a cookies; CORS no autoriza un objeto.
- Revalidar estado/ownership en el punto transaccional. Seguir replay, pérdida
  de respuesta, claim vencido y dos actores; consultar ejecución solo si ese
  ámbito cambia, conservando journals y conciliación.
- Revisar dependencias/locks nuevos y permisos de una integración antes de
  confiar en SDKs o ejemplos generados. Un advisory necesita versión afectada
  y contexto; consultar fuentes primarias actuales si se lo reporta.

## Build/despliegue, cuando el cambio los afecta

Leer Dockerfiles, Compose, `.dockerignore`, workflow y configuración documentada.
Comprobar secretos fuera del build web, usuario no root, rutas técnicas,
privilegios de CI y artefacto asociado al commit. Compose no describe toda la
topología del hosting real; separar configuración y permiso efectivo/staging.

Evaluar el control requerido y la plataforma disponible, sin exigir AWS,
Kubernetes, Vault, mTLS, SIEM o WAF por una guía importada. Mantener readiness,
fingerprint y release gates; un gate nuevo requiere alcance pedido, herramienta
verificada y baseline revisado.

Elegir regresiones que detecten el fallo: otro taller/rol, límite excedido, dato
prohibido ausente, destino rechazado, replay/concurrencia o efecto incierto.
Usar unittest/mocks y fixtures actuales; SQL aplicado y proveedor real necesitan
el [cierre de migración](../../../../docs/operacion/despliegue.md) y alcance autorizado.
No ejecutar escaneo de puertos, fuzzing remoto, cobros, emisiones o envíos para
justificar una revisión estática. Reportar verificaciones y pendientes.
Referencia: [OWASP Secure Code Review](https://cheatsheetseries.owasp.org/cheatsheets/Secure_Code_Review_Cheat_Sheet.html).

## Revisión operativa por diff

1. Identificar base y HEAD/worktree, archivos modificados, objetivo y consumidores.
   Para una revisión de estado actual seguir también cambios no comiteados;
   distinguirlos del diff de rama. Guardar esa base con el informe.
2. Marcar puntos de entrada nuevos/cambiados: ruta, botón/comando, webhook,
   payload de job, URL/archivo o contenido leído por el modelo. Anotar quién
   puede controlarlos y qué autenticación atraviesan.
3. Leer el caso de uso, DTO, adapter de datos/proveedor y SQL/consumidor necesarios.
   Seguir cada dato sensible hasta lectura, escritura, envío o log. Si el diff
   elimina un guard, averiguar qué garantía aportaba y quién lo reemplaza.
4. Contrastar acceso legítimo y abuso: otro objeto/taller, estado incompatible,
   repetición, valor extremo o proveedor que devuelve contenido inesperado.
   Comprobar rechazo antes del efecto, no solo mensaje de error al final.
5. Separar hallazgos confirmados, riesgos condicionados y endurecimientos;
   proponer el cambio mínimo y las regresiones. Usar
   [remediación](remediation.md) para cierre y comparación posterior.

Para un relevamiento amplio, recorrer familias/entradas declaradas y registrar
cobertura; el mismo checklist no acredita haber revisado todas las rutas. En
implementación, identificar estas garantías antes de escribir el efecto y
mantenerlas en los tests del cambio, sin exigir escanear todo el repo.

## Checklist de desarrollo: elegir filas aplicables

| Control | Pregunta concreta | Evidencia de aceptación |
| --- | --- | --- |
| DTO/validación | ¿IDs, enums, dinero, listas y tamaños se validan antes del I/O? | Límite/entrada malformada rechazados sin llamadas al proveedor ni mutación. |
| Mass assignment | ¿El body puede cambiar actor, rol, taller, saldo o estado protegido? | Campos prohibidos rechazados/ignorados explícitamente; la operación deriva su autoridad. |
| Inyección | ¿Un dato alcanza SQL, filtro PostgREST, shell, template o XML como sintaxis? | Parámetros/encoding apropiados y tests de valores reservados conservan scope. |
| Acceso | ¿Se comprueba el recurso y la operación después de validar identidad? | A puede acceder a lo suyo; B/rol insuficiente no lee ni modifica el mismo ID. |
| Datos devueltos | ¿El DTO entrega columnas/tokens o links que el consumidor no necesita? | Campos privados ausentes en respuesta, contexto IA, log y artefacto público. |
| Archivo | ¿Se limita body antes del buffer y expansión antes del parser completo? | Exceso, formato ambiguo, dimensiones falsas y miembros inválidos fallan acotados. |
| Path/URL | ¿Se acepta destino libre, credencial en URL, redirect o nombre que sale del directorio? | Destino/path rechazado antes de leer/red; no hay fallback que omita el control. |
| Webhook | ¿Firma/receptor se validan y la intención se persiste antes del ACK requerido? | Firma/receptor inválidos no publican; replay no crea otro efecto. |
| Carrera | ¿La validación sigue vigente al commit/efecto? | CAS/ownership/transacción impiden ejecutar con versión o claim ajenos. |
| Dinero/créditos | ¿Monto/moneda/precio vienen de autoridad comercial y se recalculan? | Cliente no impone saldo/plan ni duplica cobro/crédito con dos requests. |
| Resultado incierto | ¿Un timeout invita a crear otra emisión/cobro? | Se consulta/reconcilia la misma identidad y se conserva journal. |
| Error/log | ¿Una excepción externa revela dato o transforma fallo en éxito? | Mensaje saneado, estado recuperable, metadata útil y sin secretos. |

Un status 400/403 sin comprobar ausencia del efecto puede ocultar una mutación
ya ejecutada. En tests async, assert sobre la dependencia donde la importa el
consumidor, y comprobar también la ruta legítima; no basta provocar una excepción.

## Ejemplos adaptados: qué buscar y dónde comprobarlo

### Acceso a presupuesto

Ejemplo conceptual de fallo: el usuario autenticado aporta `ticket_id` y un
handler consulta ese ID con service role sin pertenencia. Validar UUID elimina
formas inválidas, pero no verifica ownership. La corrección depende del canal:
para usuario usar contexto/taller verificado y recurso autorizado; para enlace
público validar su capacidad y ciclo de vida. No copiar ownership de usuario
a un flujo público diseñado para compartir.

El [resend de tickets](../../../../apps/api/app/api/routes/tickets.py) muestra
`get_user_client`, `workshop_context` y filtro conjunto de ID/profesionales.
Seguir RLS además del filtro; no trasladar automáticamente el handler a service
role. En la prueba negativa usar un ticket de otro taller y comprobar que
`send_text`/`send_document` no se invocan. La fixture SQL real debe verificar
grants/RLS de esos roles, fuera de los mocks de la ruta.

### Deserialización y transformación

JSON parseado sigue siendo entrada sin autoridad. Después de validar schema,
revisar campos mutables y cálculo. El [borrador de presupuesto](../../../../apps/api/app/bot/budget_draft.py)
usa Pydantic y recalcula importes; esa es una garantía de negocio distinta de
parsear JSON. Mantener límites de ítems, Decimal y contratos de serialización.
No introducir `eval`, pickle de entrada externa o un parser inseguro para
reemplazar la validación. Que una librería acepte JSON no demuestra ejecución
de código ni la ausencia de problemas de acceso/recursos.

### HTML, PDF y enlace de pago

Seguir el dato hasta [budget_pdf.py](../../../../apps/api/app/infrastructure/budget_pdf.py):
preservar escape de texto/atributos y verificación del destino de un link.
En el frontend revisar HTML insertado, componentes cliente y links externos
según el sink. El test compara la salida para caracteres reservados y URL
rechazada; no usa solo el mismo helper de escape como oracle.

### SSRF, transporte y uploads

Inspeccionar [whatsapp_media.py](../../../../apps/api/app/infrastructure/whatsapp_media.py)
y [bot_logo.py](../../../../apps/api/app/infrastructure/bot_logo.py) para allowlist,
paths, redirects y bytes. Una URL HTTPS puede seguir apuntando a un destino no
autorizado. Probar el destino final/redirect y error del transporte con mocks;
no consultar una metadata endpoint interna para demostrar el caso.
Para [arca_batch_upload.py](../../../../apps/api/app/core/arca_batch_upload.py)
revisar tamaño comprimido/expandido, hojas/miembros y límites de procesamiento.

## Matriz mínima de pruebas por cambio

| Cambio | Caso válido | Casos negativos/fallos a elegir |
| --- | --- | --- |
| Nuevo endpoint con datos privados | Usuario/membresía vigente y objeto permitido | JWT ausente/inválido, otro taller, rol insuficiente, miembro revocado, objeto borrado. |
| Mutación/acción pública | Capacidad vigente y estado compatible | Revocación, baja lógica, otro estado, doble click, versión vieja y efecto confirmado con respuesta perdida. |
| Import/upload | Archivo pequeño y formato permitido | MIME mentiroso, bytes/píxeles/filas excesivos, archivo truncado, timeout y temporal liberado. |
| Integración | Destino/actor/payload permitidos | URL/path/redirect ajenos, firma inválida, proveedor malformado, 429/5xx y timeout incierto. |
| Cambio de privilegio/config | Configuración requerida y permiso mínimo | Falta de secreto en ingreso real, public env prefix, permisos de CI/SQL demasiado amplios. |

Registrar fixtures/tests usados y verificaciones no realizadas. Enlazar resultados
al hallazgo, sin imponer un porcentaje de cobertura ni una suite genérica nueva.
