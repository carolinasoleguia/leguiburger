# LeguiBurger 🍔

LeguiBurger organiza marcas, sucursales, catálogo, equipo, producción y operación de hamburgueserías. El frontend es Vue con Vite; el backend Go usa PostgreSQL en Supabase.

- `main.go`: servidor Go y registro de rutas.
- `internal/`: autenticación y módulos de negocio.
- `internal/operations`: pedidos, cobros, inventario, gastos, membresías y analítica.
- `frontend/`: aplicación Vue.
- `database/migrations/`: migraciones incrementales versionadas.
- `cmd/migrate/`: ensayo, aplicación y verificación de migraciones.
- `skills/`: biblioteca de Al Toque y `unslop`, con [adaptación a LeguiBurger](skills/leguiburger-context.md).

## Desarrollo local

Usá la versión de Go indicada en `go.mod` y una versión de Node compatible con las dependencias de `frontend/package.json`. Configurá `.env` en la raíz a partir de `.env.example`. `DATABASE_URL` y `JWT_SECRET` son necesarios; la configuración de Supabase Storage se usa para imágenes del catálogo.

Desde la raíz:

```powershell
go run main.go
```

En otra terminal:

```powershell
npm --prefix frontend install
npm --prefix frontend run dev
```

La interfaz queda normalmente en `http://127.0.0.1:5173/` y Vite reenvía `/api` a `http://localhost:8080`. El backend comprueba la identidad y las migraciones de la base antes de iniciar. Para servir la interfaz desde Go, generá `frontend/dist` con `npm --prefix frontend run build`.

La [organización por perfil](docs/experiencias-por-perfil.md) separa plataforma, restaurante, comprador con cuenta e invitado, conservando componentes visuales y el motor de pedidos compartidos.

## Base de datos

La guía de [migraciones](database/README.md) explica el ensayo con rollback, la verificación de datos y la aplicación en el proyecto correcto. `resources/migration.slq` es un esquema histórico con comandos destructivos; no se usa para actualizar la base existente.

El seed de un administrador inicial requiere `SEED_OWNER_EMAIL` y `SEED_OWNER_PASSWORD` definidos explícitamente en el entorno. No se ejecuta al iniciar el backend ni al migrar.

Las [cuentas de compradores](docs/cuentas-compradores.md) permiten historial entre marcas,
seguimiento autenticado, preferencias y cupones emitidos por cada restaurante.

## API y pruebas

La guía de [soporte y continuidad](docs/soporte-y-continuidad.md) documenta soporte con historial, respaldos cifrados y verificados, cierre de restaurante/cuenta, imágenes seguras, monitoreo y consistencia de pantallas.

La [jerarquía de restaurantes](docs/jerarquia-y-restaurantes.md) cubre alta unificada,
invitación y transferencia del dueño, sucursales, equipo con varios locales, cupos
de planes y suspensión administrativa con acceso a membresía.

El [espacio operativo](docs/espacio-operativo.md) documenta dashboard real, caja,
cocina, stock/costos, gastos, clientes, seguimiento privado, notificaciones,
onboarding y procesos de segundo plano. El ingreso actual usa usuario y contraseña;
el segundo factor queda desactivado por configuración.

La [integración de Mercado Pago](docs/mercadopago.md) documenta OAuth, cobros de
pedidos, membresías, devoluciones, webhooks, conciliación y configuración.

La guía de [seguridad y sesiones](docs/seguridad-y-sesiones.md) explica permisos,
MFA, recuperación, invitaciones, auditoría y configuración del correo SMTP.

La guía de [carta pública y turnos de caja](docs/menu-y-caja.md) explica la publicación
del catálogo existente, QR, búsqueda completa y arqueos. La [revisión de aislamiento
de la API](docs/revision-aislamiento-api.md) documenta los límites entre plataforma,
restaurantes y colaboradores, con pruebas de IDs y ámbitos forzados.

La [experiencia de compradores](docs/experiencia-compradores.md) agrega carrito, extras,
entrega, confirmación sin registro, seguimiento y pago de pedidos aceptados sobre
la misma carta pública.

La [API de operación y analítica](docs/api-operaciones.md) documenta roles, endpoints, DTOs y ejemplos. El [diagnóstico de analítica](docs/analitica-dashboards.md) conserva la comparación con Al Toque y el inventario previo a la migración.

```powershell
go test ./...
go test ./internal/operations -run TestDatabaseIntegration -count=1 -v -args -db-integration
```

La prueba de integración es explícita, verifica el proyecto esperado y revierte todos los datos temporales.
