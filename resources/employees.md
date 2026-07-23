## 1. Crear un Nuevo Empleado
Este comando registra un empleado para un comercio específico y responde con un 201 Created.

```text
curl -X POST http://localhost:8080/api/employees \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92" \
  -d '{
    "first_name": "Ana",
    "last_name": "Eguia",
    "email": "ana@email.com",
    "password_hash": "hash_previamente_generado",
    "phone": "2215555555",
    "role": "admin"
  }'
```

## 2. Listar Todos los Empleados de un Tenant

```text
curl --location 'http://localhost:8080/api/employees' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 3. Obtener un Empleado por ID

```text
curl --location 'http://localhost:8080/api/employees/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 4. Editar un Empleado

```text
curl --location --request PUT 'http://localhost:8080/api/employees/ef6c9545-7792-4713-a1a4-6819decd06f5' \
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
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```
