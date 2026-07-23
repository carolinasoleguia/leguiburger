## 1. Crear un Nuevo Extra
Este comando registra un extra para un comercio específico y responde con un 201 Created.

```text
curl -X POST http://localhost:8080/api/extras \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92" \
  -d '{
    "name": "Cheddar",
    "current_price": 250.00,
    "current_stock": 10,
    "track_stock": true
  }'
```

## 2. Listar Todos los Extras de un Tenant

```text
curl --location 'http://localhost:8080/api/extras' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 3. Obtener un Extra por ID

```text
curl --location 'http://localhost:8080/api/extras/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 4. Editar un Extra

```text
curl --location --request PUT 'http://localhost:8080/api/extras/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92' \
--data '{
    "current_price": 300.00,
    "current_stock": 12,
    "is_active": true
  }'
```

## 5. Borrar un Extra
Elimina por completo el registro de la base de datos y responde con 204 No Content si es exitoso.

```text
curl --location --request DELETE 'http://localhost:8080/api/extras/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```
