## 1. Crear un Producto de Catálogo de Marca
Este comando registra un producto a nivel marca. El stock, disponibilidad y precio por tienda se configuran en tenant_products.

```text
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "name": "Doble Cheddar",
    "description": "Burger con doble cheddar",
    "base_price": 4500.00,
    "image_url": "https://example.com/burger.jpg"
  }'
```

Si el usuario autenticado es owner, puede enviar la marca de forma explícita:

```text
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -H "X-Brand-ID: BRAND_ID" \
  -d '{
    "name": "Doble Cheddar",
    "description": "Burger con doble cheddar",
    "base_price": 4500.00
  }'
```

## 2. Listar Productos de la Marca

```text
curl --location 'http://localhost:8080/api/products' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN'
```

## 3. Obtener un Producto por ID

```text
curl --location 'http://localhost:8080/api/products/PRODUCT_ID' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN'
```

## 4. Editar un Producto de Catálogo

```text
curl --location --request PUT 'http://localhost:8080/api/products/PRODUCT_ID' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN' \
--data '{
    "base_price": 4900.00,
    "is_active": true
  }'
```

## 5. Borrar un Producto

```text
curl --location --request DELETE 'http://localhost:8080/api/products/PRODUCT_ID' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN'
```
