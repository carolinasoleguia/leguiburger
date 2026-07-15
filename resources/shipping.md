## 1. Crear un Nuevo Método de Envío (Caso Exitoso)
Este comando registra un método de envío para un comercio específico y responde con un 201 Created.

```text
curl -X POST http://localhost:8080/api/shipping-methods \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a" \
  -d '{
    "name": "Envío Moto Express",
    "description": "Entrega en menos de 45 minutos",
    "typification": "DELIVERY",
    "cost": 1500.00,
    "estimated_time": "30-45 min"
  }'
```
## 2. Listar Todos los Métodos de Envío de un Tenant
Obtiene la lista completa de los métodos de envío configurados para el comercio especificado en el header.

```text
curl -X GET http://localhost:8080/api/shipping-methods \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a"
```

## 3. Obtener un Método de Envío Específico por ID
Recupera el detalle de un único método de envío mediante su ID.

```text
curl -X GET http://localhost:8080/api/shipping-methods/efc2c513-cde1-48ab-a8c1-e0f563268c29 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a"
```

## 4. Editar un Método de Envío (Ejemplo: Cambiar Costo y Estado)
Actualiza los campos modificados de un método de envío existente.

```text
curl -X PUT http://localhost:8080/api/shipping-methods/efc2c513-cde1-48ab-a8c1-e0f563268c29 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a" \
  -d '{
    "cost": 1800.00,
    "is_active": false
  }'
```

## 5. Borrar un Método de Envío (Eliminación Física)
Elimina por completo el registro de la base de datos (responde con un 244 No Content si es exitoso).

```text
curl -X DELETE http://localhost:8080/api/shipping-methods/efc2c513-cde1-48ab-a8c1-e0f563268c29 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 763fe534-2df7-4770-bdf8-77db9af11d8a"
```