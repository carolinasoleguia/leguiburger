## 1. Asignar un Producto a una Tienda
Este comando habilita un producto del catálogo de marca para un tenant específico.

```text
curl -X POST http://localhost:8080/api/tenant-products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -H "X-Tenant-ID: TENANT_ID" \
  -d '{
    "product_id": "PRODUCT_ID",
    "price_override": 4800.00,
    "current_stock": 20,
    "track_stock": true,
    "is_available": true
  }'
```

## 2. Listar Productos Disponibles/Configurados Para una Tienda

```text
curl --location 'http://localhost:8080/api/tenant-products' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN' \
--header 'X-Tenant-ID: TENANT_ID'
```

## 3. Obtener Configuración de un Producto en una Tienda

```text
curl --location 'http://localhost:8080/api/tenant-products/PRODUCT_ID' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN' \
--header 'X-Tenant-ID: TENANT_ID'
```

## 4. Editar Configuración Local del Producto

```text
curl --location --request PUT 'http://localhost:8080/api/tenant-products/PRODUCT_ID' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN' \
--header 'X-Tenant-ID: TENANT_ID' \
--data '{
    "price_override": 5000.00,
    "current_stock": 12,
    "is_available": true,
    "is_active": true
  }'
```

## 5. Quitar un Producto de una Tienda

```text
curl --location --request DELETE 'http://localhost:8080/api/tenant-products/PRODUCT_ID' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer TOKEN' \
--header 'X-Tenant-ID: TENANT_ID'
```
