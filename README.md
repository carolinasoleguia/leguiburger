# LeguiBurger SaaS 🍔

¡Bienvenido a **LeguiBurger**! Un software como servicio (SaaS) moderno y de alto rendimiento diseñado para la gestión integral y el autopedido en comercios gastronómicos. 

A diferencia de las cartas QR digitales convencionales, LeguiBurger está concebido como un ERP híbrido con aislamiento multi-tenant estricto, cálculo automático del costo real de producción (BOM), control de stock inteligente basado en recetas y soporte para analíticas avanzadas.

---

## 🛠️ Stack Tecnológico

El proyecto utiliza una arquitectura desacoplada moderna de alto rendimiento:

* **Backend:** [Go (Golang)](https://go.dev/) utilizando el framework de enrutamiento estándar, aislamiento por capas (Handler -> Service -> Repository) y [GORM](https://gorm.io/) como ORM para la interacción de datos.
* **Base de Datos:** [PostgreSQL](https://www.postgresql.org/) alojado en la nube mediante [Supabase](https://supabase.com/), con soporte nativo de UUIDs para mitigar riesgos de seguridad de ID secuenciales y asegurar el aislamiento multi-tenant.
* **Frontend:** [Vue.js](https://vuejs.org/) para una interfaz reactiva, rápida y ligera pensada para dispositivos móviles.

---

## 📁 Estructura del Proyecto

El backend sigue las mejores prácticas de la comunidad de Go para mantener el código desacoplado, testeable y mantenible:

```text
leguiburger/
├── cmd/
│   └── api/
│       └── main.go           # Punto de entrada de la aplicación
├── internal/
│   ├── db/
│   │   └── db.go             # Inicialización de la conexión de GORM
│   ├── models/
│   │   └── tenant.go         # Modelos de datos compartidos (Structs de GORM)
│   └── tenants/              # Módulo de ejemplo (Estructura por capas)
│       ├── handler.go        # Capa de Presentación (Controladores HTTP)
│       ├── handler_test.go   # Tests de Integración de Endpoints HTTP
│       ├── repository.go     # Capa de Datos (Acceso directo a DB)
│       ├── service.go        # Capa de Negocio (Reglas del SaaS)
│       └── service_test.go   # Tests Unitarios de Lógica de Negocio
├── frontend/                 # Código fuente del cliente en Vue.js
├── .env                      # Variables de entorno locales (NO subir a GitHub)
├── go.mod                    # Dependencias de Go
└── README.md

```

## 🚀 Guía de Desarrollo Local
📋 Prerrequisitos
Asegurate de tener instalado en tu sistema operativo:

* Go 1.20+

* Node.js 18+ (para el Frontend)

### 1. Configuración del Entorno (.env)

Crea un archivo .env en la raíz del proyecto con las siguientes variables:

```text
PORT=8080
DATABASE_URL="postgres://postgres.[TU_ID_PROYECTO]:[TU_CONTRASEÑA]@aws-0-sa-east-1.pooler.supabase.com:6543/postgres?sslmode=require"
```

### 2. Levantar el Backend (Go)
Abrí tu terminal en la raíz del proyecto.

Descargá y sincronizá las dependencias necesarias de Go:

```text
go mod tidy
```

Ejecutá el servidor de desarrollo:

```text
go run main.go
```

El servidor backend levantará exitosamente en http://localhost:8080.

### 3. Levantar el Frontend (Vue.js)
Abrí una nueva pestaña de la terminal y navega al directorio del frontend:

```text
cd frontend
npm install
npm run dev
```

La aplicación web se abrirá localmente (por lo general en http://localhost:5173).

## 🧪 Ejecución de Pruebas (Tests)
El sistema cuenta con una sólida cobertura de pruebas tanto para la lógica de negocio como para las respuestas de la API, sin requerir conexión activa a internet ni bases de datos externas mediante el uso de interfaces y mocks.

Para ejecutar las pruebas del proyecto, utiliza la terminal en la raíz de la aplicación:

Ejecutar todos los tests del proyecto:

```text
go test -v -cover ./...
```

Backend: Configurado y optimizado para ejecutarse en Render. Al hacer un git push a la rama principal, el pipeline compila el archivo cmd/api/main.go automáticamente.
Base de datos: Sincronizada directamente en la nube mediante Supabase (PostgreSQL).
