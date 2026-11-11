# Seguridad, sesiones y equipos

Implementado sobre Vue y Go, con estado privado en Supabase. La migración
`202611100007_identity_sessions_and_team` conserva los datos anteriores. Los JWT
anteriores, sin sesión persistida, ya no autorizan peticiones: hay que volver a entrar.

## Autenticación

- Access JWT HS256 de 15 minutos, con issuer, audience, subject, session ID y propósito.
- Access token únicamente en memoria del navegador. Refresh aleatorio de 256 bits
  en cookie HttpOnly, SameSite=Strict, Secure cuando se usa HTTPS/producción.
- Refresh rotativo de un solo uso. Reutilizar uno consumido revoca toda su sesión.
- Sesión máxima de 7 días y vencimiento tras 30 minutos sin actividad. El frontend
  avisa del cierre, revoca la sesión y coordina renovación/cierre entre pestañas.
- Cada petición verifica cuenta activa, rol, asignación, credenciales y sesión en DB.
  Cambiar email, contraseña, rol, local, marca o estado revoca las sesiones mediante
  triggers. Las escrituras vuelven a bloquear y comprobar la sesión.
- La cuenta de plataforma debe completar TOTP antes de recibir acceso o refresh.
  Alta con clave manual compatible con autenticadores, tolerancia de un paso de
  30 segundos y protección contra reutilización. Diez códigos de recuperación de
  un solo uso se muestran al completar el alta. Deben guardarse fuera de la app.
- Recuperar contraseña no elimina el segundo factor. No hay bypass administrativo
  público para desactivar MFA. Si se pierden autenticador y todos los códigos, hace
  falta un procedimiento operativo de verificación de identidad fuera de la app.

## Permisos

`auth.AuthorizeLegacy` contiene la lista explícita de acciones de la API original.
Una ruta nueva no hereda permisos. `internal/operations` conserva controles propios
de rol, comercio, recurso e idempotencia, usando las mismas sesiones verificadas.

| Área | Plataforma (`owner`/`super_admin` en users) | Marca (`admin` en users) | Colaborador |
|---|---|---|---|
| Marcas, administradores, alta/baja/cambio de locales | Administración | Sin escritura | Sin escritura |
| Listar locales | Todos | Su marca | Su local |
| Equipo e invitaciones | Administración | Su marca/locales | Sin acceso |
| Catálogo/configuración | Administración | Su marca/locales | Lectura dentro de su asignación |
| Clientes | Administración | Su marca/locales | Caja, en su local |
| Producción | Administración | Su marca/locales | Cocina/colaborador en su local, sin borrar |
| Pedidos y pagos | Administración | Su marca/locales | Caja, en su local |
| Analítica financiera y gastos | Administración | Su marca/locales | Sin acceso |
| Analítica operativa/inventario de consulta | Administración | Su marca/locales | Su local |
| Membresías/analítica de plataforma | Administración | Sin acceso | Sin acceso |
| Auditoría | Toda la plataforma | Su marca | Sin acceso |
| Sesiones/contraseña | Su propia cuenta | Su propia cuenta | Su propia cuenta |

Los empleados nuevos solo pueden tener `employee`, `cashier` o `kitchen`; hay una
restricción adicional en DB. Los roles superiores se administran en users. El
comercio/marca enviado por header, query o cuerpo no concede permisos. Los IDs se
consultan junto con el ámbito autorizado. Las mutaciones originales comparten una
transacción que guarda actor, cambios y auditoría; nunca se registra el cuerpo con
contraseñas, tokens o secretos.

## API y pantallas

- `/login`: contraseña, desafío MFA y entrega inicial de códigos de recuperación.
- `/forgot-password`, `/reset-password`: recuperación por enlace de 30 minutos.
- `/accept-invitation`: acepta una invitación de 48 horas y elige contraseña.
- `/account/security`: sesiones, revocación individual/global y cambio de contraseña.
- `/team/invitations`: crear, consultar y revocar invitaciones para locales propios.
- `/audit`: actividad de negocio y seguridad con paginación y actor.

