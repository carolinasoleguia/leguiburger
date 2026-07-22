## 1. Crear un Nuevo Cliente
Este comando registra un cliente para un comercio específico y responde con un 201 Created.

```text
curl -X POST http://localhost:8080/api/customers \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a" \
  -d '{
    "first_name": "Juan",
    "last_name": "Perez",
    "email": "juan@email.com",
    "phone": "2215555555"
  }'
```

## 2. Listar Todos los Clientes de un Tenant

```text
curl --location 'http://localhost:8080/api/customers' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a'
```

## 3. Obtener un Cliente por ID

```text
curl --location 'http://localhost:8080/api/customers/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a'
```

## 4. Editar un Cliente

```text
curl --location --request PUT 'http://localhost:8080/api/customers/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a' \
--data '{
    "first_name": "Juana",
    "email": "juana@email.com"
  }'
```

## 5. Borrar un Cliente
Elimina por completo el registro de la base de datos y responde con 204 No Content si es exitoso.

```text
curl --location --request DELETE 'http://localhost:8080/api/customers/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a'
```
