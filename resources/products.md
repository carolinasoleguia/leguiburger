## 1. Crear un Nuevo Producto
Este comando registra un producto para un comercio específico y responde con un 201 Created.

```text
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92" \
  -d '{
    "name": "Doble Cheddar",
    "description": "Burger con doble cheddar",
    "current_price": 4500.00,
    "current_stock": 20,
    "track_stock": true,
    "image_url": "https://example.com/burger.jpg"
  }'
```

## 2. Listar Todos los Productos de un Tenant

```text
curl --location 'http://localhost:8080/api/products' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 3. Obtener un Producto por ID

```text
curl --location 'http://localhost:8080/api/products/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 4. Editar un Producto

```text
curl --location --request PUT 'http://localhost:8080/api/products/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92' \
--data '{
    "current_price": 4900.00,
    "current_stock": 18,
    "is_active": true
  }'
```

## 5. Borrar un Producto
Elimina por completo el registro de la base de datos y responde con 204 No Content si es exitoso.

```text
curl --location --request DELETE 'http://localhost:8080/api/products/ef6c9545-7792-4713-a1a4-6819decd06f5' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```
