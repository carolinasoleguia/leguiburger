# Guía de Pruebas: Endpoints de Empleados
Esta sección detalla cómo probar la API de empleados utilizando tokens de autenticación JWT.

## Requisito Previo: Obtener el Token JWT
Antes de realizar peticiones a los endpoints protegidos, debés autenticarte en el endpoint /api/auth/login para recibir un Bearer Token.

```text
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92" \
  -d '{
    "email": "admin@email.com",
    "password": "PasswordSegura123!"
  }'
```

Nota: Si ingresás como OWNER, el header X-Tenant-ID es opcional en el login. Para el resto de los roles (admin, cashier, etc.), el header X-Tenant-ID debe coincidir con el tenant asignado.

## 1. Crear un Nuevo Empleado
Registra un empleado para un comercio específico. Requiere token de autenticación (ADMIN o OWNER). Responde con 201 Created.

```text
curl -X POST http://localhost:8080/api/v1/tenants/202bc473-c221-469c-9fb9-a03b45259d92/employees \
  -H "Authorization: Bearer TU_JWT_TOKEN_AQUI" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Ana",
    "last_name": "Eguia",
    "email": "ana@email.com",
    "password": "PasswordSegura123!",
    "phone": "2215555555",
    "role": "admin"
  }'
```

## 2. Listar Todos los Empleados de un Tenant
Devuelve la lista completa de empleados correspondientes al comercio.

```text
curl --location 'http://localhost:8080/api/employees' \
  --header 'Authorization: Bearer TU_JWT_TOKEN_AQUI' \
  --header 'Content-Type: application/json' \
  --header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 3. Obtener un Empleado por ID

```text
curl --location 'http://localhost:8080/api/employees/ef6c9545-7792-4713-a1a4-6819decd06f5' \
  --header 'Authorization: Bearer TU_JWT_TOKEN_AQUI' \
  --header 'Content-Type: application/json' \
  --header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 4. Editar un Empleado
Permite actualizar información como rol, teléfono o estado de activación.

```text
curl --location --request PUT 'http://localhost:8080/api/employees/ef6c9545-7792-4713-a1a4-6819decd06f5' \
  --header 'Authorization: Bearer TU_JWT_TOKEN_AQUI' \
  --header 'Content-Type: application/json' \
  --header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92' \
  --data '{
    "role": "cashier",
    "phone": "2219999999",
    "is_active": true
  }'
```

## 5. Borrar un Empleado
Elimina por completo el registro de la base de datos y responde con 204 No Content si es exitoso.

```text
curl --location --request DELETE 'http://localhost:8080/api/employees/ef6c9545-7792-4713-a1a4-6819decd06f5' \
  --header 'Authorization: Bearer TU_JWT_TOKEN_AQUI' \
  --header 'Content-Type: application/json' \
  --header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```