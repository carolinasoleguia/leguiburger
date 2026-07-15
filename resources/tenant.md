## 1. Crear un Nuevo Tenant (Caso Exitoso)
Este comando registra un comercio normalizando el subdominio y respondiendo con un 201 Created.


```text
curl -X POST http://localhost:8080/api/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "LeguiBurger Centro",
    "subdomain": "leguiburger-centro"
  }'
```


## 2. Editar el nombre de Poggs (ID de ejemplo)

```text
curl -X PUT http://localhost:8080/api/tenants/763fe534-2df7-4770-bdf8-77db9af11d8a \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Poggs City Bell Premium"
  }'
```

## 3. Borrar tenant (Pasa el estado active a false)
```text
curl --location --request DELETE 'http://localhost:8080/api/tenants/763fe534-2df7-4770-bdf8-77db9af11d8a' \
--header 'Content-Type: application/json' \
--data '{
    "name": "Poggs City Bell Premium"
  }'
```