Rutas públicas POST: `/api/auth/login`, `/refresh`, `/logout`, `/password/forgot`,
`/password/reset`, `/mfa/enroll`, `/mfa/verify`, `/invitations/accept`, todas bajo
`/api/auth`. El desafío MFA es un bearer restringido que no sirve para la API de negocio.
Rutas autenticadas: GET `/api/auth/me`, GET `/api/auth/sessions`, DELETE
`/api/auth/sessions/{id}`, POST `/api/auth/logout-all`, POST `/api/auth/password/change`.
Invitaciones: GET/POST `/api/team/invitations`, DELETE `/api/team/invitations/{id}`.
Auditoría: GET `/api/audit?limit=50&offset=0`, opcional `tenant_id` autorizado.

Los tokens de acción son aleatorios, se guarda solamente su SHA256 y se consumen en
transacción. El enlace usa fragmento URL; la pantalla lo retira del historial. Las
respuestas de recuperación no distinguen si existe la cuenta. Los límites de login,
MFA, recuperación e invitaciones persisten en DB y devuelven 429 al excederse. La IP
se toma de la conexión TCP: un despliegue con proxy debe dimensionar el límite global
considerando que no se confía en encabezados IP enviados por el cliente.

## Correo y despliegue

`PUBLIC_SITE_URL` fija el origen permitido y los enlaces. En producción se exige
HTTPS y `APP_ENV=production`. `JWT_SECRET` debe ser largo, aleatorio y privado.
`AUTH_ENCRYPTION_KEY` permite separar la clave de cifrado de MFA/correo de la firma
JWT; hay que respaldarla y mantenerla estable. Cambiar la clave efectiva requiere
migrar los secretos cifrados, no reemplazarla sin planificación.

Para envío real configurar `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USERNAME`
y `SMTP_PASSWORD` según el proveedor. Se exige TLS (587/STARTTLS o 465/TLS).
El worker procesa la cola cada 15 segundos, reintenta hasta 8 veces y borra el cuerpo
cifrado al entregar o vencer. Sin SMTP el mensaje queda pendiente; no se imprime el
enlace de recuperación. Las invitaciones también entregan al gestor un enlace para
compartir directamente. En el entorno local actual SMTP no está configurado.

Las tablas privadas tienen RLS y no conceden acceso a anon, authenticated ni
service_role. Se verificó que users, employees, customers, tenants, products,
supplies y orders tampoco conceden lectura/actualización a anon. El backend usa
su conexión PostgreSQL, por lo que sigue necesitando los controles de aplicación.
Las consultas del logger de producción ocultan parámetros.

## Validación

```powershell
go test ./...
go test ./internal/operations -run TestSecurityIntegration -count=1 -v -args -security-integration
go test ./internal/operations -run TestDatabaseIntegration -count=1 -v -args -db-integration
npm --prefix frontend run build
```

Ambas integraciones remotas trabajan dentro de una transacción exterior que se
revierte: no dejan cuentas, contraseñas, sesiones, invitaciones ni correos de prueba.
La prueba de seguridad usa los handlers y repositorios reales para comprobar
aislamiento, escalamiento bloqueado, MFA/replay, rotación/reutilización, revocación,
recuperación, invitaciones, auditoría, cookies y límites.

`tools/test-security-browser.mjs` comprueba el build con respuestas HTTP simuladas,
sin cuentas reales. Requiere `PUPPETEER_MODULE` y `CHROME_PATH` apuntando a una
instalación local. Verifica los flujos y anchos 1440/390, incluido cierre por inactividad.

Referencias: [TOTP RFC 6238](https://www.rfc-editor.org/rfc/rfc6238),
[sesiones OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html),
[recuperación OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html).
