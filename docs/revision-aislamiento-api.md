# Revisión de aislamiento de la API — 12/11/2026

Política confirmada por el usuario:

- `owner`/`super_admin` de plataforma puede consultar y administrar los negocios.
- `admin` solo accede a su marca y sus locales.
- Los colaboradores solo acceden al local vigente de su sesión, dentro de sus
  asignaciones activas, y a las operaciones de su rol. Cambiar a otro local
  asignado requiere cambiar el contexto de sesión; forzar un header no lo cambia.
- Publicar una carta crea una proyección pública explícita, no acceso al catálogo
  interno. La publicación está desactivada por defecto.

## Evidencia

`internal/operations/isolation_integration_test.go` agrega 147 comprobaciones
HTTP sobre handlers, middleware y repositorios reales. Usa dos administradores
de marcas distintas, tres locales, plataforma, caja y cocina. Se crean datos
temporales y se revierte la transacción completa.

Se comprueban headers y queries de marca/local ajenos, IDs reales de productos,
asignaciones locales, insumos y recetas de la otra marca; lecturas y escrituras
bloqueadas; rechazo de catálogos y recursos financieros fuera de ámbito; denegación
de endpoints de plataforma a admins y colaboradores; y casos legítimos de cada rol.
La API no devuelve los datos del recurso en las respuestas denegadas y el producto
ajeno conserva su nombre tras los intentos de modificación y borrado.

Las nuevas rutas de publicación y caja se prueban también con IDs de otro local,
sin sesión, con rol insuficiente y con campos de autoridad/saldo no aceptados.
La vista pública se comprueba sin sesión, retirada, publicada y con negocio
suspendido, incluyendo ausencia de campos internos. Se verifica además búsqueda
parametrizada, apertura única, reintentos y cierre calculado desde los movimientos.

Las integraciones existentes de seguridad, jerarquía y operaciones cubren
sesiones revocadas, cambios de credenciales, invitaciones, transferencias,
colaboradores con varios locales, pedidos, pagos y auditoría. La integración de
Mercado Pago utiliza un proveedor simulado; no acredita cobros reales ni la
configuración del proveedor en producción.

## Resultado y corrección

No se observó acceso cruzado en los escenarios ejecutados. El acceso global de
plataforma se conservó intencionalmente. La revisión detectó una incompatibilidad
de esquema: el check antiguo `chk_employee_role` impedía crear colaboradores de
cocina. La migración 010 lo elimina solo si existe el control más estricto
`employees_collaborator_roles`; no habilita roles administrativos en employees.
Los roles históricos incompatibles siguen sin poder iniciar una sesión válida.

Los nuevos datos de caja viven en tablas privadas con RLS y grants revocados a
anon, authenticated y service_role. El backend usa su conexión confiable, por lo
que la autorización de aplicación sigue siendo necesaria y se prueba por separado.
La carta pública selecciona campos explícitos; no reutiliza respuestas privadas
ni devuelve `SELECT *` de productos o locales.

Esta revisión es evidencia de los escenarios enumerados, no una garantía de
ausencia de vulnerabilidades en toda la API ni una prueba del despliegue público.
Las imágenes del catálogo usan URLs públicas de Storage por diseño; una imagen
compartida no es una autorización para leer el catálogo interno.

```powershell
go test ./...
go vet ./...
go test ./internal/operations -count=1 -v -args -db-integration -security-integration
go test ./internal/mercadopago -run TestDatabaseIntegration -count=1 -v -args -mp-db-integration
```
