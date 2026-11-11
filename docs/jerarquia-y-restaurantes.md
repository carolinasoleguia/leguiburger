# Jerarquía de cuentas y restaurantes

La plataforma usa `users.owner` y `users.super_admin`. El dueño de un restaurante
usa `users.admin`, asociado a un negocio (`brands`). Los colaboradores siguen en
`employees`, con funciones `employee`, `cashier` o `kitchen`; nunca reciben rol admin.
Los nombres en las pantallas son Plataforma LeguiBurger, Mi restaurante, Equipo y Locales.

## Alta e invitaciones

`POST /api/platform/restaurants` crea negocio, primer local e invitación al dueño
en una transacción. Requiere `Idempotency-Key`; un reintento conserva la misma
respuesta. La plataforma indica nombre, correo, identificador del local y cupos
base; no crea ni conoce la contraseña del dueño.

La invitación vence en 48 horas, se guarda como hash y puede compartirse manualmente
mientras SMTP no esté configurado. El correo queda en la outbox cifrada.
`POST /api/auth/business-invitations/accept` valida la capacidad de un solo uso,
crea al administrador y lo establece como responsable principal. Para una cuenta
existente del mismo negocio exige su contraseña actual y la conserva. No permite
tomar una identidad de otra marca, de plataforma o de un colaborador.

`GET /api/platform/restaurants` lista negocios para plataforma y `GET /api/business`
devuelve los datos del negocio propio a su administrador. Los registros previos
se conservan sin adivinar un responsable. Si no está designado, la plataforma
puede invitar al administrador existente para que lo confirme con su contraseña.

## Responsabilidad y suspensión

El responsable principal o la plataforma pueden generar
`POST /api/business/{id}/owner-invitation`. Un responsable actual debe confirmar
su contraseña. Al aceptar el destinatario, se cambia el responsable y se desactiva
al anterior, revocando sus sesiones. Las operaciones, el equipo y el historial
siguen perteneciendo al mismo negocio. La DB impide quitar o desactivar al principal
antes de transferirlo. Una nueva invitación revoca la anterior; también existe
`DELETE /api/business/{id}/owner-invitation` para cancelarla explícitamente.

`PATCH /api/platform/restaurants/{id}/status` guarda suspensión, motivo y cupos base.
Suspender revoca sesiones de colaboradores, seguimientos públicos e invitaciones
pendientes. El dueño puede ingresar a su cuenta y consultar membresía/seguridad,
pero las APIs operativas rechazan sus solicitudes. Reactivar no revive sesiones
revocadas: el equipo debe volver a iniciar sesión.

La suspensión impide nuevos checkouts comerciales y autorizaciones OAuth. Los
webhooks y workers continúan verificando dinero que el proveedor ya recibió.
No cancela automáticamente preferencias de pago ya emitidas por Mercado Pago:
un pago posterior por un checkout anterior se concilia y queda registrado.
Pagar una membresía no levanta una suspensión administrativa automáticamente.

## Locales y colaboradores

- `GET/POST /api/business/locations`: locales propios y alta dentro del cupo.
- `PATCH /api/business/locations/{id}`: nombre visible, identificador, dirección,
  teléfono, horarios y estado; nunca cambia la pertenencia a otro negocio.
- `GET /api/business/team`: equipo propio, roles y locales asignados.
- `PATCH /api/business/team/{id}`: nombre, función, estado, local principal y lista
  de sucursales del mismo negocio. No elimina la identidad ni su historial.
- `POST /api/team/invitations`: invita a una cuenta nueva o agrega un local a una
  cuenta existente del mismo negocio y función. El destinatario existente confirma
  con su contraseña. Cambiar la función requiere la gestión explícita del equipo.
- `POST /api/auth/location`: cambia el contexto de trabajo de un colaborador sólo
  a una sucursal activa que tenga asignada. Crea sesión/token nuevos y revoca los
  anteriores. El navegador coordina el cambio de cookie y contexto entre pestañas.

La sesión resuelve el local activo desde asignaciones verificadas en DB. Un header
arbitrario no cambia el contexto. Quitar una asignación invalida sus sesiones y
las escrituras vuelven a comprobar la pertenencia dentro de la transacción.
El rol del colaborador es el mismo en todas sus sucursales asignadas.

`employee_locations` registra asignaciones vigentes; `employee_location_history`
conserva pares históricos inmutables. Ventas/stock/eventos mantienen la atribución
al lugar donde ocurrieron aunque se retire la asignación o cambie el local principal.
Los inserts de eventos comprueban además la asignación vigente del actor.

## Cupos

Cada negocio empieza con cupos base configurables por plataforma, predeterminados
de un local y cinco colaboradores. Un contrato vigente de producción reemplaza esos
cupos con los límites de su precio; `NULL` significa sin límite. Un contrato por
negocio tiene prioridad. Si sólo hay contratos por local, se usa el más reciente
vigente como política de capacidad del negocio; no se suman cupos entre contratos.
Los contratos sandbox no modifican la capacidad real.

La plataforma puede publicar precios con límites explícitos desde Cobros. Cambiar
el precio crea una versión y conserva los contratos anteriores. Al vencer un contrato,
se vuelve al acceso base: no se borran ni desactivan recursos existentes que excedan
el límite, pero no se permite ampliar capacidad hasta resolver el exceso.

Los triggers de DB serializan el conteo de capacidad por negocio y controlan alta
o reactivación de locales/colaboradores, además de reservas de invitaciones. Una
persona con varias asignaciones cuenta una vez. Las invitaciones nuevas pendientes
reservan cupo por correo; vencidas, canceladas o ya aceptadas no lo consumen otra vez.
La comprobación vuelve a ejecutarse al aceptar una invitación.

## Migraciones y pruebas

Aplicadas `202611110004_business_hierarchy`, `202611110005_multilocation_history`
y `202611110006_contracted_business_capacity`, con RLS y grants privados revocados,
ensayo/verificación del proyecto y conservación de los datos originales.

```powershell
go test ./...
go test ./internal/operations -run TestBusinessHierarchyIntegration -count=1 -v -args -security-integration
```

La integración HTTP usa una transacción externa y revierte todo: alta/replay,
invitaciones, cupos/reservas, transferencia, contextos, escrituras y auditoría en
segunda sucursal, revocación, aislamiento y excepción de membresía bajo suspensión.
`tools/test-hierarchy-browser.mjs` comprueba los formularios y navegación en 1440/390,
con servidor aislado y API simulada; también se ejecuta en CI.

El acceso sigue siendo con usuario y contraseña. MFA permanece desactivado por
configuración, como se solicitó; los permisos y las sesiones revocables siguen activos.
