## 1. Crear una Nueva Receta
Este comando vincula un producto con un insumo y define la cantidad usada.

```text
curl -X POST http://localhost:8080/api/recipes \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92" \
  -d '{
    "product_id": "PRODUCT_ID",
    "supply_id": "SUPPLY_ID",
    "quantity_used": 0.250
  }'
```

## 2. Listar Todas las Recetas de un Tenant

```text
curl --location 'http://localhost:8080/api/recipes' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 3. Obtener una Receta por Producto e Insumo

```text
curl --location 'http://localhost:8080/api/recipes/PRODUCT_ID/SUPPLY_ID' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```

## 4. Editar una Receta

```text
curl --location --request PUT 'http://localhost:8080/api/recipes/PRODUCT_ID/SUPPLY_ID' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92' \
--data '{
    "quantity_used": 0.300
  }'
```

## 5. Borrar una Receta

```text
curl --location --request DELETE 'http://localhost:8080/api/recipes/PRODUCT_ID/SUPPLY_ID' \
--header 'Content-Type: application/json' \
--header 'X-Tenant-ID: 202bc473-c221-469c-9fb9-a03b45259d92'
```
