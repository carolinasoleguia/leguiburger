# Credenciales, cifrado y exposición

Usar al revisar secretos, transporte privilegiado, OAuth, cifrado, logs/bundles
o exposición concreta. Leer el [contexto del repo](../../../../AGENTS.md), ejemplos
de entorno y adaptadores afectados; no volcar `.env` o claves.

- Clasificar por proveedor, propósito, servicio y acceso. Una clave Supabase
  anon/publicable está diseñada para exposición; su seguridad depende de grants,
  RLS y autorización. Service role, token Meta/MP, refresh token, clave de cifrado
  o secreto interno no deben alcanzar browser, URLs, modelo o reportes públicos.
- Seguir `NEXT_PUBLIC_`, imports cliente y build/runtime, capas Docker, CI y
  Sentry. Reportar tipo y ubicación, sin valor ni muestras que lo reconstruyan.
  Un patrón requiere triage; una clave pública o fixture no es una filtración real.
- Variables privadas de runtime pueden ser el mecanismo de la plataforma;
  comprobar inyección, acceso y separación de propósitos. No exigir vault ni
  prohibir env vars universalmente. Evitar defaults inseguros y validar capacidad
  requerida sin bloquear por una integración opcional ajena al proceso.
- Revisar alcance/audiencia y propósito de firma del secreto compartido;
  separar transporte de state OAuth cuando proceda. Rotación/expiración dependen
  del proveedor y consumidores: inventariar orden de corte y compatibilidad.
- Preservar cifrado autenticado y nonces. En `infrastructure/secrets.py` seguir
  envelope, fallback plaintext, key ID/AAD y errores; no inventar criptografía
  ni afirmar formato/estado de todos los registros sin evidencia.
- Revisar Storage, export/backup y links-capacidad por ciclo de vida. Cifrar una
  columna no protege el dato que el servicio entrega al modelo o usuario ajeno.

Para detección automática usar `sast-configuration` solo si se pidió escaneo;
comprobar redacción, alcance y salida. No subir repo/credenciales/datos a un
servicio externo por defecto.

## Si hay evidencia de exposición

Documentar ubicación, capacidad, superficie y ventana conocida sin atribuir uso
indebido no demostrado. Quitar el valor del código no revoca la credencial.
Proponer revocación/rotación específica, consumidores, revisión de uso y prueba
del nuevo acceso; comprobar procedimiento primario actual. Conservar evidencia
saneada y distinguir cambio local de estado del proveedor.

No revocar, rotar, modificar permisos, purgar historial o enviar avisos sin
alcance autorizado. Si esas acciones ya están autorizadas, ejecutarlas con
comprobaciones y reportar el resultado, sin volver a pedir permiso por una regla
importada. Un commit de limpieza no demuestra cierre externo.
Referencia: [OWASP Secrets Management](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html).

## Inventario mínimo, sin valores

| Campo | Qué registrar |
| --- | --- |
| Proveedor/tipo | Supabase service role, Meta, MP/ML OAuth, OpenAI, Resend, secreto interno o cifrado. |
| Propósito/scope | Qué recursos/operaciones permite y cuáles no necesita. |
| Consumidores | API, worker, web server, CI/build y jobs/tokens históricos afectados. |
| Inyección/almacenamiento | Variable privada, secret de plataforma o envelope de datos, con owner del cambio. |
| Superficie | Browser, bundle, logs, artefacto, modelo, URL, backup o solo runtime privado. |
| Ciclo de vida | Emisión, expiración/refresh, posibilidad de coexistencia y revocación según proveedor. |
| Evidencia | Archivo/control observado y comprobación efectiva pendiente; nunca el valor. |

Partir de ejemplos de entorno y usos en código. El inventario no obliga a abrir
secrets reales para descubrir nombres, ni a sustituir el mecanismo del hosting.
Revisar build web y sus imports: mover OpenAI a una ruta Next con una clave
privada no es el diseño de este repo; el proveedor pertenece a la API Python.
La web debe transportar la operación autenticada y recibir el DTO necesario.

## Runbook de exposición, con alcance y orden de corte

1. Confirmar superficie y tipo de credencial. Distinguir patrón de test/clave
   pública de un secreto sensible, y exposición de uso indebido demostrado.
   Registrar commit/artefacto/ventana conocida y preservar evidencia saneada.
2. Identificar consumidores y mecanismo del proveedor. Si revocar ya está
   autorizado, usar el procedimiento vigente y acotar la ventana; si falta
   alcance de cuenta/acción, dejar explícito qué autorización/accesso falta.
   El análisis local no prueba que la clave esté activa o haya sido utilizada.
3. Elegir corte: revocación inmediata puede ser necesaria ante abuso confirmado;
   coexistencia controlada puede evitar interrupción si el riesgo lo permite.
   Documentar ese tradeoff, no aplicar “overlap siempre” ni “revocar todo” por regla.
4. Distribuir reemplazo por el mecanismo privado del servicio, actualizar solo
   consumidores necesarios y verificar acceso válido/fallo con la credencial
   retirada. Comprobar jobs activos, refresh tokens, state firmado y cifrado.
5. Quitar exposición de fuente/build/log y revisar derivados/copias accesibles.
   Reescribir historia no revoca secretos y afecta clones/ramas: requiere alcance
   explícito, coordinación y respaldo; no hacerlo como limpieza automática.
6. Revisar uso autorizado del proveedor durante la ventana, documentar límites
   de evidencia y añadir prevención proporcional. Avisos externos/incidentes
   siguen su alcance específico, no se disparan al cargar esta referencia.

Cierre distingue cuatro estados: exposición local retirada, consumidor actualizado,
credencial revocada por el proveedor y evidencia de funcionamiento/uso revisada.
Si solo se completó el primero, el informe no marca la filtración resuelta.

## Cifrado y rotación de claves no son el mismo caso

Para `TOKEN_ENCRYPTION_KEY`, seguir envelope, formatos legacy y todas las lecturas
antes de cambiar la clave: revocar acceso a un token no requiere perder capacidad
de descifrar datos existentes. Un diseño de rotación puede necesitar key ID,
lectura de versión anterior, backfill verificable y fecha de retiro. Definir
pruebas con datos ficticios: versión vieja/nueva, contexto/AAD equivocado,
ciphertext alterado, nonce y fallo cerrado. No migrar tokens reales por la revisión.

Para state OAuth firmado, revisar TTL/replay y efecto de retirar la clave con
callbacks pendientes. Para secreto interno, revisar servicios que todavía lo
transportan. Los procedimientos concretos se basan en el proveedor/config real,
sin introducir Vault o credenciales dinámicas donde no hay soporte verificado.

## Detección y prevención a elegir

Separar working tree, staged diff, historia y artefactos; una detección actual
no acredita revisar las cuatro superficies. Usar comandos redactados y checks
de bundle/arquitectura existentes. Allowlist solo valores públicos/fixtures
específicos con razón, no todo archivo que disparó una alerta. Un hook de commit
puede omitirse; un gate CI futuro necesita cobertura y evidencia propias.
El procedimiento está en la referencia operativa de `sast-configuration`.
